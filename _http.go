package chef

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	. "github.com/chefsgo/base"
	"github.com/chefsgo/util"
)

var (
	mHttp = newHttpModule()
)

func newHttpModule() *httpModule {
	http := &httpModule{
		config: HttpConfig{
			Driver: DEFAULT, Port: 80, Charset: UTF8,
		},
		cross: CrossConfig{
			Allow: false,
		},

		drivers: make(map[string]HttpDriver),

		sites: make(map[string]SiteConfig, 0),
		hosts: make(map[string]string, 0),

		routers:       make(map[string]Router),
		routerNames:   make([]string, 0),
		routerActions: make(map[string][]HttpFunc),

		requestFilters:  make(map[string]RequestFilter),
		requestActions:  make(map[string][]HttpFunc),
		executeFilters:  make(map[string]ExecuteFilter),
		executeActions:  make(map[string][]HttpFunc),
		responseFilters: make(map[string]ResponseFilter),
		responseActions: make(map[string][]HttpFunc),

		foundNames:    make([]string, 0),
		foundHandlers: make(map[string]FoundHandler),
		foundActions:  make(map[string][]HttpFunc),

		errorNames:    make([]string, 0),
		errorHandlers: make(map[string]ErrorHandler),
		errorActions:  make(map[string][]HttpFunc),

		failedNames:    make([]string, 0),
		failedHandlers: make(map[string]FailedHandler),
		failedActions:  make(map[string][]HttpFunc),

		deniedNames:    make([]string, 0),
		deniedHandlers: make(map[string]DeniedHandler),
		deniedActions:  make(map[string][]HttpFunc),

		items: make(map[string]Item, 0),
	}

	http.url = &httpUrl{httpEmpty(http)}

	return http
}

type (
	HttpConfig struct {

		//驱动
		Driver string

		// Port HTTP端口
		Port int

		// HTTP 证书文件
		CertFile string
		KeyFile  string

		//这个配置为，空站点是否自动下发token，不覆盖子站点
		Issue bool `toml:"issue"`

		Charset string
		Domain  string
		Cookie  string
		Expiry  time.Duration
		MaxAge  time.Duration

		Upload string `toml:"upload"`
		Static string `toml:"static"`
		Shared string `toml:"shared"`

		Defaults []string `toml:"defaults"`

		Setting Map `toml:"setting"`
	}
	SiteConfig struct {
		Name string `toml:"name"`
		Ssl  bool   `toml:"ssl"`

		//  Hosts 站点域名
		Hosts []string
		// Weights 域名对应的权重
		// 当有多个的时候，可以设置每个域名的权重
		// 为了获取URL时候，
		Weights []int

		//域名的权重，获取URL的时候，分配
		// Weights []int    `toml:"weights"`

		// Charset 字符集
		Charset string
		// Domain 站点主域名，为空则继承http的Domain,
		Domain string

		// Issue 是否自动生成token
		Issue bool
		// Cookie 写入的cookie名，如果不为空，自动写入token到cookie
		Cookie string

		// Expiry 自定义的token过期时间
		Expiry time.Duration
		// MaxAge 自定义的cookie客户端过期时间
		MaxAge time.Duration

		Confuse  string `toml:"confuse"`
		Validate string `toml:"validate"`

		// Format 客户端验证 格式
		Format string `toml:"format"`
		// Timeout 客户端验证 超时时间
		Timeout string `toml:"timeout"`

		Setting Map `toml:"setting"`
	}
	CrossConfig struct {
		// Allow 是否允许跨域
		Allow bool
		// Methods 允许的方法
		Methods []string
		// Origins 允许的来源
		Origins []string
		// Headers 允许的HTTP头
		Headers []string
	}
	// HttpDriver 日志驱动
	HttpDriver interface {
		Connect(config HttpConfig) (HttpConnect, error)
	}
	// HttpHealth HTTP健康信息
	HttpHealth struct {
		Workload int64
	}

	//事件连接
	HttpConnect interface {
		Open() error
		Health() (HttpHealth, error)
		Close() error

		Accept(HttpHandler) error
		Register(name string, config HttpRegister) error

		//开始
		Start() error
		//开始TLS
		StartTLS(certFile, keyFile string) error
	}

	HttpRegister struct {
		Site    string
		Uris    []string
		Methods []string
		Hosts   []string
	}

	HttpHandler func(HttpThread)
	HttpThread  interface {
		Name() string
		Site() string
		Params() Map
		Request() *http.Request
		Response() http.ResponseWriter
		Finish() error
	}

	//跳转
	httpGotoBody struct {
		url string
	}
	httpTextBody struct {
		text string
	}
	httpHtmlBody struct {
		html string
	}
	httpScriptBody struct {
		script string
	}
	httpJsonBody struct {
		json Any
	}
	httpJsonpBody struct {
		json     Any
		callback string
	}
	httpApiBody struct {
		code int
		text string
		data Map
	}
	httpXmlBody struct {
		xml Any
	}
	httpFileBody struct {
		file string
		name string
	}
	httpDownBody struct {
		bytes []byte
		name  string
	}
	httpBufferBody struct {
		buffer io.ReadCloser
		name   string
	}
	httpViewBody struct {
		view  string
		model Any
	}
	httpProxyBody struct {
		url *url.URL
	}

	RawBody string

	httpSite struct {
		name string
		root string
	}

	httpModule struct {
		mutex  sync.Mutex
		config HttpConfig
		cross  CrossConfig

		drivers map[string]HttpDriver

		sites map[string]SiteConfig
		hosts map[string]string

		routers       map[string]Router
		routerNames   []string
		routerActions map[string][]HttpFunc

		//拦截器
		requestNames    []string
		requestFilters  map[string]RequestFilter
		requestActions  map[string][]HttpFunc
		executeNames    []string
		executeFilters  map[string]ExecuteFilter
		executeActions  map[string][]HttpFunc
		responseNames   []string
		responseFilters map[string]ResponseFilter
		responseActions map[string][]HttpFunc

		//处理器
		foundNames     []string
		foundHandlers  map[string]FoundHandler
		foundActions   map[string][]HttpFunc
		errorNames     []string
		errorHandlers  map[string]ErrorHandler
		errorActions   map[string][]HttpFunc
		failedNames    []string
		failedHandlers map[string]FailedHandler
		failedActions  map[string][]HttpFunc
		deniedNames    []string
		deniedHandlers map[string]DeniedHandler
		deniedActions  map[string][]HttpFunc

		items map[string]Item

		connect HttpConnect
		url     *httpUrl
	}

	Filter struct {
		site     string   `json:"-"`
		Name     string   `json:"name"`
		Desc     string   `json:"desc"`
		Request  HttpFunc `json:"-"`
		Execute  HttpFunc `json:"-"`
		Response HttpFunc `json:"-"`
	}
	RequestFilter struct {
		site   string   `json:"-"`
		Name   string   `json:"name"`
		Desc   string   `json:"desc"`
		Action HttpFunc `json:"-"`
	}
	ExecuteFilter struct {
		site   string   `json:"-"`
		Name   string   `json:"name"`
		Desc   string   `json:"desc"`
		Action HttpFunc `json:"-"`
	}
	ResponseFilter struct {
		site   string   `json:"-"`
		Name   string   `json:"name"`
		Desc   string   `json:"desc"`
		Action HttpFunc `json:"-"`
	}

	Handler struct {
		site   string   `json:"-"`
		Name   string   `json:"name"`
		Desc   string   `json:"desc"`
		Found  HttpFunc `json:"-"`
		Error  HttpFunc `json:"-"`
		Failed HttpFunc `json:"-"`
		Denied HttpFunc `json:"-"`
	}
	FoundHandler struct {
		site   string   `json:"-"`
		Name   string   `json:"name"`
		Desc   string   `json:"desc"`
		Action HttpFunc `json:"-"`
	}
	ErrorHandler struct {
		site   string   `json:"-"`
		Name   string   `json:"name"`
		Desc   string   `json:"desc"`
		Action HttpFunc `json:"-"`
	}
	FailedHandler struct {
		site   string   `json:"-"`
		Name   string   `json:"name"`
		Desc   string   `json:"desc"`
		Action HttpFunc `json:"-"`
	}
	DeniedHandler struct {
		site   string   `json:"-"`
		Name   string   `json:"name"`
		Desc   string   `json:"desc"`
		Action HttpFunc `json:"-"`
	}

	Routing map[string]Router
	Router  struct {
		site     string   `json:"-"`
		Uri      string   `json:"uri"`
		Uris     []string `json:"uris"`
		Name     string   `json:"name"`
		Desc     string   `json:"desc"`
		Method   string   `json:"method"` //真实记录实际的method，view中要用，要开放
		Nullable bool     `json:"nullable"`
		Socket   bool     `json:"socket"`
		Setting  Map      `json:"setting"`
		Coding   bool     `json:"coding"`

		Sign map[string]Sign `json:"sign"`
		Find Find            `json:"find"`
		Args Vars            `json:"args"`
		Data Vars            `json:"data"`

		Routing Routing    `json:"routing"`
		Action  HttpFunc   `json:"-"`
		Actions []HttpFunc `json:"-"`

		Found  HttpFunc `json:"-"`
		Error  HttpFunc `json:"-"`
		Failed HttpFunc `json:"-"`
		Denied HttpFunc `json:"-"`

		Token bool `json:"token"`
		Auth  bool `json:"auth"`
	}

	// Signs map[string]Sign
	Sign struct {
		Sign     string `json:"sign"`
		Required bool   `json:"required"`
		Method   string `json:"method"`
		Args     string `json:"args"`
		Name     string `json:"name"`
		Desc     string `json:"desc"`
		Empty    Res    `json:"-"`
		Error    Res    `json:"-"`
	}
	Find map[string]Item
	Item struct {
		Value    string   `json:"value"`
		Required bool     `json:"required"`
		Method   string   `json:"method"`
		Args     string   `json:"args"`
		Name     string   `json:"name"`
		Desc     string   `json:"desc"`
		Alias    []string `json:"-"`
		Empty    Res      `json:"-"`
		Error    Res      `json:"-"`
	}
)

