package chef

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	. "github.com/chefsgo/base"
)

type (
	chef struct {
		parsed      bool
		initialized bool
		connected   bool
		launched    bool

		mutex   sync.RWMutex
		config  config
		modules []Module
	}
	config struct {
		// name 项目名称
		name string

		// role 节点角色
		role string

		mode env

		// version 节点版本
		version string

		// config 节点配置key
		// 一般是为了远程获取配置
		config string

		// setting 设置，主要是自定义的setting
		// 实际业务代码中一般需要用的配置
		setting Map
	}
)

// setting 获取setting
func (this *chef) setting() Map {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	// 为了线程安全，为了避免外部修改，这里复制一份
	// 需要不需要深度复制？二层以下的map,[]map会不会因为引用的关系被修改？
	// 以上需要测试go的特性再处理，但是通常不会有什么问题，待优化
	setting := Map{}
	for k, v := range this.config.setting {
		setting[k] = v
	}

	return setting
}

// loader 把模块注册到core
// 遍历所有已经注册过的模块，避免重复注册
func (this *chef) loader(mod Module) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.modules = append(this.modules, mod)
}

// register 遍历所有模块调用注册
// 动态参数，以支持以下几种可能性
// 并且此方法兼容configure，为各模块加载默认配置
// name, config, override 包括name的注册
// config, override	不包括name的注册，比如  langstring, regular, mimetype等
// configs
// func (k *chef) register(name string, config Any, overrides ...bool) {
func (k *chef) register(regs ...Any) {
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
		} else if mod, ok := cfg.(Module); ok {
			// 兼容所有模块的配置注册
			k.loader(mod)
		} else {
			//实际注册到各模块
			for _, mod := range k.modules {
				mod.Register(name, cfg, override)
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
func (k *chef) parse() {
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

	// 待处理，从环境变量，和命令行读取配置

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
func (k *chef) identify(role string, versions ...string) {
	k.config.role = role
	if len(versions) > 0 {
		k.config.version = versions[0]
	}
}

// configure 为所有模块加载配置
// 此方法有可能会被多次调用，解析文件后可调用
// 从配置中心获取到配置后，也会调用
func (k *chef) configure(config Map) {
	if config == nil {
		return
	}

	// 如果已经初始化就不让修改了
	if k.initialized || k.launched {
		return
	}

	//注意，集群模块最先处理

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
	if mode, ok := config["mode"].(string); ok {
		mode = strings.ToLower(mode)
		if mode == "t" || mode == "test" || mode == "testing" {
			k.config.mode = testing
		} else if mode == "p" || mode == "prod" || mode == "production" {
			k.config.mode = production
		} else {
			k.config.mode = developing
		}
	}

	// 配置写到配置中
	if setting, ok := config["setting"].(Map); ok {
		for key, val := range setting {
			k.config.setting[key] = val
		}
	}

	// 把配置下发到各个模块
	for _, mod := range k.modules {
		mod.Configure(config)
	}
}

// cluster 独立运行集群
func (k *chef) cluster() {
	// mCluster.Initialize()
	// mCluster.Launch()
}

// initialize 初始化所有模块
func (k *chef) initialize() {
	if k.initialized {
		return
	}
	for _, mod := range k.modules {
		mod.Initialize()
	}
	k.initialized = true
}

// connect
func (k *chef) connect() {
	if k.connected {
		return
	}
	for _, mod := range k.modules {
		mod.Connect()
	}
	k.connected = true
}

// launch 启动所有模块
// 只有部分模块是需要启动的，比如HTTP
func (k *chef) launch() {
	if k.launched {
		return
	}
	for _, mod := range k.modules {
		mod.Launch()
	}

	//这里是触发器，异步
	Trigger(StartTrigger)

	k.launched = true

	if k.config.name == k.config.role || k.config.role == "" {
		log.Println(fmt.Sprintf("%s %s-%s is running", CHEFSGO, k.config.name, k.config.version))
	} else {
		log.Println(fmt.Sprintf("%s %s-%s-%s is running", CHEFSGO, k.config.name, k.config.role, k.config.version))
	}
}

// waiting 等待系统退出信号
// 为了程序做好退出前的善后工作，优雅的退出程序
func (k *chef) waiting() {
	// 待处理，加入自己的退出信号
	// 并开发 chef.Stop() 给外部调用
	waiter := make(chan os.Signal, 1)
	signal.Notify(waiter, os.Kill, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-waiter
}

// terminate 终止结束所有模块
// 终止顺序需要和初始化顺序相反以保证各模块依赖
func (k *chef) terminate() {

	//停止前触发器，同步
	Execute(StopTrigger)

	for i := len(k.modules) - 1; i >= 0; i-- {
		mod := k.modules[i]
		mod.Terminate()
	}
	k.launched = false

	if k.config.name == k.config.role || k.config.role == "" {
		log.Println(fmt.Sprintf("%s %s-%s stopted", CHEFSGO, k.config.name, k.config.version))
	} else {
		log.Println(fmt.Sprintf("%s %s-%s-%s stopted", CHEFSGO, k.config.name, k.config.role, k.config.version))
	}
}
