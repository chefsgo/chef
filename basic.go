package chef

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	. "github.com/chefsgo/base"
)

var (
	mBasic = &basicModule{
		states:   make(State, 0),
		langs:    make(Lang, 0),
		mimes:    make(MIME, 0),
		regulars: make(Regular, 0),
		types:    make(map[string]Type, 0),
	}
)

type (

	// 注意，以下几个类型，不能使用 xxx = map[xxx]yy 的方法定义
	// 因为无法使用.(type)来断言类型

	// State 状态定义，方便注册
	State map[string]int

	// Lang 自定义lang类型，方便注册
	Lang map[string]string
	// MIME mimetype集合
	MIME map[string]string
	Mime = MIME
	// Regular 正则表达式，方便注册
	Regular map[string][]string

	// State struct {
	// 	Code int
	// 	Text string
	// }

	// Type 类型定义
	Type struct {
		Name    string        `json:"name"`
		Desc    string        `json:"desc"`
		Alias   []string      `json:"alias"`
		Setting Map           `json:"setting"`
		Valid   TypeValidFunc `json:"-"`
		Value   TypeValueFunc `json:"-"`
	}
	TypeValidFunc func(Any, Var) bool
	TypeValueFunc func(Any, Var) Any

	// basicModule 是基础模块
	// 主要用功能是 状态、多语言字串、MIME类型、正则表达式等等
	basicModule struct {
		mutex sync.Mutex

		//存储所有状态定义
		states State
		// langs 多语言字串集合
		langs Lang
		// mimes MIME集合
		mimes MIME
		// regulars 正则表达式集合
		regulars Regular
		// types 参数类型集合
		types map[string]Type
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
		module.State(val, override)
	case MIME:
		module.MIME(val, override)
	case Regular:
		module.Regular(val, override)
	case Type:
		module.Type(name, val, override)
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

// State 注册状态
// 如果State携带了String，则自动注册成默认语言字串
func (module *basicModule) State(config State, override bool) {
	for key, val := range config {
		if override {
			module.states[key] = val
		} else {
			if _, ok := module.states[key]; ok == false {
				module.states[key] = val
			}
		}
	}
}

// func (module *basicModule) State(name string, config State, override bool) {
// 	alias := make([]string, 0)
// 	if name != "" {
// 		alias = append(alias, name)
// 	}

// 	if override {
// 		module.states[name] = config
// 	} else {
// 		if _, ok := module.states[name]; ok == false {
// 			module.states[name] = config
// 		}
// 	}

// 	//自动注册默认的语言字串
// 	if config.Text != "" {
// 		module.Lang(DEFAULT, Lang{name: config.Text}, override)
// 	}
// }

// StateCode 获取状态的代码
// defs 可指定默认code，不存在时将返回默认code
func (module *basicModule) StateCode(state string, defs ...int) int {
	if code, ok := module.states[state]; ok {
		return code
	}
	if len(defs) > 0 {
		return defs[0]
	}
	return -1
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

// String 获取语言字串
func (module *basicModule) String(lang, name string, args ...Any) string {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	if lang == "" {
		lang = DEFAULT
	}

	//把所有语言字串的.都替换成_
	name = strings.Replace(name, ".", "_", -1)

	defaultKey := fmt.Sprintf("%v.%v", DEFAULT, name)
	langKey := fmt.Sprintf("%v.%v", lang, name)

	langStr := ""

	if vv, ok := module.langs[langKey]; ok && vv != "" {
		langStr = vv
	} else if vv, ok := module.langs[defaultKey]; ok && vv != "" {
		langStr = vv
	} else {
		langStr = name
	}

	if len(args) > 0 {
		ccc := strings.Count(langStr, "%") - strings.Count(langStr, "%%")
		if ccc == len(args) {
			return fmt.Sprintf(langStr, args...)
		}
	}
	return langStr
}

// Mime 注册mimetype
func (module *basicModule) MIME(config MIME, override bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	for key, val := range config {
		if override {
			module.mimes[key] = val
		} else {
			if _, ok := module.mimes[key]; ok == false {
				module.mimes[key] = val
			}
		}
	}
}

// Extension 按MIME取扩展名
// defs 为默认值，如果找不到对英的mime，则返回默认
func (module *basicModule) Extension(mime string, defs ...string) string {
	for ext, mmm := range module.mimes {
		if strings.ToLower(mmm) == strings.ToLower(mime) {
			return ext
		}
	}
	if len(defs) > 0 {
		return defs[0]
	}
	return ""
}

// Mimetype 按扩展名拿 MIMEType
// defs 为默认值，如果找不到对应的mime，则返回默认
func (module *basicModule) Mimetype(ext string, defs ...string) string {
	if strings.Contains(ext, "/") {
		return ext
	}

	//去掉点.
	if strings.HasPrefix(ext, ".") {
		ext = strings.TrimPrefix(ext, ".")
	}

	if mime, ok := module.mimes[ext]; ok {
		return mime
	}
	// 如果定义了*，所有不匹配的扩展名，都返回*
	if mime, ok := module.mimes["*"]; ok {
		return mime
	}
	if len(defs) > 0 {
		return defs[0]
	}

	return "application/octet-stream"
}

// Regular 注册正则表达式
func (module *basicModule) Regular(config Regular, override bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	for key, val := range config {
		if override {
			module.regulars[key] = val
		} else {
			if _, ok := module.regulars[key]; ok == false {
				module.regulars[key] = val
			}
		}
	}
}

// Expressions 获取正则的表达式
func (module *basicModule) Expressions(name string, defs ...string) []string {
	if exps, ok := module.regulars[name]; ok {
		return exps
	}
	return defs
}

// Match 正则匹配
func (module *basicModule) Match(regular, value string) bool {
	matchs := module.Expressions(regular)
	for _, v := range matchs {
		regx := regexp.MustCompile(v)
		if regx.MatchString(value) {
			return true
		}
	}
	return false
}

// Type 注册类型
func (module *basicModule) Type(name string, config Type, override bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	alias := make([]string, 0)
	if name != "" {
		alias = append(alias, name)
	}
	if config.Alias != nil {
		alias = append(alias, config.Alias...)
	}

	for _, key := range alias {
		if override {
			module.types[key] = config
		} else {
			if _, ok := module.types[key]; ok == false {
				module.types[key] = config
			}
		}
	}

}

// Types 获取所有类型
func (module *basicModule) Types() map[string]Type {
	types := map[string]Type{}
	for k, v := range module.types {
		types[k] = v
	}
	return types
}

// typeDefaultValid 默认的类型校验方法
// 直接转到正则去匹配
func (module *basicModule) typeDefaultValid(value Any, config Var) bool {
	return module.Match(config.Type, fmt.Sprintf("%s", value))
}

// typeDefaultValue 默认值包装方法
func (module *basicModule) typeDefaultValue(value Any, config Var) Any {
	return fmt.Sprintf("%s", value)
}

// typeValid 获取类型的校验方法
func (module *basicModule) typeValid(name string) TypeValidFunc {
	if config, ok := module.types[name]; ok {
		if config.Valid != nil {
			return config.Valid
		}
	}
	return module.typeDefaultValid
}

// typeValue 获取类型的值包装方法
func (module *basicModule) typeValue(name string) TypeValueFunc {
	if config, ok := module.types[name]; ok {
		if config.Value != nil {
			return config.Value
		}
	}
	return module.typeDefaultValue
}

// typeValue 获取类型的校验和值包装方法
func (module *basicModule) typeMethod(name string) (TypeValidFunc, TypeValueFunc) {
	return module.typeValid(name), module.typeValue(name)
}

// StateCode 返回状态码
func StateCode(name string, defs ...int) int {
	return mBasic.StateCode(name, defs...)
}

// Mimetype 按扩展名获取 MIME 中的 类型
func Mimetype(ext string, defs ...string) string {
	return mBasic.Mimetype(ext, defs...)
}

// Extension 按MIMEType获取扩展名
func Extension(mime string, defs ...string) string {
	return mBasic.Extension(mime, defs...)
}

// String 获取多语言字串
func String(lang, name string, args ...Any) string {
	return mBasic.String(lang, name, args...)
}

// Expressions 获取正则的表达式
func Expressions(name string, defs ...string) []string {
	return mBasic.Expressions(name, defs...)
}

// Match 正则做匹配校验
func Match(regular, value string) bool {
	return mBasic.Match(regular, value)
}

// Types 获取所有类型定义
func Types() map[string]Type {
	return mBasic.Types()
}