// Builtin
func (module *httpModule) Builtin() {
	module.Register(".index", Router{
		Uri: "/", Name: "index", Desc: "index",
		Action: func(ctx *Access) {
			ctx.Text("hello arkgo")
		},
	}, false)
}

// Register
func (module *httpModule) Register(key string, value Any, override bool) {
	switch val := value.(type) {
	case HttpDriver:
		module.Driver(key, val, override)
	case Router:
		module.Router(key, val, override)

	case Filter:
		module.Filter(key, val, override)
	case RequestFilter:
		module.RequestFilter(key, val, override)
	case ExecuteFilter:
		module.ExecuteFilter(key, val, override)
	case ResponseFilter:
		module.ResponseFilter(key, val, override)
	case Handler:

		module.Handler(key, val, override)
	case FoundHandler:
		module.FoundHandler(key, val, override)
	case ErrorHandler:
		module.ErrorHandler(key, val, override)
	case FailedHandler:
		module.FailedHandler(key, val, override)
	case DeniedHandler:
		module.DeniedHandler(key, val, override)

	case Item:
		module.Item(key, val, override)
	}

}

func (module *httpModule) httpConfigure(config Map) {
	// 驱动
	if vv, ok := config["driver"].(string); ok {
		module.config.Driver = vv
	}
	// 端口
	if vv, ok := config["port"].(int); ok {
		module.config.Port = vv
	}
	if vv, ok := config["port"].(int64); ok {
		module.config.Port = int(vv)
	}

	// charset
	if vv, ok := config["charset"].(string); ok {
		module.config.Charset = vv
	}

	// cookie
	if vv, ok := config["cookie"].(string); ok {
		module.config.Cookie = vv
	}
	//Expiry
	expiry := parseDurationFromMap(config, "expiry")
	if expiry >= 0 {
		module.config.Expiry = expiry
	}

	//MaxAge
	maxage := parseDurationFromMap(config, "maxage")
	if maxage >= 0 {
		module.config.MaxAge = maxage
	}

	// upload
	if vv, ok := config["upload"].(string); ok {
		module.config.Upload = vv
	}

	// static
	if vv, ok := config["static"].(string); ok {
		module.config.Static = vv
	}

	// shared
	if vv, ok := config["shared"].(string); ok {
		module.config.Shared = vv
	}

	// defaults
	if vvs, ok := config["default"].(string); ok {
		module.config.Defaults = []string{vvs}
	}
	if vvs, ok := config["defaults"].([]string); ok {
		module.config.Defaults = vvs
	}

	//setting
	if vv, ok := config["setting"].(Map); ok {
		module.config.Setting = vv
	}
}

func (module *httpModule) crossConfigure(config Map) {
	// 是否允许
	if vv, ok := config["allow"].(bool); ok {
		module.cross.Allow = vv
	}

	// methods
	if vv, ok := config["method"].(string); ok {
		module.cross.Methods = []string{vv}
	}
	if vvs, ok := config["methods"].([]string); ok {
		module.cross.Methods = vvs
	}

	// Origins
	if vv, ok := config["origins"].(string); ok {
		module.cross.Origins = []string{vv}
	}
	if vvs, ok := config["origins"].([]string); ok {
		module.cross.Origins = vvs
	}

	// Headers
	if vv, ok := config["headers"].(string); ok {
		module.cross.Headers = []string{vv}
	}
	if vvs, ok := config["headers"].([]string); ok {
		module.cross.Headers = vvs
	}
}

func (module *httpModule) siteConfigure(name string, config Map) {

	site := SiteConfig{}

	// 名称
	if vv, ok := config["name"].(string); ok {
		site.Name = vv
	}
	// SSL
	if vv, ok := config["ssl"].(bool); ok {
		site.Ssl = vv
	}

	// issue
	if vv, ok := config["issue"].(bool); ok {
		site.Issue = vv
	}

	// hosts
	if vv, ok := config["host"].(string); ok {
		site.Hosts = []string{vv}
	}
	if vvs, ok := config["hosts"].([]string); ok {
		site.Hosts = vvs
	}

	// charset
	if vv, ok := config["charset"].(string); ok {
		site.Charset = vv
	}

	// cookie
	if vv, ok := config["cookie"].(string); ok {
		site.Cookie = vv
	}

	//Expiry
	expiry := parseDurationFromMap(config, "expiry")
	if expiry >= 0 {
		site.Expiry = expiry
	}

	//MaxAge
	maxage := parseDurationFromMap(config, "maxage")
	if maxage >= 0 {
		site.MaxAge = maxage
	}

	// Confuse
	if vv, ok := config["confuse"].(string); ok {
		site.Confuse = vv
	}
	// Validate
	if vv, ok := config["validate"].(string); ok {
		site.Validate = vv
	}
	// Format
	if vv, ok := config["format"].(string); ok {
		site.Format = vv
	}

	//MaxAge
	timeout := parseDurationFromMap(config, "timeout")
	if timeout >= 0 {
		site.Timeout = timeout
	}

	//保存site配置
	module.sites[name] = site
}

func (module *httpModule) Configure(config Map) {
	if http, ok := config["http"].(Map); ok {
		module.httpConfigure(http)
	}
	if cross, ok := config["cross"].(Map); ok {
		module.httpConfigure(cross)
	}

	// 站点配置
	var sites Map
	if vvv, ok := config["site"].(Map); ok {
		sites = vvv
	}

	//记录上一层的site配置，如果有的话
	defSiteConfig := Map{}

	for key, val := range sites {
		if conf, ok := val.(Map); ok {
			//直接注册，然后删除当前key
			module.siteConfigure(key, conf)
		} else {
			//记录上一层的配置，如果有的话
			defSiteConfig[key] = val
		}
	}

	if len(defSiteConfig) > 0 {
		//默认为空站点
		module.siteConfigure("", defSiteConfig)
	}
}

// func (module *httpModule) connecting(config HttpConfig) (HttpConnect, error) {
// 	if driver, ok := module.drivers[config.Driver]; ok {
// 		return driver.Connect(config)
// 	}
// 	panic("[HTTP]不支持的驱动" + config.Driver)
// }

