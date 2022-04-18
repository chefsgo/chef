package chef

import (
	. "github.com/chef-go/base"
)

// Register 注册各种内容
func Register(name string, config Any, overrides ...bool) {
	core.register(name, config, overrides...)
}

// Config 开放修改默认配置
// 比如，在代码中就可以设置一些默认配置
// 这样就可以最大化的减少配置文件的依赖
func Config(cfg Map) {
	core.configure(cfg)
}

func Setting() Map {
	return core.setting()
}

// Ready 准备好各模块
// 当你需要写一个临时程序，但是又需要使用程序里的代码
// 比如，导入老数据，整理文件或是数据，临时的采集程序等等
// 就可以在临时代码中，调用chef.Ready()，然后做你需要做的事情
func Ready() {
	core.parse()
	core.initialize()
}

// Go 直接开跑
func Go() {
	core.parse()
	core.initialize()
	core.launch()
	core.waiting()
	core.terminate()
}
