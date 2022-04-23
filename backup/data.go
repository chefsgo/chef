package chef

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	. "github.com/chefsgo/base"
)

var (
	mData = newDataModule()
)

func newDataModule() *dataModule {
	return &dataModule{
		configs:   make(map[string]DataConfig, 0),
		drivers:   make(map[string]DataDriver, 0),
		instances: make(map[string]dataInstance, 0),
		tables:    make(map[string]Table, 0),
		views:     make(map[string]View, 0),
		models:    make(map[string]Model, 0),
	}
}

const (
	DataCreateTrigger  = "$.data.create"
	DataChangeTrigger  = "$.data.change"
	DataRemoveTrigger  = "$.data.remove"
	DataRecoverTrigger = "$.data.recover"
)

type (
	DataConfig struct {
		Driver  string `toml:"driver"`
		Url     string `toml:"url"`
		Serial  string `toml:"serial"`
		Setting Map    `toml:"setting"`
	}
	// DataDriver 数据驱动
	DataDriver interface {
		Connect(name string, config DataConfig) (DataConnect, error)
	}

	DataConnect interface {
		Open() error
		Health() (DataHealth, error)
		Close() error

		Base() DataBase
	}

	DataBase interface {
		Close() error
		Erred() error

		Table(name string) DataTable
		View(name string) DataView
		Model(name string) DataModel

		Serial(key string, start, step int64) int64
		Break(key string)

		//开启手动提交事务模式
		Begin() (*sql.Tx, error)
		Submit() error
		Cancel() error
	}

	DataTable interface {
		Create(Map) Map
		Change(Map, Map) Map
		Remove(...Any) Map
		Update(sets Map, args ...Any) int64
		Delete(args ...Any) int64

		Entity(Any) Map
		Count(args ...Any) float64
		First(args ...Any) Map
		Query(args ...Any) []Map
		Limit(offset, limit Any, args ...Any) (int64, []Map)
		Group(field string, args ...Any) []Map
	}

	//数据视图接口
	DataView interface {
		Count(args ...Any) float64
		First(args ...Any) Map
		Query(args ...Any) []Map
		Limit(offset, limit Any, args ...Any) (int64, []Map)
		Group(field string, args ...Any) []Map
	}

	//数据模型接口
	DataModel interface {
		First(args ...Any) Map
		Query(args ...Any) []Map
	}

	DataHealth struct {
		Workload int64
	}
	DataTrigger struct {
		Name  string
		Value Map
	}

	Table struct {
		Name    string `json:"name"`
		Desc    string `json:"desc"`
		Schema  string `json:"schema"`
		Table   string `json:"table"`
		Key     string `json:"key"`
		Fields  Vars   `json:"fields"`
		Setting Map    `toml:"setting"`
	}
	View struct {
		Name    string `json:"name"`
		Desc    string `json:"desc"`
		Schema  string `json:"schema"`
		View    string `json:"view"`
		Key     string `json:"key"`
		Fields  Vars   `json:"fields"`
		Setting Map    `toml:"setting"`
	}
	Model struct {
		Name    string `json:"name"`
		Desc    string `json:"desc"`
		Model   string `json:"model"`
		Key     string `json:"key"`
		Fields  Vars   `json:"fields"`
		Setting Map    `toml:"setting"`
	}

	Relate struct {
		Key, Field, Status, Type string
	}

	dataInstance struct {
		name    string
		config  DataConfig
		connect DataConnect
	}

	dataModule struct {
		mutex   sync.Mutex
		configs map[string]DataConfig

		drivers map[string]DataDriver
		tables  map[string]Table
		views   map[string]View
		models  map[string]Model

		//连接
		instances map[string]dataInstance
	}

	//dataGroup struct {
	//	data *dataModule
	//	base string
	//}
)

// Builtin
func (module *dataModule) Builtin() {

}

// Register
func (module *dataModule) Register(key string, value Any, override bool) {
	switch val := value.(type) {
	case DataDriver:
		module.Driver(key, val, override)
	case Table:
		module.Table(key, val, override)
	case View:
		module.View(key, val, override)
	case Model:
		module.Model(key, val, override)
	}
}