// Initialize
func (module *httpModule) Initialize() {
	driver, ok := module.drivers[module.config.Driver]
	if ok == false {
		panic("Invalid http driver: " + module.config.Driver)
	}

	if module.config.Port <= 0 || module.config.Port > 65535 {
		module.config.Port = 80
	}

	if module.config.Upload == "" {
		module.config.Upload = os.TempDir()
	}
	if module.config.Static == "" {
		module.config.Static = "asset/statics"
	}
	if module.config.Shared == "" {
		module.config.Shared = "shared"
	}

	if module.config.Defaults == nil || len(module.config.Defaults) == 0 {
		module.config.Defaults = []string{
			"index.html", "default.html", "index.htm", "default.html",
		}
	}

	for key, site := range module.sites {
		if site.Charset == "" {
			site.Charset = module.config.Charset
		}

		if site.Domain == "" {
			site.Domain = module.config.Domain
		}

		if site.Expiry == 0 {
			site.Expiry = module.config.Expiry
		}
		if site.MaxAge <= 0 {
			site.MaxAge = module.config.MaxAge
		}

		if site.Hosts == nil {
			site.Hosts = make([]string, 0)
		}

		//如果没有域名，把站点的key加上，做为子域名
		if len(site.Hosts) == 0 {
			site.Hosts = append(site.Hosts, key)
		}

		//加上域名
		for i, host := range site.Hosts {
			if strings.HasSuffix(host, site.Domain) == false {
				site.Hosts[i] = host + "." + site.Domain
			}
		}

		//待优化，这个权重是老代码复制
		//多host情况下，获取URL的时候，来分布
		if site.Weights == nil || len(site.Weights) == 0 {
			site.Weights = []int{}
			for range site.Hosts {
				site.Weights = append(site.Weights, 1)
			}
		}

		if site.Format == "" {
			site.Format = `{device}/{system}/{version}/{client}/{number}/{time}/{path}`
		}

		//记录http的所有域名
		for _, host := range site.Hosts {
			module.hosts[host] = key
		}
	}

	//空站点
	if _, ok := module.sites[""]; ok == false {
		module.sites[""] = SiteConfig{
			Name: "空站点",
		}
	}

	module.initRouterActions()
	module.initFilterActions()
	module.initHandlerActions()

}

// Connect
func (module *httpModule) Connect() {

	driver, ok := module.drivers[module.config.Driver]
	if ok == false {
		panic("Invalid http driver: " + module.config.Driver)
	}

	// 建立连接
	connect, err := driver.Connect(module.config)
	if err != nil {
		panic("Failed to connect to http: " + err.Error())
	}

	// 打开连接
	err = connect.Open()
	if err != nil {
		panic("Failed to open http connect: " + err.Error())
	}

	//绑定回调
	connect.Accept(module.serve)

	//注册路由
	// for k, v := range module.routers {
	for _, name := range module.routerNames {
		config := module.routers[name]
		regis := module.registering(config)
		err := connect.Register(name, regis)
		if err != nil {
			panic("[HTTP]注册失败：" + err.Error())
		}
	}

	module.connect = connect
}

func (module *httpModule) registering(config Router) HttpRegister {

	//Uris
	uris := []string{}
	if config.Uri != "" {
		uris = append(uris, config.Uri)
	}
	if config.Uris != nil {
		uris = append(uris, config.Uris...)
	}

	//方法
	methods := []string{}
	if config.Method != "" {
		methods = append(methods, config.Method)
	}

	site := config.site

	regis := HttpRegister{Site: site, Uris: config.Uris, Methods: methods}

	if cfg, ok := module.sites[site]; ok {
		regis.Hosts = cfg.Hosts
	}

	return regis
}

func (module *httpModule) Luanch() {
	config := module.config
	if config.CertFile != "" && config.KeyFile != "" {
		module.connect.StartTLS(config.CertFile, config.KeyFile)
	} else {
		module.connect.Start()
	}
}

//Terminate
func (module *httpModule) Terminate() {
	if module.connect != nil {
		module.connect.Close()
	}
}

