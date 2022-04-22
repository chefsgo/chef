package chef

import (
	"errors"
	"sync"
	"time"

	. "github.com/chefsgo/base"
)

var (
	mView = newViewModule()
)

func newViewModule() *viewModule {
	return &viewModule{
		config: ViewConfig{
			Driver: DEFAULT, Root: "asset/views",
			Shared: "shared", Left: "{%", Right: "%}",
		},
		drivers: make(map[string]ViewDriver),
		helpers: make(map[string]Helper),
		actions: make(Map),
	}
}

type (
	ViewConfig struct {
		Driver  string
		Root    string
		Shared  string
		Left    string
		Right   string
		Setting Map
	}
	// ViewDriver 日志驱动
	ViewDriver interface {
		Connect(config ViewConfig) (ViewConnect, error)
	}
	// ViewConnect 日志连接
	ViewConnect interface {
		//打开、关闭
		Open() error
		Health() (ViewHealth, error)
		Close() error

		Parse(ViewBody) (string, error)
	}

	// ViewHealth 日志健康信息
	ViewHealth struct {
		Workload int64
	}
	ViewBody struct {
		View    string
		Site    string
		Lang    string
		Zone    *time.Location
		Data    Map
		Helpers Map
	}

	viewModule struct {
		mutex  sync.Mutex
		config ViewConfig

		drivers map[string]ViewDriver
		helpers map[string]Helper
		actions Map

		//视图配置，视图连接
		connect ViewConnect
	}

	Helper struct {
		Name   string   `json:"name"`
		Desc   string   `json:"desc"`
		Alias  []string `json:"alias"`
		Action Any      `json:"-"`
	}
)

func (module *viewModule) Builtin() {

}

//模块注册中心
func (module *viewModule) Register(key string, value Any, override bool) {
	switch val := value.(type) {
	case ViewDriver:
		module.Driver(key, val, override)
	case Helper:
		module.Helper(key, val, override)
	}
}
func (module *viewModule) Configure(config Map) {
	//设置驱动
	if driver, ok := config["driver"].(string); ok {
		module.config.Driver = driver
	}

	if vv, ok := config["root"].(string); ok {
		module.config.Root = vv
	}
	if vv, ok := config["shared"].(string); ok {
		module.config.Shared = vv
	}
	if vv, ok := config["left"].(string); ok {
		module.config.Left = vv
	}
	if vv, ok := config["right"].(string); ok {
		module.config.Right = vv
	}

	if vv, ok := config["setting"].(Map); ok {
		module.config.Setting = vv
	}
}

// Initialize
func (module *viewModule) Initialize() {

}

func (module *viewModule) connection(config ViewConfig) (ViewConnect, error) {
	if driver, ok := module.drivers[config.Driver]; ok {
		return driver.Connect(config)
	}
	panic("[视图]不支持的驱动" + config.Driver)
}
func (module *viewModule) Connect() {

	driver, ok := module.drivers[module.config.Driver]
	if ok == false {
		panic("Invalid cache driver: " + module.config.Driver)
	}

	// 建立连接
	connect, err := driver.Connect(module.config)
	if err != nil {
		panic("Failed to connect to cache: " + err.Error())
	}

	// 打开连接
	err = connect.Open()
	if err != nil {
		panic("Failed to open cache connect: " + err.Error())
	}

	//保存连接
	module.connect = connect
}
func (module *viewModule) Terminate() {
	if module.connect != nil {
		module.connect.Close()
	}
}

//注册驱动
func (module *viewModule) Driver(name string, driver ViewDriver, overrides ...bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	if driver == nil {
		panic("[日志]驱动不可为空")
	}

	override := true
	if len(overrides) > 0 {
		override = overrides[0]
	}

	if override {
		module.drivers[name] = driver
	} else {
		if module.drivers[name] == nil {
			module.drivers[name] = driver
		}
	}
}
func (module *viewModule) Helper(name string, config Helper, overrides ...bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	override := true
	if len(overrides) > 0 {
		override = overrides[0]
	}

	alias := make([]string, 0)
	if name != "" {
		alias = append(alias, name)
	}
	if config.Alias != nil {
		alias = append(alias, config.Alias...)
	}

	for _, key := range alias {
		if override {
			module.helpers[key] = config
			module.actions[key] = config.Action
		} else {
			if _, ok := module.helpers[key]; ok == false {
				module.helpers[key] = config
				module.actions[key] = config.Action
			}
		}

	}
}

func (module *viewModule) Parse(body ViewBody) (string, error) {
	if module.connect == nil {
		return "", errors.New("[视图]无效连接")
	}
	return module.connect.Parse(body)
}

func ParseView(body ViewBody) (string, error) {
	return mView.Parse(body)
}