// 处理单个配置
func (module *dataModule) configure(name string, config Map) {
	cfg := DataConfig{
		Driver: DEFAULT, Serial: "serial",
	}
	//如果已经存在了，用现成的改写
	if vv, ok := module.configs[name]; ok {
		cfg = vv
	}

	if driver, ok := config["driver"].(string); ok {
		cfg.Driver = driver
	}

	if url, ok := config["url"].(string); ok {
		cfg.Url = url
	}
	if serial, ok := config["serial"].(string); ok {
		cfg.Serial = serial
	}
	if setting, ok := config["setting"].(Map); ok {
		cfg.Setting = setting
	}

	//保存配置
	module.configs[name] = cfg
}
func (module *dataModule) Configure(config Map) {
	var confs Map
	if vvv, ok := config["data"].(Map); ok {
		confs = vvv
	}

	//记录上一层的配置，如果有的话
	defConfig := Map{}

	for key, val := range confs {
		if conf, ok := val.(Map); ok {
			//直接注册，然后删除当前key
			module.configure(key, conf)
		} else {
			//记录上一层的配置，如果有的话
			defConfig[key] = val
		}
	}

	if len(defConfig) > 0 {
		module.configure(DEFAULT, defConfig)
	}
}

//初始化
func (module *dataModule) Initialize() {

}

//Launch
func (module *dataModule) Connect() {
	for name, config := range module.configs {
		driver, ok := module.drivers[config.Driver]
		if ok == false {
			panic("Invalid data driver: " + config.Driver)
		}

		// 建立连接
		connect, err := driver.Connect(name, config)
		if err != nil {
			panic("Failed to connect to data: " + err.Error())
		}

		// 打开连接
		err = connect.Open()
		if err != nil {
			panic("Failed to open data connect: " + err.Error())
		}

		//保存连接
		module.instances[name] = dataInstance{
			name, config, connect,
		}
	}
}

//Launch
func (module *dataModule) Launch() {

}

//退出
func (module *dataModule) Terminate() {
	for _, ins := range module.instances {
		ins.connect.Close()
	}
}

//注册驱动
func (module *dataModule) Driver(name string, driver DataDriver, override bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	if driver == nil {
		panic("Invalid data driver: " + name)
	}

	if override {
		module.drivers[name] = driver
	} else {
		if module.drivers[name] == nil {
			module.drivers[name] = driver
		}
	}
}

func (module *dataModule) Table(name string, config Table, override bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	if override {
		module.tables[name] = config
	} else {
		if _, ok := module.tables[name]; ok == false {
			module.tables[name] = config
		}
	}
}
func (module *dataModule) TableConfig(name string) *Table {
	if config, ok := module.tables[name]; ok {
		//注意：这里应该是复制了一份
		return &Table{
			config.Name, config.Desc,
			config.Schema, config.Table,
			config.Key, config.Fields, config.Setting,
		}
		// return &config
	}
	return nil
}

func (module *dataModule) View(name string, config View, override bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	if override {
		module.views[name] = config
	} else {
		if _, ok := module.views[name]; ok == false {
			module.views[name] = config
		}
	}
}
func (module *dataModule) ViewConfig(name string) *View {
	if config, ok := module.views[name]; ok {
		//注意：这里应该是复制了一份
		return &View{
			config.Name, config.Desc,
			config.Schema, config.View,
			config.Key, config.Fields, config.Setting,
		}
		// return &config
	}
	return nil
}

//注册模型
func (module *dataModule) Model(name string, config Model, override bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	if override {
		module.models[name] = config
	} else {
		if _, ok := module.models[name]; ok == false {
			module.models[name] = config
		}
	}
}
func (module *dataModule) ModelConfig(name string) *Model {
	if config, ok := module.models[name]; ok {
		//注意：这里应该是复制了一份
		return &Model{
			config.Name, config.Desc, config.Model,
			config.Key, config.Fields, config.Setting,
		}
		return &config
	}
	return nil
}