//注册驱动
func (module *httpModule) Driver(name string, driver HttpDriver, overrides ...bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	if driver == nil {
		panic("[HTTP]驱动不可为空")
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

func (module *httpModule) Router(name string, config Router, overrides ...bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	override := true
	if len(overrides) > 0 {
		override = overrides[0]
	}

	names := strings.Split(name, ".")
	if len(names) <= 1 {
		name = "*." + name
	}

	//直接的时候直接拆分成目标格式
	objects := make(map[string]Router)
	if strings.HasPrefix(name, "*.") {
		//全站点
		for site, _ := range module.sites {
			siteName := strings.Replace(name, "*", site, 1)
			siteConfig := config //直接复制一份

			siteConfig.site = site

			//先记录下
			objects[siteName] = siteConfig
		}
	} else {
		if len(names) >= 2 {
			config.site = names[0]
		}
		//单站点
		objects[name] = config
	}

	//处理对方是单方法，还是多方法
	routers := make(map[string]Router)
	for routerName, routerConfig := range objects {

		if routerConfig.Routing != nil {
			//多method版本
			for method, methodConfig := range routerConfig.Routing {
				realName := fmt.Sprintf("%s.%s", routerName, method)
				realConfig := routerConfig
				//从顶级复制	这样有问题，realConfig中的非引用变量，是复制了，但是引用变量，是引的引用，所以，如果修改比如  realConfig.Args，会把 routerConfig也修改了
				//MAP要重新复制
				if routerConfig.Sign != nil {
					realConfig.Sign = make(map[string]Sign)
					for k, v := range routerConfig.Sign {
						realConfig.Sign[k] = v
					}
				}
				if routerConfig.Find != nil {
					realConfig.Find = make(Find)
					for k, v := range routerConfig.Find {
						realConfig.Find[k] = v
					}
				}
				if routerConfig.Args != nil {
					realConfig.Args = make(Vars)
					for k, v := range routerConfig.Args {
						realConfig.Args[k] = v
					}
				}
				if routerConfig.Data != nil {
					realConfig.Data = make(Vars)
					for k, v := range routerConfig.Data {
						realConfig.Data[k] = v
					}
				}
				if routerConfig.Setting != nil {
					realConfig.Setting = make(Map)
					for k, v := range routerConfig.Setting {
						realConfig.Setting[k] = v
					}
				}

				//相关参数
				realConfig.Method = method
				realConfig.Socket = methodConfig.Socket
				realConfig.Nullable = methodConfig.Nullable

				//复制子级的定义
				if methodConfig.Name != "" {
					realConfig.Name = methodConfig.Name
				}
				if methodConfig.Desc != "" {
					realConfig.Desc = methodConfig.Desc
				}

				//复制设置
				if methodConfig.Setting != nil {
					if realConfig.Setting == nil {
						realConfig.Setting = make(Map)
					}
					for k, v := range methodConfig.Setting {
						realConfig.Setting[k] = v
					}
				}

				//复制args
				if methodConfig.Args != nil {
					if realConfig.Args == nil {
						realConfig.Args = Vars{}
					}
					for k, v := range methodConfig.Args {
						realConfig.Args[k] = v
					}
				}

				//复制data
				if methodConfig.Data != nil {
					if realConfig.Data == nil {
						realConfig.Data = Vars{}
					}
					for k, v := range methodConfig.Data {
						realConfig.Data[k] = v
					}
				}
				//复制auth
				//待优化：是否使用专用类型
				if methodConfig.Sign != nil {
					if realConfig.Sign == nil {
						realConfig.Sign = map[string]Sign{}
					}
					for k, v := range methodConfig.Sign {
						realConfig.Sign[k] = v
					}
				}
				//复制item
				//待优化：是否使用专用类型
				if methodConfig.Find != nil {
					if realConfig.Find == nil {
						realConfig.Find = Find{}
					}
					for k, v := range methodConfig.Find {
						realConfig.Find[k] = v
					}
				}

				//复制方法
				if methodConfig.Action != nil {
					realConfig.Action = methodConfig.Action
				}
				if methodConfig.Actions != nil {
					realConfig.Actions = methodConfig.Actions
				}

				//复制处理器
				if methodConfig.Found != nil {
					realConfig.Found = methodConfig.Found
				}
				if methodConfig.Error != nil {
					realConfig.Error = methodConfig.Error
				}
				if methodConfig.Failed != nil {
					realConfig.Failed = methodConfig.Failed
				}
				if methodConfig.Denied != nil {
					realConfig.Denied = methodConfig.Denied
				}

				//加入列表
				routers[realName] = realConfig
			}

		} else {
			//单方法版本，记录一个*，好在记录power的时候，去掉最后一节
			realName := fmt.Sprintf("%s.*", routerName)
			routers[realName] = routerConfig
		}
	}

	//这里才是真的注册
	for key, val := range routers {
		key = strings.ToLower(key)

		//一些默认的处理

		//复制uri到uris，默认使用uris
		if val.Uris == nil {
			val.Uris = make([]string, 0)
		}
		if val.Uri != "" {
			val.Uris = append(val.Uris, val.Uri)
			val.Uri = ""
		}
		//复制action
		if val.Actions == nil {
			val.Actions = make([]HttpFunc, 0)
		}
		if val.Action != nil {
			val.Actions = append(val.Actions, val.Action)
			val.Action = nil
		}

		//这里全局置空
		val.Routing = nil

		if override {
			module.routers[key] = val
			module.routerNames = append(module.routerNames, key)
		} else {
			if _, ok := module.routers[key]; ok == false {
				module.routers[key] = val
				module.routerNames = append(module.routerNames, key)
			}
		}
	}
}

func (module *httpModule) initRouterActions() {
	for name, config := range module.routers {
		if _, ok := module.routerActions[name]; ok == false {
			module.routerActions[name] = make([]HttpFunc, 0)
		}

		if config.Action != nil {
			module.routerActions[name] = append(module.routerActions[name], config.Action)
		}
		if config.Actions != nil {
			module.routerActions[name] = append(module.routerActions[name], config.Actions...)
		}
	}
}
func (module *httpModule) Routers(sites ...string) map[string]Router {
	prefix := ""
	if len(sites) > 0 {
		prefix = sites[0] + "."
	}

	routers := make(map[string]Router)
	for name, config := range module.routers {
		if prefix == "" || strings.HasPrefix(name, prefix) {
			routers[name] = config
		}
	}

	return routers
}
func (module *httpModule) Filter(name string, config Filter, override bool) {
	if config.Request != nil {
		module.RequestFilter(name, RequestFilter{config.site, config.Name, config.Desc, config.Request}, override)
	}
	if config.Execute != nil {
		module.ExecuteFilter(name, ExecuteFilter{config.site, config.Name, config.Desc, config.Execute}, override)
	}
	if config.Response != nil {
		module.ResponseFilter(name, ResponseFilter{config.site, config.Name, config.Desc, config.Response}, override)
	}
}

func (module *httpModule) RequestFilter(name string, config RequestFilter, overrides ...bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	override := true
	if len(overrides) > 0 {
		override = overrides[0]
	}

	//从名称里找站点
	names := strings.Split(name, ".")
	if len(names) <= 1 {
		name = "*." + name
	}

	//直接的时候直接拆分成目标格式
	filters := make(map[string]RequestFilter)
	if strings.HasPrefix(name, "*.") {
		//全站点
		for site, _ := range module.sites {
			siteName := strings.Replace(name, "*", site, 1)
			filters[siteName] = RequestFilter{
				site, config.Name, config.Desc, config.Action,
			}
		}
	} else {
		if len(names) >= 2 {
			config.site = names[0]
		}
		//单站点
		filters[name] = config
	}

	for key, val := range filters {
		if override {
			module.requestFilters[key] = val
			module.requestNames = append(module.requestNames, key)
		} else {
			if _, ok := module.requestFilters[key]; ok == false {
				module.requestFilters[key] = val
				module.requestNames = append(module.requestNames, key)
			}
		}
	}
}

func (module *httpModule) ExecuteFilter(name string, config ExecuteFilter, overrides ...bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	override := true
	if len(overrides) > 0 {
		override = overrides[0]
	}

	//从名称里找站点
	names := strings.Split(name, ".")
	if len(names) <= 1 {
		name = "*." + name
	}

	//直接的时候直接拆分成目标格式
	filters := make(map[string]ExecuteFilter)
	if strings.HasPrefix(name, "*.") {
		//全站点
		for site, _ := range module.sites {
			siteName := strings.Replace(name, "*", site, 1)
			filters[siteName] = ExecuteFilter{
				site, config.Name, config.Desc, config.Action,
			}
		}
	} else {
		if len(names) >= 2 {
			config.site = names[0]
		}
		//单站点
		filters[name] = config
	}

	for key, val := range filters {
		if override {
			module.executeFilters[key] = val
			module.executeNames = append(module.executeNames, key)
		} else {
			if _, ok := module.executeFilters[key]; ok == false {
				module.executeFilters[key] = val
				module.executeNames = append(module.executeNames, key)
			}
		}
	}
}

func (module *httpModule) ResponseFilter(name string, config ResponseFilter, overrides ...bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	override := true
	if len(overrides) > 0 {
		override = overrides[0]
	}

	//从名称里找站点
	names := strings.Split(name, ".")
	if len(names) <= 1 {
		name = "*." + name
	}

	//直接的时候直接拆分成目标格式
	filters := make(map[string]ResponseFilter)
	if strings.HasPrefix(name, "*.") {
		//全站点
		for site, _ := range module.sites {
			siteName := strings.Replace(name, "*", site, 1)
			filters[siteName] = ResponseFilter{
				site, config.Name, config.Desc, config.Action,
			}
		}
	} else {
		if len(names) >= 2 {
			config.site = names[0]
		}
		//单站点
		filters[name] = config
	}

	for key, val := range filters {
		if override {
			module.responseFilters[key] = val
			module.responseNames = append(module.responseNames, key)
		} else {
			if _, ok := module.responseFilters[key]; ok == false {
				module.responseFilters[key] = val
				module.responseNames = append(module.responseNames, key)
			}
		}
	}
}

func (module *httpModule) initFilterActions() {
	//请求拦截器
	for _, name := range module.requestNames {
		config := module.requestFilters[name]
		if _, ok := module.requestActions[config.site]; ok == false {
			module.requestActions[config.site] = make([]HttpFunc, 0)
		}
		module.requestActions[config.site] = append(module.requestActions[config.site], config.Action)
	}

	//执行拦截器
	for _, name := range module.executeNames {
		config := module.executeFilters[name]
		if _, ok := module.executeActions[config.site]; ok == false {
			module.executeActions[config.site] = make([]HttpFunc, 0)
		}
		module.executeActions[config.site] = append(module.executeActions[config.site], config.Action)
	}

	//响应拦截器
	for _, name := range module.responseNames {
		config := module.responseFilters[name]
		if _, ok := module.responseActions[config.site]; ok == false {
			module.responseActions[config.site] = make([]HttpFunc, 0)
		}
		module.responseActions[config.site] = append(module.responseActions[config.site], config.Action)
	}
}

func (module *httpModule) Handler(name string, config Handler, override bool) {
	if config.Found != nil {
		module.FoundHandler(name, FoundHandler{config.site, config.Name, config.Desc, config.Found}, override)
	}
	if config.Error != nil {
		module.ErrorHandler(name, ErrorHandler{config.site, config.Name, config.Desc, config.Error}, override)
	}
	if config.Failed != nil {
		module.FailedHandler(name, FailedHandler{config.site, config.Name, config.Desc, config.Failed}, override)
	}
	if config.Denied != nil {
		module.DeniedHandler(name, DeniedHandler{config.site, config.Name, config.Desc, config.Denied}, override)
	}
}

func (module *httpModule) FoundHandler(name string, config FoundHandler, overrides ...bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	override := true
	if len(overrides) > 0 {
		override = overrides[0]
	}

	//从名称里找站点
	names := strings.Split(name, ".")
	if len(names) <= 1 {
		name = "*." + name
	}

	//直接的时候直接拆分成目标格式
	handlers := make(map[string]FoundHandler)
	if strings.HasPrefix(name, "*.") {
		//全站点
		for site, _ := range module.sites {
			siteName := strings.Replace(name, "*", site, 1)
			handlers[siteName] = FoundHandler{
				site, config.Name, config.Desc, config.Action,
			}
		}
	} else {
		if len(names) >= 2 {
			config.site = names[0]
		}
		//单站点
		handlers[name] = config
	}

	for key, val := range handlers {
		if override {
			module.foundHandlers[key] = val
			module.foundNames = append(module.foundNames, key)
		} else {
			if _, ok := module.foundHandlers[key]; ok == false {
				module.foundHandlers[key] = val
				module.foundNames = append(module.foundNames, key)
			}
		}
	}
}

func (module *httpModule) ErrorHandler(name string, config ErrorHandler, overrides ...bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	override := true
	if len(overrides) > 0 {
		override = overrides[0]
	}

	//从名称里找站点
	names := strings.Split(name, ".")
	if len(names) <= 1 {
		name = "*." + name
	}

	//直接的时候直接拆分成目标格式
	handlers := make(map[string]ErrorHandler)
	if strings.HasPrefix(name, "*.") {
		//全站点
		for site, _ := range module.sites {
			siteName := strings.Replace(name, "*", site, 1)
			handlers[siteName] = ErrorHandler{
				site, config.Name, config.Desc, config.Action,
			}
		}
	} else {
		if len(names) >= 2 {
			config.site = names[0]
		}
		//单站点
		handlers[name] = config
	}

	for key, val := range handlers {
		if override {
			module.errorHandlers[key] = val
			module.errorNames = append(module.errorNames, key)
		} else {
			if _, ok := module.errorHandlers[key]; ok == false {
				module.errorHandlers[key] = val
				module.errorNames = append(module.errorNames, key)
			}
		}
	}
}

func (module *httpModule) FailedHandler(name string, config FailedHandler, overrides ...bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	override := true
	if len(overrides) > 0 {
		override = overrides[0]
	}

	//从名称里找站点
	names := strings.Split(name, ".")
	if len(names) <= 1 {
		name = "*." + name
	}

	//直接的时候直接拆分成目标格式
	handlers := make(map[string]FailedHandler)
	if strings.HasPrefix(name, "*.") {
		//全站点
		for site, _ := range module.sites {
			siteName := strings.Replace(name, "*", site, 1)
			handlers[siteName] = FailedHandler{
				site, config.Name, config.Desc, config.Action,
			}
		}
	} else {
		if len(names) >= 2 {
			config.site = names[0]
		}
		//单站点
		handlers[name] = config
	}

	for key, val := range handlers {
		if override {
			module.failedHandlers[key] = val
			module.failedNames = append(module.failedNames, key)
		} else {
			if _, ok := module.failedHandlers[key]; ok == false {
				module.failedHandlers[key] = val
				module.failedNames = append(module.failedNames, key)
			}
		}
	}
}

func (module *httpModule) DeniedHandler(name string, config DeniedHandler, overrides ...bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	override := true
	if len(overrides) > 0 {
		override = overrides[0]
	}

	//从名称里找站点
	names := strings.Split(name, ".")
	if len(names) <= 1 {
		name = "*." + name
	}

	//直接的时候直接拆分成目标格式
	handlers := make(map[string]DeniedHandler)
	if strings.HasPrefix(name, "*.") {
		//全站点
		for site, _ := range module.sites {
			siteName := strings.Replace(name, "*", site, 1)
			handlers[siteName] = DeniedHandler{
				site, config.Name, config.Desc, config.Action,
			}
		}
	} else {
		if len(names) >= 2 {
			config.site = names[0]
		}
		//单站点
		handlers[name] = config
	}

	for key, val := range handlers {
		if override {
			module.deniedHandlers[key] = val
			module.deniedNames = append(module.deniedNames, key)
		} else {
			if _, ok := module.deniedHandlers[key]; ok == false {
				module.deniedHandlers[key] = val
				module.deniedNames = append(module.deniedNames, key)
			}
		}
	}
}

func (module *httpModule) Item(name string, config Item, overrides ...bool) {
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
			module.items[key] = config
		} else {
			if _, ok := module.items[key]; ok == false {
				module.items[key] = config
			}
		}
	}
}

