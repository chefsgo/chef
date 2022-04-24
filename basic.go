package chef

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	. "github.com/chefsgo/base"
)

//先放这里，type系统可能独立成一个模块，现在内置
// basic不好独立，因为result依赖
// codec 也得内置， 因为 mapping 依赖，  或是把 type+mapping独立

var (
	mBasic = &basicModule{
		langConfigs: make(map[string]langConfig, 0),

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

	//多语言配置
	langConfig struct {
		Name    string   `toml:"name"`
		Text    string   `toml:"text"`
		Accepts []string `toml:"accepts"`
	}

	// Type 类型定义
	Type struct {
		Name    string        `json:"name"`
		Text    string        `json:"text"`
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
		mutex       sync.Mutex
		langConfigs map[string]langConfig

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

func (module *basicModule) Builtin() {
}
func (module *basicModule) Register(name string, value Any, override bool) {
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
	}

}

func (module *basicModule) Configure(value Any) {
	if cfg, ok := value.(map[string]langConfig); ok {
		module.langConfigs = cfg
		return
	}

	var global Map
	if cfg, ok := value.(Map); ok {
		global = cfg
	} else {
		return
	}

	var config Map
	if vvv, ok := global["lang"].(Map); ok {
		config = vvv
	}

	//记录上一层的配置，如果有的话
	defConfig := Map{}

	for key, val := range config {
		if conf, ok := val.(Map); ok {
			//直接注册，然后删除当前key
			module.langConfigure(key, conf)
		} else {
			//记录上一层的配置，如果有的话
			defConfig[key] = val
		}
	}

	if len(defConfig) > 0 {
		module.langConfigure(DEFAULT, defConfig)
	}

	// if lang, ok := config["lang"].(Map); ok {
	// 	for key, val := range lang {
	// 		if conf, ok := val.(Map); ok {
	// 			module.langConfigure(key, conf)
	// 		}
	// 	}
	// }
}
func (module *basicModule) langConfigure(name string, config Map) {
	lang := langConfig{}

	// 如果之前存在过，就直接拿过来
	if vv, ok := module.langConfigs[name]; ok {
		lang = vv
	}

	if vv, ok := config["name"].(string); ok {
		lang.Name = vv
	}
	if vv, ok := config["text"].(string); ok {
		lang.Text = vv
	}
	if vvs, ok := config["accepts"].([]string); ok {
		lang.Accepts = vvs
	}

	//保存配置
	module.langConfigs[name] = lang
}

func (module *basicModule) Initialize() {
}

func (module *basicModule) Connect() {
}

func (module *basicModule) Launch() {
}

func (module *basicModule) Terminate() {
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

func (module *basicModule) Mapping(config Vars, data Map, value Map, argn bool, pass bool, ctxs ...Context) Res {
	var ctx Context
	if len(ctxs) > 0 && ctxs[0] != nil {
		ctx = ctxs[0]
	} else {
		ctx = newContext()
	}

	/*
	   argn := false
	   if len(args) > 0 {
	       argn = args[0]
	   }
	*/

	//遍历配置	begin
	for fieldName, fieldConfig := range config {

		//注意，这里存在2种情况
		//1. Map对象
		//2. map[string]interface{}
		//要分开处理
		//go1.9以后可以 type xx=yy 就只要处理一个了

		// switch c := fv.(type) {
		// case Map:
		// 	fieldConfig = c
		// default:
		// 	//类型不对，跳过
		// 	continue
		// }

		//解过密？
		decoded := false
		passEmpty := false
		passError := false

		//Map 如果是JSON文件，或是发过来的消息，就不支持
		//而用下面的，就算是MAP也可以支持，OK
		//麻烦来了，web.args用下面这样处理不了
		//if fieldConfig, ok := fv.(map[string]interface{}); ok {

		fieldMust, fieldEmpty := fieldConfig.Required, fieldConfig.Nullable
		fieldValue, fieldExist := data[fieldName]
		fieldAuto, fieldJson := fieldConfig.Default, fieldConfig.Children
		//_, fieldEmpty = data[fieldName]

		// if argn {
		//	//这里应该是可以的，相当于，所有字段为nullable，表示，可以不存在
		// 	fieldEmpty = true
		// }

		//trees := append(tree, fieldName)
		//fmt.Printf("trees=%v". strings.Join(trees, "."))

		//fmt.Printf("t=%s, argn=%v, value=%v\n", strings.Join(trees, "."), argn, fieldValue)
		//fmt.Printf("trees=%v, must=%v, empty=%v, exist=%v, auto=%v, value=%v, config=%v\n\n",
		//	strings.Join(trees, "."), fieldMust, fieldEmpty, fieldExist, fieldAuto, fieldValue, fieldConfig)

		//fmt.Println("mapping", fieldName)

		strVal := fmt.Sprintf("%v", fieldValue)

		//等一下。 空的map[]无字段。 需要也表示为空吗?
		//if strVal == "" || strVal == "map[]" || strVal == "{}"{

		//因为go1.6之后。把一个值为nil的map  再写入map之后, 判断 if map[key]==nil 就无效了
		if strVal == "" || data[fieldName] == nil || (fieldMust != true && strVal == "map[]") {
			fieldValue = nil
		}

		//如果不可为空，但是为空了，
		if fieldMust && fieldEmpty == false && (fieldValue == nil || strVal == "") && fieldAuto == nil && fieldJson == nil && argn == false {

			//是否跳过
			if pass {
				passEmpty = true
			} else {
				//是否有自定义的状态
				if fieldConfig.Empty != nil {
					return fieldConfig.Empty
				} else {
					//这样方便在多语言环境使用
					key := "_mapping_empty_" + fieldName
					if module.StateCode(key, -999) == -999 {
						return textResult("_mapping_empty", fieldConfig.Name)
					}
					return textResult(key)
				}
			}

		} else {

			//如果值为空的时候，看有没有默认值
			//到这里值是可以为空的
			if fieldValue == nil || strVal == "" {

				//如果有默认值
				//可为NULL时，不给默认值
				if fieldAuto != nil && !argn {

					//暂时不处理 $id, $date 之类的定义好的默认值，不包装了
					switch autoValue := fieldAuto.(type) {
					case func() interface{}:
						fieldValue = autoValue()
					case func() time.Time:
						fieldValue = autoValue()
						//case func() bson.ObjectId:	//待处理
						//fieldValue = autoValue()
					case func() string:
						fieldValue = autoValue()
					case func() int:
						fieldValue = int64(autoValue())
					case func() int8:
						fieldValue = int64(autoValue())
					case func() int16:
						fieldValue = int64(autoValue())
					case func() int32:
						fieldValue = int64(autoValue())
					case func() int64:
						fieldValue = autoValue()
					case func() float32:
						fieldValue = float64(autoValue())
					case func() float64:
						fieldValue = autoValue()
					case int:
						{
							fieldValue = int64(autoValue)
						}
					case int8:
						{
							fieldValue = int64(autoValue)
						}
					case int16:
						{
							fieldValue = int64(autoValue)
						}
					case int32:
						{
							fieldValue = int64(autoValue)
						}
					case float32:
						{
							fieldValue = float64(autoValue)
						}

					case []int:
						{
							arr := []int64{}
							for _, nv := range autoValue {
								arr = append(arr, int64(nv))
							}
							fieldValue = arr
						}
					case []int8:
						{
							arr := []int64{}
							for _, nv := range autoValue {
								arr = append(arr, int64(nv))
							}
							fieldValue = arr
						}
					case []int16:
						{
							arr := []int64{}
							for _, nv := range autoValue {
								arr = append(arr, int64(nv))
							}
							fieldValue = arr
						}
					case []int32:
						{
							arr := []int64{}
							for _, nv := range autoValue {
								arr = append(arr, int64(nv))
							}
							fieldValue = arr
						}

					case []float32:
						{
							arr := []float64{}
							for _, nv := range autoValue {
								arr = append(arr, float64(nv))
							}
							fieldValue = arr
						}

					default:
						fieldValue = autoValue
					}

					//默认值是不是也要包装一下，这里只包装值，不做验证
					if fieldConfig.Type != "" {
						_, fieldValueCall := module.typeMethod(fieldConfig.Type)

						//如果配置中有自己的值函数
						if fieldConfig.Value != nil {
							fieldValueCall = fieldConfig.Value
						}

						//包装值
						if fieldValueCall != nil {
							fieldValue = fieldValueCall(fieldValue, fieldConfig)
						}
					}

				} else { //没有默认值, 且值为空

					//有个问题, POST表单的时候.  表单字段如果有，值是存在的，会是空字串
					//但是POST的时候如果有argn, 实际上是不想存在此字段的

					//如果字段可以不存在
					if fieldEmpty || argn {

						//当empty(argn)=true，并且如果传过值中存在字段的KEY，值就要存在，以更新为null
						if argn && fieldExist {
							//不操作，自然往下执行
						} else { //值可以不存在
							continue
						}

					} else if argn {

					} else {
						//到这里不管
						//因为字段必须存在，但是值为空
					}
				}

			} else { //值不为空，处理值

				//值处理前，是不是需要解密
				//如果解密哦
				//decode
				if fieldConfig.Decode != "" {

					//有一个小bug这里，在route的时候， 如果传的是string，本来是想加密的， 结果这里变成了解密
					//还得想个办法解决这个问题，所以，在传值的时候要注意，另外string型加密就有点蛋疼了
					//主要是在route的时候，其它的时候还ok，所以要在encode/decode中做类型判断解还是不解

					//而且要值是string类型
					// if sv,ok := fieldValue.(string); ok {

					//得到解密方法
					if val, err := mCodec.Decrypt(fieldConfig.Decode, strVal); err == nil {
						//前方解过密了，表示该参数，不再加密
						//因为加密解密，只有一个2选1的
						//比如 args 只需要解密 data 只需要加密
						//route 的时候 args 需要加密，而不用再解，所以是单次的
						fieldValue = val
						decoded = true
					}
					// }
				}

				//验证放外面来，因为默认值也要验证和包装

				//按类型来做处理

				//验证方法和值方法
				//但是因为默认值的情况下，值有可能是为空的，所以要判断多一点
				if fieldConfig.Type != "" {
					fieldValidCall, fieldValueCall := module.typeMethod(fieldConfig.Type)

					//如果配置中有自己的验证函数
					if fieldConfig.Valid != nil {
						fieldValidCall = fieldConfig.Valid
					}
					//如果配置中有自己的值函数
					if fieldConfig.Value != nil {
						fieldValueCall = fieldConfig.Value
					}

					//开始调用验证
					if fieldValidCall != nil {
						//如果验证通过
						if fieldValidCall(fieldValue, fieldConfig) {
							//包装值
							if fieldValueCall != nil {
								//对时间值做时区处理
								if ctx != nil {
									if ctx.Zone() != time.Local {
										if vv, ok := fieldValue.(time.Time); ok {
											fieldValue = vv.In(ctx.Zone())
										} else if vvs, ok := fieldValue.([]time.Time); ok {
											newTimes := []time.Time{}
											for _, vv := range vvs {
												newTimes = append(newTimes, vv.In(ctx.Zone()))
											}
											fieldValue = newTimes
										}
									}
								}

								fieldValue = fieldValueCall(fieldValue, fieldConfig)
							}
						} else { //验证不通过

							//是否可以跳过
							if pass {
								passError = true
							} else {

								//是否有自定义的状态
								if fieldConfig.Error != nil {
									return fieldConfig.Error
								} else {
									//这样方便在多语言环境使用
									//待优化， 通一成一个state
									key := "_mapping_error_" + fieldName
									if module.StateCode(key, -999) == -999 {
										return textResult("_mapping_error", fieldConfig.Name)
									}
									return textResult(key)
								}
							}
						}
					}
				}

			}

		}

		//这后面是总的字段处理
		//如JSON，加密

		//如果是JSON， 或是数组啥的处理
		//注意，当 json 本身可为空，下级有不可为空的，值为空时， 应该跳过子级的检查
		//如果 json 可为空， 就不应该有 默认值， 定义的时候要注意啊啊啊啊
		//理论上，只要JSON可为空～就不处理下一级json
		jsonning := true
		if !fieldMust && fieldValue == nil {
			jsonning = false
		}

		//还有种情况要处理. 当type=json, must=true的时候,有默认值, 但是没有定义json节点.

		if fieldConfig.Children != nil && jsonning {
			jsonConfig := fieldConfig.Children
			//注意，这里存在2种情况
			//1. Map对象 //2. map[string]interface{}

			// switch c := m.(type) {
			// case Map:
			// 	jsonConfig = c
			// }

			//如果是数组
			isArray := false
			//fieldData到这里定义
			fieldData := []Map{}

			switch v := fieldValue.(type) {
			case Map:
				fieldData = append(fieldData, v)
			case []Map:
				isArray = true
				fieldData = v
			default:
				fieldData = []Map{}
			}

			//直接都遍历
			values := []Map{}

			for _, d := range fieldData {
				v := Map{}

				// err := module.Parse(trees, jsonConfig, d, v, argn, pass);
				res := module.Mapping(jsonConfig, d, v, argn, pass, ctx)
				if res != nil && res.Fail() {
					return res
				} else {
					//fieldValue = append(fieldValue, v)
					values = append(values, v)
				}
			}

			if isArray {
				fieldValue = values
			} else {
				if len(values) > 0 {
					fieldValue = values[0]
				} else {
					fieldValue = Map{}
				}
			}

		}

		// 跳过且为空时，不写值
		if pass && (passEmpty || passError) {
		} else {

			//当pass=true的时候， 这里可能会是空值，那应该跳过
			//最后，值要不要加密什么的
			//如果加密
			//encode
			if fieldConfig.Encode != "" && decoded == false && passEmpty == false && passError == false {

				/*
				   //全都转成字串再加密
				   //为什么要全部转字串才能加密？
				   //不用转了，因为hashid这样的加密就要int64
				*/

				// encrypt := mCodec.getEncrypt(fieldConfig.Encode)
				if val, err := mCodec.Encrypt(fieldConfig.Encode, fieldValue); err == nil {
					fieldValue = val
				}
			}
		}

		//没有JSON要处理，所以给值
		value[fieldName] = fieldValue

	}

	return OK
	//遍历配置	end
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

func Mapping(config Vars, data Map, value Map, argn bool, pass bool, ctxs ...Context) Res {
	return mBasic.Mapping(config, data, value, argn, pass, ctxs...)
}