func (module *dataModule) Field(name string, field string, extends ...Any) Var {
	fields := module.Fields(name, []string{field})
	var config Var
	if vv, ok := fields[field]; ok {
		config = vv
	}

	return VarExtend(config, extends...)
}
func (module *dataModule) Fields(name string, keys []string, extends ...Vars) Vars {
	if _, ok := module.tables[name]; ok {
		return module.TableFields(name, keys, extends...)
	} else if _, ok := module.views[name]; ok {
		return module.ViewFields(name, keys, extends...)
	} else if _, ok := module.models[name]; ok {
		return module.ModelFields(name, keys, extends...)
	} else {
		if len(extends) > 0 {
			return extends[0]
		}
		return Vars{}
	}
}
func (module *dataModule) TableFields(name string, keys []string, extends ...Vars) Vars {
	fields := Vars{}
	if config, ok := module.tables[name]; ok && config.Fields != nil {
		//空数组一个也不返
		if keys == nil {
			for k, v := range config.Fields {
				fields[k] = v
			}
		} else {
			for _, k := range keys {
				if v, ok := config.Fields[k]; ok {
					fields[k] = v
				}

			}
		}
	}

	if len(extends) > 0 {
		for k, v := range extends[0] {
			if v.Nil() {
				delete(fields, k)
			} else {
				fields[k] = v
			}
		}
	}

	return fields
}
func (module *dataModule) ViewFields(name string, keys []string, exts ...Vars) Vars {
	fields := Vars{}
	if config, ok := module.views[name]; ok && config.Fields != nil {
		//空数组一个也不返
		if keys == nil {
			for k, v := range config.Fields {
				fields[k] = v
			}
		} else {
			for _, k := range keys {
				if v, ok := config.Fields[k]; ok {
					fields[k] = v
				}

			}
		}
	}

	if len(exts) > 0 {
		for k, v := range exts[0] {
			fields[k] = v
		}
	}

	return fields
}
func (module *dataModule) ModelFields(name string, keys []string, exts ...Vars) Vars {
	fields := Vars{}
	if config, ok := module.models[name]; ok && config.Fields != nil {
		//空数组一个也不返
		if keys == nil {
			for k, v := range config.Fields {
				fields[k] = v
			}
		} else {
			for _, k := range keys {
				if v, ok := config.Fields[k]; ok {
					fields[k] = v
				}
			}
		}
	}

	if len(exts) > 0 {
		for k, v := range exts[0] {
			fields[k] = v
		}
	}

	return fields
}
func (module *dataModule) Option(name, field, key string) Any {
	enums := mData.Options(name, field)
	if vv, ok := enums[key]; ok {
		return vv
	}
	return key
}

func (module *dataModule) Options(name, field string) Map {
	if _, ok := module.tables[name]; ok {
		return module.TableOptions(name, field)
	} else if _, ok := module.views[name]; ok {
		return module.ViewOptions(name, field)
	} else if _, ok := module.models[name]; ok {
		return module.ModelOptions(name, field)
	} else {
		return Map{}
	}
}

//2021-03-04支持一级子字段的option
func (module *dataModule) TableOptions(name, field string) Map {
	options := Map{}
	fields := strings.Split(field, ".")
	if len(fields) > 1 {
		field = fields[0]
		child := fields[1]
		if config, ok := module.tables[name]; ok && config.Fields != nil {
			if field, ok := config.Fields[field]; ok {
				if childConfig, ok := field.Children[child]; ok {
					if childConfig.Options != nil {
						for k, v := range childConfig.Options {
							options[k] = v
						}
					}
				}
			}
		}
	} else {
		if config, ok := module.tables[name]; ok && config.Fields != nil {
			if field, ok := config.Fields[field]; ok {
				if field.Options != nil {
					for k, v := range field.Options {
						options[k] = v
					}
				}
			}
		}
	}

	return options
}
func (module *dataModule) ViewOptions(name, field string) Map {
	options := Map{}
	if config, ok := module.views[name]; ok && config.Fields != nil {
		if field, ok := config.Fields[field]; ok {
			if field.Options != nil {
				for k, v := range field.Options {
					options[k] = v
				}
			}
		}
	}
	return options
}
func (module *dataModule) ModelOptions(name, field string) Map {
	options := Map{}
	if config, ok := module.models[name]; ok && config.Fields != nil {
		if field, ok := config.Fields[field]; ok {
			if field.Options != nil {
				for k, v := range field.Options {
					options[k] = v
				}
			}
		}
	}
	return options
}

//Instance
func (module *dataModule) Instance(names ...string) dataInstance {
	if len(names) > 0 {
		if inst, ok := module.instances[names[0]]; ok {
			return inst
		}
	} else {
		for _, val := range mData.instances {
			return val
		}
	}
	panic("Invalid data connection.")
}

//返回数据Base对象
func (module *dataModule) Base(names ...string) DataBase {
	inst := module.Instance(names...)
	return inst.connect.Base()
}

//----------------------------------------------------------------------