func (module *httpModule) ItemConfig(name string) *Item {
	if vv, ok := module.items[name]; ok == false {
		return &vv
	}
	return nil
}

func (module *httpModule) initHandlerActions() {
	//found处理器
	for _, name := range module.foundNames {
		config := module.foundHandlers[name]
		if _, ok := module.foundActions[config.site]; ok == false {
			module.foundActions[config.site] = make([]HttpFunc, 0)
		}
		module.foundActions[config.site] = append(module.foundActions[config.site], config.Action)
	}

	//error处理器
	for _, name := range module.errorNames {
		config := module.errorHandlers[name]
		if _, ok := module.errorActions[config.site]; ok == false {
			module.errorActions[config.site] = make([]HttpFunc, 0)
		}
		module.errorActions[config.site] = append(module.errorActions[config.site], config.Action)
	}

	//failed处理器
	for _, name := range module.failedNames {
		config := module.failedHandlers[name]
		if _, ok := module.failedActions[config.site]; ok == false {
			module.failedActions[config.site] = make([]HttpFunc, 0)
		}
		module.failedActions[config.site] = append(module.failedActions[config.site], config.Action)
	}

	//denied处理器
	for _, name := range module.deniedNames {
		config := module.deniedHandlers[name]
		if _, ok := module.deniedActions[config.site]; ok == false {
			module.deniedActions[config.site] = make([]HttpFunc, 0)
		}
		module.deniedActions[config.site] = append(module.deniedActions[config.site], config.Action)
	}
}

//事件Http  请求开始
func (module *httpModule) serve(thread HttpThread) {
	ctx := httpContext(module, thread)
	if config, ok := module.routers[ctx.Name]; ok {
		ctx.Config = config
		if config.Setting != nil {
			ctx.Setting = config.Setting
		}
	}

	now := time.Now()

	token := ""

	if ctx.siteConfig.Cookie != "" {
		//token直接读，不从Cookie解密读，因为token是直接写的
		if c, e := ctx.request.Cookie(ctx.siteConfig.Cookie); e == nil {
			token = c.Value
		}
	}

	if vv := ctx.Header("Authorization"); vv != "" {
		token = vv
	}

	//验证token
	if token != "" {
		verify, err := mToken.Validate(token)

		if err == nil {
			ctx.token = token
			ctx.verify = verify

			//把payload写入session，直接写
			if ctx.verify.Payload != nil {
				for key, val := range ctx.verify.Payload {
					ctx.sessions[key] = val
				}
			}
		}
	}

	//请求id
	//2022更新为从token获取
	// id := ctx.Cookie(ctx.siteConfig.Cookie)
	if ctx.token == "" {
		//这里要处理，是否颁发空的token，待处理
		//site应该做一个开关，标记是否自动颁发token

		//是否自动颁发token
		if ctx.siteConfig.Issue {
			verify := &Token{}
			token, err := mToken.Sign(verify)
			if err == nil {
				ctx.issue = true
				ctx.token = token
				ctx.verify = verify
			}
		}

	} else {
		//请求的一开始，主要是SESSION处理
		if ctx.sessional(true) {
			//session只在一个token生命周期内有效
			mmm, eee := mSession.Read(ctx.ActId())
			if eee == nil && mmm != nil {
				for k, v := range mmm {
					ctx.sessions[k] = v
				}
			} else {
				//活跃超过1天，就更新一下session
				if vv, ok := ctx.sessions["$alive"].(float64); ok {
					if (now.Unix() - int64(vv)) > 60*60*24 {
						ctx.Session("$alive", now.Unix())
					}
				} else {
					ctx.Session("$alive", now.Unix())
				}
			}
		}
	}

	// 上面执行前置处理 ---------------------------------

	//request拦截器，加入调用列表
	if funcs, ok := module.requestActions[ctx.Site]; ok {
		ctx.next(funcs...)
	}

	ctx.next(module.access)
	ctx.next(module.request)
	ctx.next(module.execute)

	//开始执行
	ctx.Next()

	// 下面执行后 ---------------------------------

	//session写回去
	if ctx.sessional(false) {
		//这样节省SESSION的资源
		if ctx.siteConfig.Expiry > 0 {
			mSession.Write(ctx.ActId(), ctx.sessions, ctx.siteConfig.Expiry)
		} else {
			mSession.Write(ctx.ActId(), ctx.sessions)
		}
	}

	module.response(ctx)
}

