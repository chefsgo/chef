package chef

import (
	"fmt"
	"strings"
	"sync"

	. "github.com/chefsgo/base"
)

var (
	mBasic = &basicModule{
		langs:  make(Lang, 0),
		states: make(map[string]State, 0),
	}
)

type (
	Lang  = map[string]string
	State struct {
		Code int
		Text string
	}

	// basicModule 是基础模块
	// 主要用功能是 状态、多语言字串、MIME类型、正则表达式等等
	basicModule struct {
		mutex sync.Mutex

		//存储所有状态定义
		states map[string]State
		//记录所有字串的定义
		langs Lang
	}
)

func (module *basicModule) configure(config Map) {
	// fmt.Println("basic configured")
}

func (module *basicModule) register(name string, value Any, override bool) {
	switch val := value.(type) {
	case Lang:
		module.Lang(name, val, override)
	case State:
		module.State(name, val, override)
		// case Mime:
		// 	module.Mime(key, val, overrides...)
		// case Regular:
		// 	module.Regular(key, val, overrides...)
		// case Type:
		// 	module.Type(key, val, overrides...)
		// case Crypto:
		// 	module.Crypto(key, val, overrides...)
	}

	// fmt.Println("basic registered", name)
}

func (module *basicModule) initialize() {
	// fmt.Println("basic initialized")
}

func (module *basicModule) launch() {
	// fmt.Println("basic launched")
}

func (module *basicModule) terminate() {
	// fmt.Println("basic terminated")
}

// Lang 注册多语言字串
// 语言lang为做前缀，全部写成同一个集合中
func (module *basicModule) Lang(lang string, config Lang, override bool) {
	for k, v := range config {
		//所有k统一把点替换为下划线，为加载语言资源文件时方便
		key := fmt.Sprintf("%v.%v", lang, strings.Replace(k, ".", "_", -1))
		if override {
			module.langs[key] = v
		} else {
			if _, ok := module.langs[k]; ok == false {
				module.langs[key] = v
			}
		}
	}
}

// State 注册状态
// 如果State携带了String，则自动注册成默认语言字串
func (module *basicModule) State(name string, config State, override bool) {
	alias := make([]string, 0)
	if name != "" {
		alias = append(alias, name)
	}

	if override {
		module.states[name] = config
	} else {
		if _, ok := module.states[name]; ok == false {
			module.states[name] = config
		}
	}

	//自动注册默认的语言字串
	if config.Text != "" {
		module.Lang(DEFAULT, Lang{name: config.Text}, override)
	}
}