//查询语法解析器
// 字段包裹成  $field$ 请自行处理
// 如mysql为反引号`field`，postgres, oracle为引号"field"，
// 所有参数使用问号(?)表示
// postgres驱动需要自行处理转成 $1,$2这样的
// oracle驱动需要自行处理转成 :1 :2这样的
//mongodb不适用，需驱动自己实现
func (module *dataModule) Parse(args ...Any) (string, []Any, string, error) {

	if len(args) > 0 {

		//如果直接写sql
		if v, ok := args[0].(string); ok {
			sql := v
			params := []interface{}{}
			orderBy := ""

			for i, arg := range args {
				if i > 0 {
					params = append(params, arg)
				}
			}

			//这里要处理一下，把order提取出来
			//先拿到 order by 的位置
			i := strings.Index(strings.ToLower(sql), "order by")
			if i >= 0 {
				orderBy = sql[i:]
				sql = sql[:i]
			}

			return sql, params, orderBy, nil

		} else {

			maps := []Map{}
			for _, v := range args {
				if m, ok := v.(Map); ok {
					maps = append(maps, m)
				}
				//如果直接是[]Map，应该算OR处理啊，暂不处理这个
			}

			querys, values, orders := module.parsing(maps...)

			orderStr := ""
			if len(orders) > 0 {
				orderStr = fmt.Sprintf("ORDER BY %s", strings.Join(orders, ","))
			}

			//sql := fmt.Sprintf("%s %s", strings.Join(querys, " OR "), orderStr)

			if len(querys) == 0 {
				querys = append(querys, "1=1")
			}

			return strings.Join(querys, " OR "), values, orderStr, nil
		}
	} else {
		return "1=1", []Any{}, "", nil
	}
}

func (module *dataModule) orderby(key string) string {
	dots := strings.Split(key, ".")
	if len(dots) > 1 {
		return fmt.Sprintf(`COALESCE(("%s"->'%s')::float8, 0)`, dots[0], dots[1])
	}
	return key
}

// func (module *dataModule) fieldby(key string) string {
// 	dots := strings.Split(key, ".")
// 	if len(dots) > 1 {
// 		return fmt.Sprintf(`"%s"->'%s'`, dots[0], dots[1])
// 	}
// 	return key
// }