//访问控制
func (module *httpModule) access(ctx *Access) {
	config := ctx.module.config.Access

	//允许跨域才处理s
	if config.Cross {

		//三项校验，全部要通过才放行
		origin := ctx.Header("Origin")
		originPassed := false
		if config.Origin == "*" || config.Origin == "" || (len(config.Origins) > 0 && config.Origins[0] == "*") {
			originPassed = true
		} else {
			if origin != "" {
				for _, prefix := range config.Origins {
					if strings.HasPrefix(origin, prefix) {
						originPassed = true
						break
					}
				}
			}
		}
		method := ctx.Header("Access-Control-Request-Method")
		methodPassed := false
		if config.Method == "*" || config.Method == "" || (len(config.Methods) > 0 && config.Methods[0] == "*") {
			methodPassed = true
		} else {
			if method != "" {
				methods := util.Split(method)
				if util.AllinStrings(methods, config.Methods) {
					methodPassed = true
				}
			}
		}

		header := ctx.Header("Access-Control-Request-Headers")
		headerPassed := false

		if config.Header == "*" || config.Header == "" || (len(config.Headers) > 0 && config.Headers[0] == "*") {
			headerPassed = true
		} else {
			if header != "" {
				headers := util.Split(header)
				if util.AllinStrings(headers, config.Headers) {
					headerPassed = true
				}
			}
		}

		if originPassed && methodPassed && headerPassed {
			ctx.Header("Access-Control-Allow-Credentials", "true")
			if origin != "" {
				ctx.Header("Access-Control-Allow-Origin", origin)
			}
			if method != "" {
				ctx.Header("Access-Control-Allow-Methods", method)
			}
			if header != "" {
				ctx.Header("Access-Control-Allow-Headers", header)
				ctx.Header("Access-Control-Expose-Headers", header)
			}

			if ctx.Method == OPTIONS {
				ctx.Text("cross domain access allowed.", http.StatusOK)
				return //中止执行
			}
		}
	}

	ctx.Next()
}

func (module *httpModule) request(ctx *Access) {
	//404么
	if ctx.Name == "" {

		//路由不存在， 找静态文件

		//静态文件放在这里处理
		isDir := false
		file := ""
		sitePath := path.Join(module.config.Static, ctx.Site, ctx.Path)
		if fi, err := os.Stat(sitePath); err == nil {
			isDir = fi.IsDir()
			file = sitePath
		} else {
			sharedPath := path.Join(module.config.Static, module.config.Shared, ctx.Path)
			if fi, err := os.Stat(sharedPath); err == nil {
				isDir = fi.IsDir()
				file = sharedPath
			}
		}

		//如果是目录，要遍历默认文档
		if isDir {
			tempFile := file
			file = ""
			if len(module.config.Defaults) == 0 {
				file = ""
			} else {
				for _, doc := range module.config.Defaults {
					docPath := path.Join(tempFile, doc)
					if fi, err := os.Stat(docPath); err == nil && fi.IsDir() == false {
						file = docPath
						break
					}
				}
			}
		}

		if file != "" {
			ctx.File(file, "")
		} else {
			ctx.Found()
		}

	} else {

		//表单这里处理，这样会在 requestFilter之前处理好
		if res := ctx.formHandler(); res.Fail() {
			ctx.lastError = res
			module.failed(ctx)
		} else {
			if res := ctx.clientHandler(); res.Fail() {
				ctx.lastError = res
				module.failed(ctx)
			} else {
				if res := ctx.argsHandler(); res.Fail() {
					ctx.lastError = res
					module.failed(ctx)
				} else {
					if res := ctx.authHandler(); res.Fail() {
						ctx.lastError = res
						module.denied(ctx)
					} else {
						if res := ctx.signHandler(); res.Fail() {
							ctx.lastError = res
							module.denied(ctx)
						} else {
							if res := ctx.itemHandler(); res.Fail() {
								ctx.lastError = res
								module.failed(ctx)
							} else {
								//往下走，到execute
								ctx.Next()
							}
						}
					}
				}
			}
		}
	}
}

//事件执行，调用action的地方
func (module *httpModule) execute(ctx *Access) {
	ctx.clear()

	//executeFilters
	if funcs, ok := module.executeActions[ctx.Site]; ok {
		ctx.next(funcs...)
	}

	// //actions
	if funcs, ok := module.routerActions[ctx.Name]; ok {
		ctx.next(funcs...)
	}

	ctx.Next()
}

//事件执行，调用action的地方
func (module *httpModule) response(ctx *Access) {
	//响应前清空执行线
	ctx.clear()

	//response拦截器，加入调用列表
	if funcs, ok := module.responseActions[ctx.Site]; ok {
		ctx.next(funcs...)
	}

	//最终的body处理，加入执行线
	ctx.next(module.body)

	ctx.Next()
}

