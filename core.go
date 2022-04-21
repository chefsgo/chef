package chef

import (
	"io/ioutil"
	"os"
	"os/signal"
	"sync"
	"syscall"

	. "github.com/chefsgo/base"
)

var (
	core = &kernel{
		parsed:      false,
		initialized: false,
		launched:    false,
		config: config{
			name: CHEF, role: CHEF, version: "0.0.0",
			setting: Map{},
		},

		names:   make([]string, 0),
		modules: make(map[string]module, 0),
	}
)

type (
	kernel struct {
		parsed      bool
		initialized bool
		launched    bool

		mutex   sync.RWMutex
		config  config
		names   []string
		modules map[string]module
	}
	config struct {
		name    string
		role    string
		version string
		setting Map
	}
	module interface {
		register(key string, val Any, override bool)
		configure(Map)
		initialize()
		launch()
		terminate()
	}
)

// setting 获取setting
func (k *kernel) setting() Map {
	k.mutex.RLock()
	defer k.mutex.RUnlock()

	// 为了线程安全，为了避免外部修改，这里复制一份
	// 需要不需要深度复制？二层以下的map,[]map会不会因为引用的关系被修改？
	// 以上需要测试go的特性再处理，但是通常不会有什么问题，待优化
	setting := Map{}
	for k, v := range k.config.setting {
		setting[k] = v
	}

	return setting
}

// loader 把模块注册到core
// 遍历所有已经注册过的模块，避免重复注册
func (k *kernel) loader(name string, m module) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	_, exists := k.modules[name]
	if exists == false {
		k.names = append(k.names, name)
	}
	k.modules[name] = m
}

// register 遍历所有模块调用注册
// 动态参数，以支持以下几种可能性
// 并且此方法兼容configure，为各模块加载默认配置
// name, config, override 包括name的注册
// config, override	不包括name的注册，比如  langstring, regular, mimetype等
// configs
// func (k *kernel) register(name string, config Any, overrides ...bool) {
func (k *kernel) register(regs ...Any) {
	name := ""
	configs := make([]Any, 0)
	override := true

	for _, reg := range regs {
		switch vvv := reg.(type) {
		case string:
			name = vvv
		case bool:
			override = vvv
		default:
			configs = append(configs, vvv)
		}
	}

	for _, cfg := range configs {
		if mmm, ok := cfg.(Map); ok {
			// 兼容所有模块的配置注册
			k.configure(mmm)
		} else {
			//实际注册到各模块
			for _, mod := range k.modules {
				mod.register(name, cfg, override)
			}
		}
	}

}

// parse 解析启动参数，参数有以下几个来源
// 1 命令行参数，直接传在启动命令后面
// 2 环境变量，各种参数单独传过来
// 3 环境变量，就像命令行参数一样，整个传过来
// 主要是方便在docker中启动，或是其它容器
// 以上功能待处理
func (k *kernel) parse() {
	if k.parsed {
		return
	}

	args := os.Args

	// 定义一个文件列表，尝试读取配置
	files := []string{
		"config.toml", "chefgo.toml", "chef.toml",
		"config.conf", "chefgo.conf", "chef.conf",
	}
	// 如果参数个数为1，表示没有传参数，使用文件名
	if len(args) == 1 {
		base := getBaseWithoutExt(args[0])
		files = append([]string{
			base + ".toml", base + ".conf",
		}, files...)
	}
	// 如果参数个数为2，就是指定了配置文件
	if len(args) == 2 {
		files = append([]string{args[1]}, files...)
	}

	// 遍历文件
	for _, file := range files {
		// 判断文件是否存在
		if _, err := os.Stat(file); err == nil {
			// 读取文件
			bytes, err := ioutil.ReadFile(file)
			if err == nil {
				// 加载配置，并中断循环，只读取第一个读到的文件
				config := parseTOML(string(bytes))
				core.configure(config)
				break
			}
		}
	}

	//从环境变量中读取

	// // 参数大于2个，就解析参数
	// if len(args) > 2 {
	// 	var name string
	// 	var node string
	// 	var bind string
	// 	var key string
	// 	var tags []string
	// 	var join []string

	// 	flag.StringVar(&name, "name", "chef", "cluster name")
	// 	flag.StringVar(&node, "node", "test", "node name")
	// 	flag.StringVar(&bind, "bind", "0.0.0.0:3000", "address to bind listeners to")
	// 	flag.StringVar(&key, "key", "", "encryption key")
	// 	flag.Var((*flagSlice)(&tags), "tag", "tag pair, specified as key=value")
	// 	flag.Var((*flagSlice)(&join), "join", "address of agent to join on startup")

	// 	flag.Parse()

	// }

	//这里要连接集群

	// core.cluster.connect()
}

// identify 声明当前节点的身份和版本
// role 当前节点的角色/身份
// version 编译的版本，建议每次发布时更新版本
// 通常在一个项目中会有多个不同模块（角色），每个模块可能会运行N个节点
// 在集群中标明当前节点的身份和版本，方便管理集群
func (k *kernel) identify(role string, versions ...string) {
	k.config.role = role
	if len(versions) > 0 {
		k.config.version = versions[0]
	}
}

// configure 为所有模块加载配置
// 此方法有可能会被多次调用，解析文件后可调用
// 从配置中心获取到配置后，也会调用
func (k *kernel) configure(config Map) {
	if config == nil {
		return
	}

	// 如果已经初始化就不让修改了
	if k.initialized || k.launched {
		return
	}

	//处理core中的配置
	if name, ok := config["name"].(string); ok && name != k.config.name {
		if k.config.name == k.config.role {
			k.config.name = name
			k.config.role = name
		} else {
			k.config.name = name
		}
	}
	if role, ok := config["role"].(string); ok && role != k.config.name {
		k.config.role = role
	}
	if version, ok := config["version"].(string); ok && version != k.config.version {
		k.config.version = version
	}

	// 配置写到配置中
	if setting, ok := config["setting"].(Map); ok {
		for key, val := range setting {
			k.config.setting[key] = val
		}
	}

	// 把配置下发到各个模块
	for _, mod := range k.modules {
		mod.configure(config)
	}
}

// initialize 初始化所有模块
func (k *kernel) initialize() {
	if k.initialized {
		return
	}
	for _, name := range k.names {
		mod := k.modules[name]
		mod.initialize()
	}
	k.initialized = true
}

// launch 启动所有模块
// 只有部分模块是需要启动的，比如HTTP
func (k *kernel) launch() {
	if k.launched {
		return
	}
	for _, mod := range k.modules {
		mod.launch()
	}
	k.launched = true

	Debug("wf么鬼东西啊")
	Warning("wf么鬼东西啊")
	// Debug("wf么鬼东西啊")
	// Debug("wf么鬼东西啊")
}

// waiting 等待系统退出信号
// 为了程序做好退出前的善后工作，优雅的退出程序
func (k *kernel) waiting() {
	// 待处理，加入自己的退出信号
	// 并开发 chef.Stop() 给外部调用
	waiter := make(chan os.Signal, 1)
	signal.Notify(waiter, os.Kill, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-waiter
}

// terminate 终止结束所有模块
// 终止顺序需要和初始化顺序相反以保证各模块依赖
func (k *kernel) terminate() {
	for i := len(k.names) - 1; i >= 0; i-- {
		name := k.names[i]
		mod := k.modules[name]
		mod.terminate()
	}
	k.launched = false
}

//将各种模块按顺序注册到核心
func init() {
	core.loader("basic", mBasic)
	core.loader("log", mLog)
	core.loader("mutex", mMutex)
}