//注意，这个是实际的解析，支持递归
func (module *dataModule) parsing(args ...Map) ([]string, []interface{}, []string) {

	querys := []string{}
	values := make([]interface{}, 0)
	orders := []string{}

	//否则是多个map,单个为 与, 多个为 或
	for _, m := range args {
		ands := []string{}

		for k, v := range m {

			// 字段名处理
			// 包含.应该是处理成json
			// 包含:就处理成数组
			jsoned := false
			if dots := strings.Split(k, ":"); len(dots) >= 2 {
				k = fmt.Sprintf(`%v%v%v[%v]`, DELIMS, dots[0], DELIMS, dots[1])
			} else if dots := strings.Split(k, "."); len(dots) >= 2 {
				//"%s"->'%s'
				jsoned = true
				k = fmt.Sprintf(`%v%v%v->>'%v'`, DELIMS, dots[0], DELIMS, dots[1])
			} else {
				k = fmt.Sprintf(`%v%v%v`, DELIMS, k, DELIMS)
			}

			//如果值是ASC,DESC，表示是排序
			//if ov,ok := v.(string); ok && (ov==ASC || ov==DESC) {
			if v == ASC {
				//正序
				orders = append(orders, fmt.Sprintf(`%s ASC`, module.orderby(k)))
			} else if v == DESC {
				//倒序
				orders = append(orders, fmt.Sprintf(`%s DESC`, module.orderby(k)))

			} else if v == RAND {
				//随机排序
				orders = append(orders, fmt.Sprintf(`%s ASC`, RANDBY))

			} else if v == nil {
				ands = append(ands, fmt.Sprintf(`%s IS NULL`, k))
			} else if v == NIL {
				ands = append(ands, fmt.Sprintf(`%s IS NULL`, k))
			} else if v == NOL {
				//不为空值
				ands = append(ands, fmt.Sprintf(`%s IS NOT NULL`, k))
				/*
				   }  else if _,ok := v.(Nil); ok {
				       //为空值
				       ands = append(ands, fmt.Sprintf(`%s IS NULL`, k))
				   } else if _,ok := v.(NotNil); ok {
				       //不为空值
				       ands = append(ands, fmt.Sprintf(`%s IS NOT NULL`, k))
				   } else if fts,ok := v.(FTS); ok {
				       //处理模糊搜索，此条后续版本会移除
				       safeFts := strings.Replace(string(fts), "'", "''", -1)
				       ands = append(ands, fmt.Sprintf(`%s LIKE '%%%s%%'`, k, safeFts))
				*/
			} else if ms, ok := v.([]Map); ok {
				//是[]Map，相当于or

				qs, vs, os := module.parsing(ms...)
				if len(qs) > 0 {
					ands = append(ands, fmt.Sprintf("(%s)", strings.Join(qs, " OR ")))
					for _, vsVal := range vs {
						values = append(values, vsVal)
					}
				}
				for _, osVal := range os {
					orders = append(orders, osVal)
				}

			} else if opMap, opOK := v.(Map); opOK {
				//v要处理一下如果是map要特别处理
				//key做为操作符，比如 > < >= 等
				//而且多个条件是and，比如 views > 1 AND views < 100
				//自定义操作符的时候，可以用  is not null 吗？
				//hai yao chu li INC in change update

				opAnds := []string{}
				for opKey, opVal := range opMap {
					//这里要支持LIKE
					if opKey == SEARCH {
						safeFts := strings.Replace(fmt.Sprintf("%v", opVal), "'", "''", -1)
						opAnds = append(opAnds, fmt.Sprintf(`upper(%s) LIKE upper('%%%s%%')`, k, safeFts))
					} else if opKey == FULLLIKE {
						safeFts := strings.Replace(fmt.Sprintf("%v", opVal), "'", "''", -1)
						opAnds = append(opAnds, fmt.Sprintf(`upper(%s) LIKE upper('%%%s%%')'`, k, safeFts))
					} else if opKey == LEFTLIKE {
						safeFts := strings.Replace(fmt.Sprintf("%v", opVal), "'", "''", -1)
						opAnds = append(opAnds, fmt.Sprintf(`upper(%s) LIKE upper('%s%%')`, k, safeFts))
					} else if opKey == RIGHTLIKE {
						safeFts := strings.Replace(fmt.Sprintf("%v", opVal), "'", "''", -1)
						opAnds = append(opAnds, fmt.Sprintf(`upper(%s) LIKE upper('%%%s')`, k, safeFts))
					} else if opKey == ANY {
						opAnds = append(opAnds, fmt.Sprintf(`? = ANY(%s)`, k))
						values = append(values, opVal)
						// } else if opKey == CON {
						// 	opAnds = append(opAnds, fmt.Sprintf(`%s @> ?`, k))
						// 	values = append(values, opVal)
						// } else if opKey == CONBY {
						// 	opAnds = append(opAnds, fmt.Sprintf(`%s <@ ?`, k))
						// 	values = append(values, opVal)
					} else if opKey == CON {
						// array contains array @>

						realArgs := []string{}
						realVals := []Any{}
						switch vs := opVal.(type) {
						case []int:
							if len(vs) > 0 {
								for _, v := range vs {
									realArgs = append(realArgs, "?::int8")
									realVals = append(realVals, v)
								}
							} else {
								realArgs = append(realArgs, "?")
								realVals = append(realVals, 0)
							}
						case []int64:
							if len(vs) > 0 {
								for _, v := range vs {
									realArgs = append(realArgs, "?::int8")
									realVals = append(realVals, v)
								}
							} else {
								realArgs = append(realArgs, "?")
								realVals = append(realVals, 0)
							}
						case []string:
							if len(vs) > 0 {
								for _, v := range vs {
									realArgs = append(realArgs, "?")
									realVals = append(realVals, v)
								}
							} else {
								realArgs = append(realArgs, "?")
								realVals = append(realVals, 0)
							}
						case []Any:
							if len(vs) > 0 {
								for _, v := range vs {
									realArgs = append(realArgs, "?")
									realVals = append(realVals, v)
								}
							} else {
								realArgs = append(realArgs, "?")
								realVals = append(realVals, 0)
							}
						default:
							realArgs = append(realArgs, "?")
							realVals = append(realVals, vs)
						}

						opAnds = append(opAnds, fmt.Sprintf(`%s @> ARRAY[%s]`, k, strings.Join(realArgs, ",")))
						for _, v := range realVals {
							values = append(values, v)
						}

					} else if opKey == CONBY {
						// array contains by array <@

						realArgs := []string{}
						realVals := []Any{}
						switch vs := opVal.(type) {
						case []int:
							if len(vs) > 0 {
								for _, v := range vs {
									realArgs = append(realArgs, "?::int8")
									realVals = append(realVals, v)
								}
							} else {
								realArgs = append(realArgs, "?")
								realVals = append(realVals, 0)
							}
						case []int64:
							if len(vs) > 0 {
								for _, v := range vs {
									realArgs = append(realArgs, "?::int8")
									realVals = append(realVals, v)
								}
							} else {
								realArgs = append(realArgs, "?")
								realVals = append(realVals, 0)
							}
						case []string:
							if len(vs) > 0 {
								for _, v := range vs {
									realArgs = append(realArgs, "?")
									realVals = append(realVals, v)
								}
							} else {
								realArgs = append(realArgs, "?")
								realVals = append(realVals, 0)
							}
						case []Any:
							if len(vs) > 0 {
								for _, v := range vs {
									realArgs = append(realArgs, "?")
									realVals = append(realVals, v)
								}
							} else {
								realArgs = append(realArgs, "?")
								realVals = append(realVals, 0)
							}
						default:
							realArgs = append(realArgs, "?")
							realVals = append(realVals, vs)
						}

						opAnds = append(opAnds, fmt.Sprintf(`%s <@ ARRAY[%s]`, k, strings.Join(realArgs, ",")))
						for _, v := range realVals {
							values = append(values, v)
						}
						opAnds = append(opAnds, fmt.Sprintf(`%s <@ '{}'`, k))

					} else if opKey == OR {

						realArgs := []string{}
						realVals := []Any{}
						if vvs, ok := opVal.([]Any); ok {
							for _, vv := range vvs {
								if vv == nil {
									realArgs = append(realArgs, fmt.Sprintf(`%s is null`, k))
								} else {
									realArgs = append(realArgs, fmt.Sprintf(`%s=?`, k))
									realVals = append(realVals, vv)
								}

							}
						} else if vvs, ok := opVal.([]int64); ok {
							for _, vv := range vvs {
								realArgs = append(realArgs, fmt.Sprintf(`%s=?`, k))
								realVals = append(realVals, vv)
							}
						} else if vvs, ok := opVal.([]float64); ok {
							for _, vv := range vvs {
								realArgs = append(realArgs, fmt.Sprintf(`%s=?`, k))
								realVals = append(realVals, vv)
							}
						} else if vvs, ok := opVal.([]string); ok {
							for _, vv := range vvs {
								realArgs = append(realArgs, fmt.Sprintf(`%s=?`, k))
								realVals = append(realVals, vv)
							}
						}

						opAnds = append(opAnds, strings.Join(realArgs, " OR "))
						for _, v := range realVals {
							values = append(values, v)
						}

					} else if opKey == NOR {

						realArgs := []string{}
						realVals := []Any{}
						incNull := true
						if vvs, ok := opVal.([]Any); ok {
							for _, vv := range vvs {
								if vv == nil {
									incNull = false
								} else {
									realArgs = append(realArgs, fmt.Sprintf(`%s=?`, k))
									realVals = append(realVals, vv)
								}
							}
						} else if vvs, ok := opVal.([]int64); ok {
							for _, vv := range vvs {
								realArgs = append(realArgs, fmt.Sprintf(`%s=?`, k))
								realVals = append(realVals, vv)
							}
						} else if vvs, ok := opVal.([]float64); ok {
							for _, vv := range vvs {
								realArgs = append(realArgs, fmt.Sprintf(`%s==?`, k))
								realVals = append(realVals, vv)
							}
						} else if vvs, ok := opVal.([]string); ok {
							for _, vv := range vvs {
								realArgs = append(realArgs, fmt.Sprintf(`%s==?`, k))
								realVals = append(realVals, vv)
							}
						}

						if incNull {
							opAnds = append(opAnds, fmt.Sprintf(`NOT (%s) or %s is null`, strings.Join(realArgs, " OR "), k))
						} else {
							opAnds = append(opAnds, fmt.Sprintf(`NOT (%s)`, strings.Join(realArgs, " OR ")))
						}

						for _, v := range realVals {
							values = append(values, v)
						}

					} else if opKey == IN {
						//IN (?,?,?)

						realArgs := []string{}
						realVals := []Any{}
						switch vs := opVal.(type) {
						case []int:
							if len(vs) > 0 {
								for _, v := range vs {
									realArgs = append(realArgs, "?")
									realVals = append(realVals, v)
								}
							} else {
								realArgs = append(realArgs, "?")
								realVals = append(realVals, 0)
							}
						case []int64:
							if len(vs) > 0 {
								for _, v := range vs {
									realArgs = append(realArgs, "?")
									realVals = append(realVals, v)
								}
							} else {
								realArgs = append(realArgs, "?")
								realVals = append(realVals, 0)
							}
						case []string:
							if len(vs) > 0 {
								for _, v := range vs {
									realArgs = append(realArgs, "?")
									realVals = append(realVals, v)
								}
							} else {
								realArgs = append(realArgs, "?")
								realVals = append(realVals, 0)
							}
						case []Any:
							if len(vs) > 0 {
								for _, v := range vs {
									realArgs = append(realArgs, "?")
									realVals = append(realVals, v)
								}
							} else {
								realArgs = append(realArgs, "?")
								realVals = append(realVals, 0)
							}
						default:
							realArgs = append(realArgs, "?")
							realVals = append(realVals, vs)
						}

						opAnds = append(opAnds, fmt.Sprintf(`%s IN(%s)`, k, strings.Join(realArgs, ",")))
						for _, v := range realVals {
							values = append(values, v)
						}

					} else if opKey == NIN {
						//NOT IN (?,?,?)

						realArgs := []string{}
						realVals := []Any{}
						switch vs := opVal.(type) {
						case []int:
							if len(vs) > 0 {
								for _, v := range vs {
									realArgs = append(realArgs, "?")
									realVals = append(realVals, v)
								}
							} else {
								realArgs = append(realArgs, "?")
								realVals = append(realVals, 0)
							}
						case []int64:
							if len(vs) > 0 {
								for _, v := range vs {
									realArgs = append(realArgs, "?")
									realVals = append(realVals, v)
								}
							} else {
								realArgs = append(realArgs, "?")
								realVals = append(realVals, 0)
							}
						case []string:
							if len(vs) > 0 {
								for _, v := range vs {
									realArgs = append(realArgs, "?")
									realVals = append(realVals, v)
								}
							} else {
								realArgs = append(realArgs, "?")
								realVals = append(realVals, 0)
							}
						case []Any:
							if len(vs) > 0 {
								for _, v := range vs {
									realArgs = append(realArgs, "?")
									realVals = append(realVals, v)
								}
							} else {
								realArgs = append(realArgs, "?")
								realVals = append(realVals, 0)
							}
						default:
							realArgs = append(realArgs, "?")
							realVals = append(realVals, vs)
						}

						opAnds = append(opAnds, fmt.Sprintf(`%s NOT IN(%s)`, k, strings.Join(realArgs, ",")))
						for _, v := range realVals {
							values = append(values, v)
						}

					} else {
						opAnds = append(opAnds, fmt.Sprintf(`%s %s ?`, k, opKey))
						values = append(values, opVal)
					}
				}

				ands = append(ands, fmt.Sprintf("(%s)", strings.Join(opAnds, " AND ")))

			} else {
				ands = append(ands, fmt.Sprintf(`%s = ?`, k))
				if jsoned {
					values = append(values, fmt.Sprintf("%v", v))
				} else {
					values = append(values, v)
				}
			}
		}

		if len(ands) > 0 {
			querys = append(querys, fmt.Sprintf("(%s)", strings.Join(ands, " AND ")))
		}
	}

	return querys, values, orders
}

//------ data group -------

func Base(name string) DataBase {
	return mData.Base(name)
}

func GetTable(name string) *Table {
	return mData.TableConfig(name)
}
func GetView(name string) *View {
	return mData.ViewConfig(name)
}
func GetModel(name string) *Model {
	return mData.ModelConfig(name)
}

func Field(name string, field string, exts ...Any) Var {
	return mData.Field(name, field, exts...)
}
func Fields(name string, keys []string, exts ...Vars) Vars {
	return mData.Fields(name, keys, exts...)
}
func Option(name string, field string, key string) Any {
	return mData.Option(name, field, key)
}
func Options(name string, field string) Map {
	return mData.Options(name, field)
}

func ParseSQL(args ...Any) (string, []Any, string, error) {
	return mData.Parse(args...)
}