//最终响应
func (module *httpModule) body(ctx *Access) {

	if ctx.Code <= 0 {
		ctx.Code = http.StatusOK
	}

	//设置cookies, headers

	//cookie超时时间
	//为了极致的性能，可以在启动的时候先解析好
	// var maxage time.Duration
	// if ctx.siteConfig.MaxAge != "" {
	// 	td, err := util.ParseDuration(ctx.siteConfig.MaxAge)
	// 	if err == nil {
	// 		maxage = td
	// 	}
	// }

	for _, v := range ctx.cookies {
		v.HttpOnly = true

		if ctx.Domain != "" {
			v.Domain = ctx.Domain
		}
		if ctx.siteConfig.MaxAge > 0 {
			v.MaxAge = int(ctx.siteConfig.MaxAge.Seconds())
		}

		//这里统一加密
		if vvv, err := TextEncrypt(v.Value); err == nil {
			v.Value = vvv
		}

		http.SetCookie(ctx.response, &v)
	}

	//最终响应之前，判断是否需要颁发token
	if ctx.issue {
		//需要站点配置中指定了cookie的，才自动写入cookie
		if ctx.siteConfig.Cookie != "" {
			cookie := http.Cookie{Name: ctx.siteConfig.Cookie, Value: ctx.token, HttpOnly: true}
			if ctx.Domain != "" {
				cookie.Domain = ctx.Domain
			}
			http.SetCookie(ctx.response, &cookie)
		}
	}

	for k, v := range ctx.headers {
		ctx.response.Header().Set(k, v)
	}

	switch body := ctx.Body.(type) {
	case httpGotoBody:
		module.bodyGoto(ctx, body)
	case httpTextBody:
		module.bodyText(ctx, body)
	case httpHtmlBody:
		module.bodyHtml(ctx, body)
	case httpScriptBody:
		module.bodyScript(ctx, body)
	case httpJsonBody:
		module.bodyJson(ctx, body)
	case httpJsonpBody:
		module.bodyJsonp(ctx, body)
	case httpApiBody:
		module.bodyApi(ctx, body)
	case httpXmlBody:
		module.bodyXml(ctx, body)
	case httpFileBody:
		module.bodyFile(ctx, body)
	case httpDownBody:
		module.bodyDown(ctx, body)
	case httpBufferBody:
		module.bodyBuffer(ctx, body)
	case httpViewBody:
		module.bodyView(ctx, body)
	case httpProxyBody:
		module.bodyProxy(ctx, body)
	default:
		module.bodyDefault(ctx)
	}

	//最终响应前做清理工作
	ctx.terminal()
}
func (module *httpModule) bodyDefault(ctx *Access) {
	ctx.Code = http.StatusNotFound
	http.NotFound(ctx.response, ctx.request)
	ctx.thread.Finish()
}
func (module *httpModule) bodyGoto(ctx *Access, body httpGotoBody) {
	http.Redirect(ctx.response, ctx.request, body.url, http.StatusFound)
	ctx.thread.Finish()
}
func (module *httpModule) bodyText(ctx *Access, body httpTextBody) {
	res := ctx.response

	if ctx.Type == "" {
		ctx.Type = "text"
	}

	ctx.Type = mBasic.Mimetype(ctx.Type, "text/explain")
	res.Header().Set("Content-Type", fmt.Sprintf("%v; charset=%v", ctx.Type, ctx.Charset()))

	res.WriteHeader(ctx.Code)
	fmt.Fprint(res, body.text)

	ctx.thread.Finish()
}
func (module *httpModule) bodyHtml(ctx *Access, body httpHtmlBody) {
	res := ctx.response

	if ctx.Type == "" {
		ctx.Type = "html"
	}

	ctx.Type = mBasic.Mimetype(ctx.Type, "text/html")
	res.Header().Set("Content-Type", fmt.Sprintf("%v; charset=%v", ctx.Type, ctx.Charset()))

	res.WriteHeader(ctx.Code)
	fmt.Fprint(res, body.html)

	ctx.thread.Finish()
}
func (module *httpModule) bodyScript(ctx *Access, body httpScriptBody) {
	res := ctx.response

	ctx.Type = mBasic.Mimetype(ctx.Type, "application/script")
	res.Header().Set("Content-Type", fmt.Sprintf("%v; charset=%v", ctx.Type, ctx.Charset()))

	res.WriteHeader(ctx.Code)
	fmt.Fprint(res, body.script)
	ctx.thread.Finish()
}
func (module *httpModule) bodyJson(ctx *Access, body httpJsonBody) {
	res := ctx.response

	bytes, err := MarshalJSON(body.json)
	if err != nil {
		//要不要发到统一的错误ctx.Error那里？再走一遍
		http.Error(res, err.Error(), http.StatusInternalServerError)
	} else {

		ctx.Type = mBasic.Mimetype(ctx.Type, "text/json")
		res.Header().Set("Content-Type", fmt.Sprintf("%v; charset=%v", ctx.Type, ctx.Charset()))

		res.WriteHeader(ctx.Code)
		fmt.Fprint(res, string(bytes))
	}
	ctx.thread.Finish()
}
func (module *httpModule) bodyJsonp(ctx *Access, body httpJsonpBody) {
	res := ctx.response

	bytes, err := MarshalJSON(body.json)
	if err != nil {
		//要不要发到统一的错误ctx.Error那里？再走一遍
		http.Error(res, err.Error(), http.StatusInternalServerError)
	} else {

		ctx.Type = Mimetype(ctx.Type, "application/script")
		res.Header().Set("Content-Type", fmt.Sprintf("%v; charset=%v", ctx.Type, ctx.Charset()))

		res.WriteHeader(ctx.Code)
		fmt.Fprint(res, fmt.Sprintf("%s(%s);", body.callback, string(bytes)))
	}
	ctx.thread.Finish()
}
func (module *httpModule) bodyXml(ctx *Access, body httpXmlBody) {
	res := ctx.response

	if ctx.Type == "" {
		ctx.Type = "xml"
	}

	content := ""
	if vv, ok := body.xml.(string); ok {
		content = vv
	} else {
		bytes, err := XMLMarshal(body.xml)
		if err == nil {
			content = string(bytes)
		}
	}

	if content == "" {
		http.Error(res, "解析xml失败", http.StatusInternalServerError)
	} else {
		ctx.Type = mBasic.Mimetype(ctx.Type, "text/xml")
		res.Header().Set("Content-Type", fmt.Sprintf("%v; charset=%v", ctx.Type, ctx.Charset()))

		res.WriteHeader(ctx.Code)
		fmt.Fprint(res, content)
	}
	ctx.thread.Finish()
}
func (module *httpModule) bodyApi(ctx *Access, body httpApiBody) {

	json := Map{
		"code": body.code,
		"time": time.Now().Unix(),
	}

	//开启自动下发，才下发
	if ctx.issue {
		json["token"] = ctx.token
	}

	if body.text != "" {
		json["text"] = body.text
	}

	if body.data != nil {
		if body.code != 0 {
			json["data"] = body.data
		} else {
			//如果body.code == 0 才成功，才需要按套路来输出
			//否则，传过来什么，就直接输出什么

			crypto := ctx.siteConfig.Confuse
			if vv, ok := ctx.Setting["confuse"].(bool); ok && vv == false {
				crypto = ""
			}
			if vv, ok := ctx.Setting["encode"].(bool); ok && vv == false {
				crypto = ""
			}
			if vv, ok := ctx.Setting["plain"].(bool); ok && vv {
				crypto = ""
			}
			//待完善，
			// if vv := ctx.Header("debug"); vv != "" && vv == Secret {
			if vv := ctx.Header("debug"); vv != "" {
				crypto = ""
			}

			tempConfig := Vars{
				"data": Var{
					Type: "json", Required: true, Encode: crypto,
				},
			}

			if ctx.Config.Data != nil {
				tempConfig = Vars{
					"data": Var{
						Type: "json", Required: true, Encode: crypto,
						Children: ctx.Config.Data,
					},
				}
			}
			tempData := Map{
				"data": body.data,
			}

			val := Map{}
			res := mBasic.Mapping(tempConfig, tempData, val, false, false, ctx.context)

			if res == nil || res.OK() {
				//兼容Res接口版
				//处理后的data
				// if body.code == 0 {
				// 	ctx.Code = http.StatusOK
				// }
				json["data"] = val["data"]
			} else {
				json["code"] = mBasic.StateCode(res.Text())
				json["text"] = ctx.String(res.Text(), res.Args()...)
			}
		}
	}

	//转到jsonbody去处理
	module.bodyJson(ctx, httpJsonBody{json})
}

func (module *httpModule) bodyFile(ctx *Access, body httpFileBody) {
	req, res := ctx.request, ctx.response

	//文件类型
	if ctx.Type != "file" {
		ctx.Type = mBasic.Mimetype(ctx.Type, "application/octet-stream")
		res.Header().Set("Content-Type", fmt.Sprintf("%v; charset=%v", ctx.Type, ctx.Charset()))
	}
	//加入自定义文件名
	if body.name != "" {
		res.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%v;", url.QueryEscape(body.name)))
	}

	http.ServeFile(res, req, body.file)
	ctx.thread.Finish()
}
func (module *httpModule) bodyDown(ctx *Access, body httpDownBody) {
	res := ctx.response

	if ctx.Type == "" {
		ctx.Type = "file"
	}

	ctx.Type = mBasic.Mimetype(ctx.Type, "application/octet-stream")
	res.Header().Set("Content-Type", fmt.Sprintf("%v; charset=%v", ctx.Type, ctx.Charset()))
	//加入自定义文件名
	if body.name != "" {
		res.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%v;", url.QueryEscape(body.name)))
	}

	res.WriteHeader(ctx.Code)
	res.Write(body.bytes)

	ctx.thread.Finish()
}
func (module *httpModule) bodyBuffer(ctx *Access, body httpBufferBody) {
	res := ctx.response

	if ctx.Type == "" {
		ctx.Type = "file"
	}

	ctx.Type = mBasic.Mimetype(ctx.Type, "application/octet-stream")
	res.Header().Set("Content-Type", fmt.Sprintf("%v; charset=%v", ctx.Type, ctx.Charset()))
	//加入自定义文件名
	if body.name != "" {
		res.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%v;", url.QueryEscape(body.name)))
	}

	res.WriteHeader(ctx.Code)
	_, err := io.Copy(res, body.buffer)
	//bytes,err := ioutil.ReadAll(body.buffer)
	if err == nil {
		http.Error(res, "read buffer error", http.StatusInternalServerError)
	}
	body.buffer.Close()
	ctx.thread.Finish()
}
func (module *httpModule) bodyView(ctx *Access, body httpViewBody) {
	res := ctx.response

	viewdata := Map{
		"args": ctx.Args, "sign": ctx.Sign,
		"config": ctx.siteConfig, "setting": Setting,
		"local": ctx.Local, "data": ctx.Data, "model": body.model,
	}

	helpers := module.viewHelpers(ctx)

	html, err := mView.Parse(ViewBody{
		Helpers: helpers,
		View:    body.view, Data: viewdata,
		Site: ctx.Site, Lang: ctx.Lang(), Zone: ctx.Zone(),
	})

	if err != nil {
		http.Error(res, ctx.String(err.Error()), 500)
	} else {
		mime := mBasic.Mimetype(ctx.Type, "text/html")
		res.Header().Set("Content-Type", fmt.Sprintf("%v; charset=%v", mime, ctx.Charset()))
		res.WriteHeader(ctx.Code)
		fmt.Fprint(res, html)
	}

	ctx.thread.Finish()
}
func (module *httpModule) viewHelpers(ctx *Access) Map {
	//系统内置的helper
	helpers := Map{
		"route":    ctx.Url.Route,
		"browse":   ctx.Url.Browse,
		"preview":  ctx.Url.Preview,
		"download": ctx.Url.Download,
		"backurl":  ctx.Url.Back,
		"lasturl":  ctx.Url.Last,
		"siteurl": func(name string, paths ...string) string {
			path := ""
			if len(paths) > 0 {
				path = paths[0]
			}
			return ctx.Url.Site(name, path)
		},

		"lang": func() string {
			return ctx.Lang()
		},
		"zone": func() *time.Location {
			return ctx.Zone()
		},
		"timezone": func() string {
			return ctx.String(ctx.Zone().String())
		},
		"format": func(format string, args ...interface{}) string {
			//支持一下显示时间
			if len(args) == 1 {
				if args[0] == nil {
					return format
				} else if ttt, ok := args[0].(time.Time); ok {
					zoneTime := ttt.In(ctx.Zone())
					return zoneTime.Format(format)
				} else if ttt, ok := args[0].(int64); ok {
					//时间戳是大于1971年是, 千万级, 2016年就是10亿级了
					if ttt >= int64(31507200) && ttt <= int64(31507200000) {
						ttt := time.Unix(ttt, 0)
						zoneTime := ttt.In(ctx.Zone())
						sss := zoneTime.Format(format)
						if strings.HasPrefix(sss, "%") == false || format != sss {
							return sss
						}
					}
				}
			}
			return fmt.Sprintf(format, args...)
		},

		"signed": func(key string) bool {
			return ctx.Signed(key)
		},
		"signal": func(key string) string {
			return ctx.Signal(key)
		},
		"signer": func(key string) string {
			return ctx.Signer(key)
		},
		"string": func(key string, args ...Any) string {
			return ctx.String(key, args...)
		},
		"option": func(name, field string, v Any) Any {
			value := fmt.Sprintf("%v", v)
			//多语言支持
			//key=enum.name.file.value
			langkey := fmt.Sprintf("option_%s_%s_%s", name, field, value)
			langval := ctx.String(langkey)
			if langkey != langval {
				return langval
			} else {
				return mData.Option(name, field, value)
				// if vv, ok := enums[value].(string); ok {
				// 	return vv
				// }
				// return value
			}
		},
	}

	for k, v := range mView.actions {
		if f, ok := v.(func(*Access, ...Any) Any); ok {
			helpers[k] = func(args ...Any) Any {
				return f(ctx, args...)
			}
		} else {
			helpers[k] = v
		}
	}

	return helpers
}

