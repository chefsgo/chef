package chef

import (
	"errors"
	"sync"
	"time"

	. "github.com/chefsgo/base"
	"github.com/chefsgo/util"
)

var (
	mSession = &sessionModule{
		configs:   make(map[string]SessionConfig, 0),
		drivers:   make(map[string]SessionDriver, 0),
		instances: make(map[string]sessionInstance, 0),
	}

	errInvalidSessionConnection = errors.New("Invalid session connection.")
)

type (
	SessionConfig struct {
		Driver  string
		Weight  int
		Prefix  string
		Expiry  time.Duration
		Setting Map
	}
	// SessionDriver 数据驱动
	SessionDriver interface {
		Connect(name string, config SessionConfig) (SessionConnect, error)
	}

	// SessionConnect 会话连接
	SessionConnect interface {
		Open() error
		Close() error

		Read(id string) (Map, error)
		Write(id string, value Map, expiry time.Duration) error
		Delete(id string) error
		Clear(perfix string) error
	}

	// sessionInstance 会话实例
	sessionInstance struct {
		config  SessionConfig
		connect SessionConnect
	}

	sessionModule struct {
		mutex   sync.Mutex
		configs map[string]SessionConfig
		drivers map[string]SessionDriver

		instances map[string]sessionInstance

		weights  map[string]int
		hashring *util.HashRing
	}
)

// register 模块注册中心
func (module *sessionModule) register(name string, value Any, override bool) {
	switch config := value.(type) {
	case SessionDriver:
		module.Driver(name, config, override)
	}
}

func (module *sessionModule) configure(config Map) {
	var confs Map
	if vvv, ok := config["session"].(Map); ok {
		confs = vvv
	}

	for key, val := range confs {
		if conf, ok := val.(Map); ok {

			cfg := SessionConfig{
				Driver: DEFAULT, Weight: 1, Expiry: time.Hour * 24,
			}

			if driver, ok := conf["driver"].(string); ok {
				cfg.Driver = driver
			}

			//分配权重
			if weight, ok := conf["weight"].(int); ok {
				cfg.Weight = weight
			}
			if weight, ok := conf["weight"].(int64); ok {
				cfg.Weight = int(weight)
			}
			if weight, ok := conf["weight"].(float64); ok {
				cfg.Weight = int(weight)
			}

			//默认过期时间，单位秒
			if expiry, ok := conf["expiry"].(string); ok {
				dur, err := util.ParseDuration(expiry)
				if err == nil {
					cfg.Expiry = dur
				}
			}
			if expiry, ok := conf["expiry"].(int); ok {
				cfg.Expiry = time.Second * time.Duration(expiry)
			}
			if expiry, ok := conf["expiry"].(float64); ok {
				cfg.Expiry = time.Second * time.Duration(expiry)
			}

			// session 全部参与分布
			if cfg.Weight <= 0 {
				cfg.Weight = 1
			}

			//保存配置
			module.configs[key] = cfg
		}
	}
}

// Driver 注册驱动
func (module *sessionModule) Driver(name string, driver SessionDriver, overrides ...bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	if driver == nil {
		panic("Invalid session driver: " + name)
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

// initialize 初始化
func (module *sessionModule) initialize() {

	// 如果没有配置任何连接时，默认一个
	if len(module.configs) == 0 {
		module.configs[DEFAULT] = SessionConfig{
			Driver: DEFAULT, Weight: 1, Expiry: time.Hour * 24,
		}
	}

	//记录要参与分布的连接和权重
	weights := make(map[string]int)

	for name, config := range module.configs {
		driver, ok := module.drivers[config.Driver]
		if ok == false {
			panic("Invalid session driver: " + config.Driver)
		}

		// 建立连接
		connect, err := driver.Connect(name, config)
		if err != nil {
			panic("Failed to connect to session: " + err.Error())
		}

		// 打开连接
		err = connect.Open()
		if err != nil {
			panic("Failed to open session connect: " + err.Error())
		}

		//保存连接
		module.instances[name] = sessionInstance{
			config, connect,
		}

		//只有设置了权重的才参与分布
		if config.Weight > 0 {
			weights[name] = config.Weight
		}
	}

	//hashring分片
	module.weights = weights
	module.hashring = util.NewHashRing(weights)
}

// launch session模块launch暂时没有用
func (module *sessionModule) launch() {
	// fmt.Println("session launched")
}

// terminate 结束模块
func (module *sessionModule) terminate() {
	for _, ins := range module.instances {
		ins.connect.Close()
	}
}

func (module *sessionModule) Read(id string) (Map, error) {
	locate := module.hashring.Locate(id)

	if inst, ok := module.instances[locate]; ok {
		key := inst.config.Prefix + id //加前缀
		return inst.connect.Read(key)
	}

	return nil, errInvalidSessionConnection

}

func (module *sessionModule) Write(id string, value Map, expiries ...time.Duration) error {
	locate := module.hashring.Locate(id)

	if inst, ok := module.instances[locate]; ok {
		expiry := inst.config.Expiry
		if len(expiries) > 0 {
			expiry = expiries[0]
		}

		//KEY加上前缀
		key := inst.config.Prefix + id

		return inst.connect.Write(key, value, expiry)
	}

	return errInvalidSessionConnection
}

func (module *sessionModule) Delete(id string) error {
	locate := module.hashring.Locate(id)

	if inst, ok := module.instances[locate]; ok {
		key := inst.config.Prefix + id
		return inst.connect.Delete(key)
	}

	return errInvalidSessionConnection
}

func (module *sessionModule) Clear() error {
	for _, inst := range module.instances {
		inst.connect.Clear(inst.config.Prefix)
	}

	return errInvalidSessionConnection
}
