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
		languages: make(map[string]Language, 0),
		strings:   make(Strings, 0),

		states:   make(States, 0),
		mimes:    make(Mimes, 0),
		regulars: make(Regulars, 0),
		types:    make(map[string]Type, 0),
	}
)

type (
	// basicModule 是基础模块
	// 主要用功能是 状态、多语言字串、MIME类型、正则表达式等等
	basicModule struct {
		mutex     sync.Mutex
		languages map[string]Language
		strings   Strings

		//存储所有状态定义
		states States
		// mimes MIME集合
		mimes Mimes
		// regulars 正则表达式集合
		regulars Regulars
		// types 参数类型集合
		types map[string]Type
	}

	// 注意，以下几个类型，不能使用 xxx = map[xxx]yy 的方法定义
	// 因为无法使用.(type)来断言类型

	// State 状态定义，方便注册
	State  int
	States map[string]State
	// MIME mimetype集合
	Mime  []string
	Mimes map[string][]string
	// Regular 正则表达式，方便注册
	Regular  []string
	Regulars map[string]Regular

	//多语言配置
	Strings  map[string]string
	Language struct {
		// Name 语言名称
		Name string
		// Text 语言说明
		Text string
		// Accepts 匹配的语言
		// 比如，znCN, zh, zh-CN 等自动匹配
		Accepts []string
		// Strings 当前语言是字符串列表
		Strings Strings
	}

	// Type 类型定义
	Type struct {
		// Name 类型名称
		Name string

		// Text 类型说明
		Text string

		// Alias 类型别名
		Alias []string

		// Valid 类型验证方法
		Valid TypeValidFunc

		// Value 值包装方法
		Value TypeValueFunc
	}

	TypeValidFunc func(Any, Var) bool
	TypeValueFunc func(Any, Var) Any
)

func (this *basicModule) Builtin() {
}
func (this *basicModule) Register(name string, value Any, override bool) {
	switch val := value.(type) {
	case Language:
		this.Language(name, val, override)
	case Strings:
		this.Strings(name, val, override)
	case State:
		this.State(name, val, override)
	case States:
		this.States(val, override)
	case Mime:
		this.Mime(name, val, override)
	case Mimes:
		this.Mimes(val, override)
	case Regular:
		this.Regular(name, val, override)
	case Regulars:
		this.Regulars(val, override)
	case Type:
		this.Type(name, val, override)
	}
}