func (module *httpModule) bodyProxy(ctx *Access, body httpProxyBody) {
	req := ctx.request
	res := ctx.response

	target := body.url
	targetQuery := body.url.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}

	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(res, req)

	ctx.thread.Finish()
}

// func (module *httpModule) sessionKey(ctx *Access) string {
// 	format := "http_%s"
// 	if vv,ok := CONFIG.mSession.Format[bHTTP].(string); ok && vv != "" {
// 		format = vv
// 	}
// 	return fmt.Sprintf(format, ctx.ActId())
// }

//事件handler,找不到
func (module *httpModule) found(ctx *Access) {
	ctx.clear()

	if ctx.Code <= 0 {
		ctx.Code = http.StatusNotFound
	}

	//如果有自定义的错误处理，加入调用列表
	// funcs := ctx.funcing("found")
	if ctx.Config.Found != nil {
		ctx.next(ctx.Config.Found)
	}

	//把处理器加入调用列表
	if funcs, ok := module.foundActions[ctx.Site]; ok {
		ctx.next(funcs...)
	}

	//加入默认的错误处理
	ctx.next(module.foundDefaultHandler)
	ctx.Next()
}

//最终还是由response处理
func (module *httpModule) foundDefaultHandler(ctx *Access) {
	found := textResult("_found")
	if res := ctx.Result(); res != nil {
		found = res
	}

	ctx.Code = http.StatusNotFound

	//如果是ajax访问，返回JSON对应，要不然返回页面
	if ctx.Ajax {
		ctx.Answer(found)
	} else {
		ctx.Text("http not found")
	}
}

//事件handler,错误的处理
func (module *httpModule) error(ctx *Access) {
	ctx.clear()

	if ctx.Code <= 0 {
		ctx.Code = http.StatusInternalServerError
	}

	//如果有自定义的错误处理，加入调用列表
	// funcs := ctx.funcing("error")
	if ctx.Config.Error != nil {
		ctx.next(ctx.Config.Error)
	}

	//把错误处理器加入调用列表
	if funcs, ok := module.errorActions[ctx.Site]; ok {
		ctx.next(funcs...)
	}

	//加入默认的错误处理
	ctx.next(module.errorDefaultHandler)
	ctx.Next()
}

//最终还是由response处理
func (module *httpModule) errorDefaultHandler(ctx *Access) {
	err := textResult("_error")
	if res := ctx.Result(); res != nil {
		err = res
	}

	ctx.Code = http.StatusInternalServerError

	if ctx.Ajax {
		ctx.Answer(err)
	} else {

		ctx.Data["status"] = ctx.Code
		ctx.Data["error"] = Map{
			"code": mBasic.StateCode(err.Text()),
			"text": ctx.String(err.Text(), err.Args()...),
		}

		ctx.View("error")
	}
}

//事件handler,失败处理，主要是args失败
func (module *httpModule) failed(ctx *Access) {
	ctx.clear()

	if ctx.Code <= 0 {
		ctx.Code = http.StatusBadRequest
	}

	//如果有自定义的失败处理，加入调用列表
	// funcs := ctx.funcing("failed")
	// ctx.next(funcs...)
	if ctx.Config.Failed != nil {
		ctx.next(ctx.Config.Failed)
	}

	//把失败处理器加入调用列表
	if funcs, ok := module.failedActions[ctx.Site]; ok {
		ctx.next(funcs...)
	}

	//加入默认的错误处理
	ctx.next(module.failedDefaultHandler)
	ctx.Next()
}

//最终还是由response处理
func (module *httpModule) failedDefaultHandler(ctx *Access) {
	failed := textResult("_failed")
	if res := ctx.Result(); res != nil {
		failed = res
	}

	ctx.Code = http.StatusBadRequest

	if ctx.Ajax {
		ctx.Answer(failed)
	} else {
		ctx.Alert(failed)
	}
}

//事件handler,失败处理，主要是args失败
func (module *httpModule) denied(ctx *Access) {
	ctx.clear()

	if ctx.Code <= 0 {
		ctx.Code = http.StatusUnauthorized
	}

	//如果有自定义的失败处理，加入调用列表
	// funcs := ctx.funcing("denied")
	// ctx.next(funcs...)
	if ctx.Config.Denied != nil {
		ctx.next(ctx.Config.Denied)
	}

	//把失败处理器加入调用列表
	if funcs, ok := module.deniedActions[ctx.Site]; ok {
		ctx.next(funcs...)
	}

	//加入默认的错误处理
	ctx.next(module.deniedDefaultHandler)
	ctx.Next()
}

//最终还是由response处理
//如果是ajax。返回拒绝
//如果不是， 返回一个脚本提示
func (module *httpModule) deniedDefaultHandler(ctx *Access) {
	denied := textResult("_denied")
	if res := ctx.Result(); res != nil {
		denied = res
	}

	ctx.Code = http.StatusUnauthorized

	if ctx.Ajax {
		ctx.Answer(denied)
	} else {
		ctx.Alert(denied)
	}
}

func (module *httpModule) newSite(name string, roots ...string) *httpSite {
	root := ""
	if len(roots) > 0 {
		root = strings.TrimRight(roots[0], "/")
	}
	return &httpSite{name, root}
}

func (site *httpSite) Route(name string, args ...Map) string {
	realName := fmt.Sprintf("%s.%s", site.name, name)
	return mHttp.url.Route(realName, args...)
}

// Register 注册中心
func (site *httpSite) Register(name string, value Any, overrides ...bool) {
	override := true
	if len(overrides) > 0 {
		override = overrides[0]
	}

	key := fmt.Sprintf("%s.%s", site.name, name)

	if router, ok := value.(Router); ok {
		if site.root != "" {
			if router.Uri != "" {
				router.Uri = site.root + router.Uri
			}
			if router.Uris != nil {
				for i, uri := range router.Uris {
					router.Uris[i] = site.root + uri
				}
			}
		}
		mHttp.Register(key, router, override)
	} else {
		mHttp.Register(key, value, override)
	}
}

//语法糖

func Site(name string, roots ...string) *httpSite {
	return mHttp.newSite(name, roots...)
}
func Route(name string, args ...Map) string {
	return mHttp.url.Route(name, args...)
}
func Routers(sites ...string) map[string]Router {
	return mHttp.Routers(sites...)
}