func (this *basicModule) langConfigure(name string, config Map) {
	lang := Language{Name: name, Strings: make(Strings, 0)}
	if vv, ok := this.languages[name]; ok {
		lang = vv //如果已经存在了，用现成的改写
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
	//这里覆盖
	if vvs, ok := config["strings"].(map[string]string); ok {
		for key, val := range vvs {
			lang.Strings[key] = val
		}
	}
	if vvs, ok := config["strings"].(Map); ok {
		for key, val := range vvs {
			if str, ok := val.(string); ok {
				lang.Strings[key] = str
			}
		}
	}

	//保存配置
	this.languages[name] = lang
}

func (this *basicModule) Configure(value Map) {
	// if cfg, ok := value.(map[string]langConfig); ok {
	// 	this.langConfigs = cfg
	// 	return
	// }

	// var global Map
	// if cfg, ok := value.(Map); ok {
	// 	global = cfg
	// } else {
	// 	return
	// }

	// var config Map
	// if vvv, ok := global["lang"].(Map); ok {
	// 	config = vvv
	// }

	// //记录上一层的配置，如果有的话
	// defConfig := Map{}

	// for key, val := range config {
	// 	if conf, ok := val.(Map); ok {
	// 		//直接注册，然后删除当前key
	// 		this.langConfigure(key, conf)
	// 	} else {
	// 		//记录上一层的配置，如果有的话
	// 		defConfig[key] = val
	// 	}
	// }

	// if len(defConfig) > 0 {
	// 	this.langConfigure(DEFAULT, defConfig)
	// }

	// if lang, ok := config["lang"].(Map); ok {
	// 	for key, val := range lang {
	// 		if conf, ok := val.(Map); ok {
	// 			this.langConfigure(key, conf)
	// 		}
	// 	}
	// }
}

func (this *basicModule) Initialize() {
}

func (this *basicModule) Connect() {
}

func (this *basicModule) Launch() {
}

func (this *basicModule) Terminate() {
}

// State 注册状态
func (this *basicModule) State(name string, config State, override bool) {
	if override {
		this.states[name] = config
	} else {
		if _, ok := this.states[name]; ok == false {
			this.states[name] = config
		}
	}
}
func (this *basicModule) States(config States, override bool) {
	for key, val := range config {
		this.State(key, State(val), override)
	}
}

// StateCode 获取状态的代码
// defs 可指定默认code，不存在时将返回默认code
func (this *basicModule) StateCode(state string, defs ...int) int {
	if code, ok := this.states[state]; ok {
		return int(code)
	}
	if len(defs) > 0 {
		return defs[0]
	}
	return -1
}

// Language 注册语言
func (this *basicModule) Language(name string, config Language, override bool) {

	if config.Strings == nil {
		config.Strings = make(Strings, 0)
	}

	if override {
		this.languages[name] = config
	} else {
		if _, ok := this.languages[name]; ok == false {
			this.languages[name] = config
		}
	}

}

func (this *basicModule) Strings(name string, config Strings, override bool) {

	// 对于不存在的语言，先自动来一个
	if _, ok := this.languages[name]; ok == false {
		this.languages[name] = Language{
			Name: name, Text: name, Accepts: []string{},
			Strings: make(Strings, 0),
		}
	}

	if lang, ok := this.languages[name]; ok {
		for key, str := range config {
			key = strings.Replace(key, ".", "_", -1)
			if override {
				lang.Strings[key] = str
			} else {
				if _, ok := lang.Strings[key]; ok == false {
					lang.Strings[key] = str
				}
			}
		}
	}
}

func (this *basicModule) Languages() map[string]Language {
	return this.languages
}

// Strings 获取语言字串列表
// func (this *basicModule) getStrings(name string) Strings {
// 	strs := Strings{}
// 	if lang, ok := this.languages[name]; ok {
// 		for key, str := range lang.Strings {
// 			strs[key] = str
// 		}
// 	}
// 	return strs
// }
func (this *basicModule) String(lang, key string, args ...Any) string {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if lang == "" {
		lang = DEFAULT
	}

	//把所有语言字串的.都替换成_
	key = strings.Replace(key, ".", "_", -1)

	langStr := ""
	if cfg, ok := this.languages[lang]; ok {
		if str, ok := cfg.Strings[key]; ok {
			langStr = str
		}
	} else if cfg, ok := this.languages[DEFAULT]; ok {
		if str, ok := cfg.Strings[key]; ok {
			langStr = str
		}
	} else {
		langStr = key
	}

	if len(args) > 0 {
		ccc := strings.Count(langStr, "%") - strings.Count(langStr, "%%")
		if ccc == len(args) {
			return fmt.Sprintf(langStr, args...)
		}
	}
	return langStr
}

// Mime 注册Mimetype
func (this *basicModule) Mime(name string, config Mime, override bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if override {
		this.mimes[name] = config
	} else {
		if _, ok := this.mimes[name]; ok == false {
			this.mimes[name] = config
		}
	}
}
func (this *basicModule) Mimes(config Mimes, override bool) {
	for key, val := range config {
		this.Mime(key, val, override)
	}
}

// Extension 按MIME取扩展名
// defs 为默认值，如果找不到对英的mime，则返回默认
func (this *basicModule) Extension(mime string, defs ...string) string {
	for ext, mmms := range this.mimes {
		for _, mmm := range mmms {
			if strings.ToLower(mmm) == strings.ToLower(mime) {
				return ext
			}
		}
	}
	if len(defs) > 0 {
		return defs[0]
	}
	return ""
}

// Mimetype 按扩展名拿 MIMEType
// defs 为默认值，如果找不到对应的mime，则返回默认
func (this *basicModule) Mimetype(ext string, defs ...string) string {
	if strings.Contains(ext, "/") {
		return ext
	}

	//去掉点.
	if strings.HasPrefix(ext, ".") {
		ext = strings.TrimPrefix(ext, ".")
	}

	if mimes, ok := this.mimes[ext]; ok && len(mimes) > 0 {
		return mimes[0]
	}
	// 如果定义了*，所有不匹配的扩展名，都返回*
	if mimes, ok := this.mimes["*"]; ok && len(mimes) > 0 {
		return mimes[0]
	}
	if len(defs) > 0 {
		return defs[0]
	}

	return "application/octet-stream"
}

// Regular 注册正则表达式
func (this *basicModule) Regular(name string, config Regular, override bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if override {
		this.regulars[name] = config
	} else {
		if _, ok := this.regulars[name]; ok == false {
			this.regulars[name] = config
		}
	}
}
func (this *basicModule) Regulars(config Regulars, override bool) {
	for key, val := range config {
		this.Regular(key, val, override)
	}
}

// Expressions 获取正则的表达式
func (this *basicModule) Expressions(name string, defs ...string) []string {
	if exps, ok := this.regulars[name]; ok {
		return exps
	}
	return defs
}

// Match 正则匹配
func (this *basicModule) Match(regular, value string) bool {
	matchs := this.Expressions(regular)
	for _, v := range matchs {
		regx := regexp.MustCompile(v)
		if regx.MatchString(value) {
			return true
		}
	}
	return false
}

// Type 注册类型
func (this *basicModule) Type(name string, config Type, override bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	alias := make([]string, 0)
	if name != "" {
		alias = append(alias, name)
	}
	if config.Alias != nil {
		alias = append(alias, config.Alias...)
	}

	for _, key := range alias {
		if override {
			this.types[key] = config
		} else {
			if _, ok := this.types[key]; ok == false {
				this.types[key] = config
			}
		}
	}

}

// Types 获取所有类型
func (this *basicModule) Types() map[string]Type {
	types := map[string]Type{}
	for k, v := range this.types {
		types[k] = v
	}
	return types
}

// typeDefaultValid 默认的类型校验方法
// 直接转到正则去匹配
func (this *basicModule) typeDefaultValid(value Any, config Var) bool {
	return this.Match(config.Type, fmt.Sprintf("%s", value))
}

// typeDefaultValue 默认值包装方法
func (this *basicModule) typeDefaultValue(value Any, config Var) Any {
	return fmt.Sprintf("%s", value)
}

// typeValid 获取类型的校验方法
func (this *basicModule) typeValid(name string) TypeValidFunc {
	if config, ok := this.types[name]; ok {
		if config.Valid != nil {
			return config.Valid
		}
	}
	return this.typeDefaultValid
}

// typeValue 获取类型的值包装方法
func (this *basicModule) typeValue(name string) TypeValueFunc {
	if config, ok := this.types[name]; ok {
		if config.Value != nil {
			return config.Value
		}
	}
	return this.typeDefaultValue
}

// typeValue 获取类型的校验和值包装方法
func (this *basicModule) typeMethod(name string) (TypeValidFunc, TypeValueFunc) {
	return this.typeValid(name), this.typeValue(name)
}

// Mapping 处理动态参数方法
// 这方法大概是在2016-2017年写的
// 最近没有时间重构优化，将就着用用吧，
// 等框架核心的东西和文档写完，再来优化这部分东西
func (this *basicModule) Mapping(config Vars, data Map, value Map, argn bool, pass bool, zones ...*time.Location) Res {
	timezone := time.Local
	if len(zones) > 0 {
		timezone = zones[0]
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
					if this.StateCode(key, -999) == -999 {
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
						_, fieldValueCall := this.typeMethod(fieldConfig.Type)

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
					fieldValidCall, fieldValueCall := this.typeMethod(fieldConfig.Type)

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

								if vv, ok := fieldValue.(time.Time); ok {
									fieldValue = vv.In(timezone)
								} else if vvs, ok := fieldValue.([]time.Time); ok {
									newTimes := []time.Time{}
									for _, vv := range vvs {
										newTimes = append(newTimes, vv.In(timezone))
									}
									fieldValue = newTimes
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
									if this.StateCode(key, -999) == -999 {
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

				// err := this.Parse(trees, jsonConfig, d, v, argn, pass);
				res := this.Mapping(jsonConfig, d, v, argn, pass, timezone)
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

func Languages() map[string]Language {
	return mBasic.Languages()
}

// func Strings(lang string) Strs {
// 	return mBasic.Strings(lang)
// }

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

func Mapping(config Vars, data Map, value Map, argn bool, pass bool, zones ...*time.Location) Res {
	return mBasic.Mapping(config, data, value, argn, pass, zones...)
}
