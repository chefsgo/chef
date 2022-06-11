package chef

import (
	"strings"
	"sync"

	. "github.com/chefsgo/base"
)

var (
	mEngine = newEngineModule()
)

func newEngineModule() *engineModule {
	return &engineModule{
		methods: make(map[string]Method, 0),
	}
}

const (
	StartTrigger = "$.ark.start"
	StopTrigger  = "$.ark.stop"
)

const (
	engineInvoke   = "invoke"
	engineInvokes  = "invokes"
	engineInvoking = "invoking"
	engineInvoked  = "invoked"
	engineInvokee  = "invokee"
	engineInvoker  = "invoker"
)

type (
	Method struct {
		Name     string   `json:"name"`
		Text     string   `json:"desc"`
		Alias    []string `json:"-"`
		Nullable bool     `json:"null"`
		Args     Vars     `json:"args"`
		Data     Vars     `json:"data"`
		Setting  Map      `json:"-"`
		Coding   bool     `json:"-"`
		Action   Any      `json:"-"`

		Token bool `json:"token"`
		Auth  bool `json:"auth"`
	}

	Context struct {
		*Meta
		Name    string
		Config  Method
		Setting Map

		Value Map
		Args  Map
	}

	engineLibrary struct {
		engine   *engineModule
		name     string
		cardinal int
	}

	Logic struct {
		meta   *Meta
		engine *engineModule

		Name    string
		Setting Map
	}

	engineModule struct {
		mutex   sync.Mutex
		methods map[string]Method
	}
)

// Builtin
func (module *engineModule) Builtin() {

}

// Register
func (module *engineModule) Register(key string, value Any, override bool) {
	switch val := value.(type) {
	case Method:
		module.Method(key, val, override)
		//待处理，服务
		// case Service:
		// 	module.Service(key, val, override)
	}
}

// Configure
func (module *engineModule) Configure(value Map) {
}

// Initialize
func (module *engineModule) Initialize() {
}

// Connect
func (module *engineModule) Connect() {
}

// Launch
func (module *engineModule) Launch() {
}

// Terminate
func (module *engineModule) Terminate() {
}

func (module *engineModule) Method(name string, config Method, override bool) {
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
			module.methods[key] = config
		} else {
			if _, ok := module.methods[key]; ok == false {
				module.methods[key] = config
			}
		}
	}
}

//给本地 invoke 的，加上远程调用
func (module *engineModule) Call(meta *Meta, name string, value Map, settings ...Map) (Map, Res, string) {
	data, callRes, tttt := module.call(meta, name, value, settings...)

	if callRes == Nothing {
		//待处理，远程调用
		// //本地不存在的时候，去总线请求
		// res, err := mBus.Request(ctx, name, value, time.Second*5)
		// if err != nil {
		// 	return nil, errorResult(err), tttt
		// }

		// //直接返因，因为mBus.Request已经处理args和data
		// return res.Data, newResult(res.Code, res.Text), res.Type

		return data, callRes, tttt

	} else if callRes == Retry {
		//待优化：非队列环境下的Retry直接改为失败
		return data, Fail, tttt
	} else {
		return data, callRes, tttt
	}
}

//真实的方法调用，纯本地调用
//此方法不能远程调用，要不然就死循环了
func (module *engineModule) call(meta *Meta, name string, value Map, settings ...Map) (Map, Res, string) {
	tttt := engineInvoke
	if _, ok := module.methods[name]; ok == false {
		return nil, Nothing, tttt
	}

	config := module.methods[name]

	ctx := &Context{Meta: meta}
	ctx.Name = name
	ctx.Config = config
	ctx.Setting = Map{}

	for k, v := range config.Setting {
		ctx.Setting[k] = v
	}
	if len(settings) > 0 {
		for k, v := range settings[0] {
			ctx.Setting[k] = v
		}
	}

	// 待处理

	//处理token
	// if config.Token && ctx.token == "" {
	// 	return nil, Unauthorized, tttt
	// }

	// if config.Auth && false == ctx.Authorized() {
	// 	return nil, Unauthorized, tttt
	// }

	if value == nil {
		value = Map{}
	}

	args := Map{}
	if config.Args != nil {
		res := mBasic.Mapping(config.Args, value, args, config.Nullable, false, ctx.Timezone())
		if res != nil && res.Fail() {
			return nil, res, tttt
		}
	}

	ctx.Value = value
	ctx.Args = args

	// process := &Process{
	// 	context: ctx, engine: module,
	// 	Name: name, Config: config, Setting: setting,
	// 	Value: value, Args: args,
	// }

	data := Map{}
	result := OK //默认为成功

	switch ff := config.Action.(type) {
	case func(*Context):
		ff(ctx)
	case func(*Context) Res:
		result = ff(ctx)
		//查询是否
	case func(*Context) bool:
		ok := ff(ctx)
		if ok {
			result = OK
		} else {
			result = Fail
		}
		//查询单个
	case func(*Context) Map:
		data = ff(ctx)
	case func(*Context) (Map, Res):
		data, result = ff(ctx)

		//查询列表
	case func(*Context) []Map:
		items := ff(ctx)
		data = Map{"items": items}
		tttt = engineInvokes
	case func(*Context) ([]Map, Res):
		items, res := ff(ctx)
		data = Map{"items": items}
		result = res
		tttt = engineInvokes

		//统计的玩法
	case func(*Context) int:
		count := ff(ctx)
		data = Map{"count": float64(count)}
		tttt = engineInvokee
	case func(*Context) int64:
		count := ff(ctx)
		data = Map{"count": float64(count)}
		tttt = engineInvokee
	case func(*Context) float64:
		count := ff(ctx)
		data = Map{"count": count}
		tttt = engineInvokee

		//查询分页的玩法
	case func(*Context) ([]Map, int64):
		items, count := ff(ctx)
		data = Map{"count": count, "items": items}
		tttt = engineInvoking
	case func(*Context) ([]Map, int64, Res):
		items, count, res := ff(ctx)
		result = res
		data = Map{"count": count, "items": items}
		tttt = engineInvoking
	case func(*Context) (int64, []Map):
		count, items := ff(ctx)
		data = Map{"count": count, "items": items}
		tttt = engineInvoking
	case func(*Context) (int64, []Map, Res):
		count, items, res := ff(ctx)
		result = res
		data = Map{"count": count, "items": items}
		tttt = engineInvoking

	case func(*Context) (Map, []Map):
		item, items := ff(ctx)
		data = Map{"item": item, "items": items}
		tttt = engineInvoker
	case func(*Context) (Map, []Map, Res):
		item, items, res := ff(ctx)
		result = res
		data = Map{"item": item, "items": items}
		tttt = engineInvoker
	}

	//参数解析
	if config.Data != nil {
		out := Map{}
		err := mBasic.Mapping(config.Data, data, out, false, false, ctx.Timezone())
		if err == nil || err.OK() {
			return out, result, tttt
		}
	}

	//参数如果解析失败，就原版返回
	return data, result, tttt
}

func (module *engineModule) Execute(meta *Meta, name string, value Map, settings ...Map) (Map, Res) {
	m, r, _ := module.Call(meta, name, value, settings...)
	return m, r
}

func (module *engineModule) Trigger(meta *Meta, name string, value Map, settings ...Map) {
	go module.Call(meta, name, value, settings...)
}

//以下几个方法要做些交叉处理
func (module *engineModule) Invoke(meta *Meta, name string, value Map, settings ...Map) (Map, Res) {
	data, res, tttt := module.Call(meta, name, value, settings...)
	if res != nil && res.Fail() {
		return nil, res
	}

	var item Map
	if tttt == engineInvoke {
		item = data
	} else if tttt == engineInvokes {
		if vvs, ok := data["items"].([]Map); ok && len(vvs) > 0 {
			item = vvs[0]
		}
	}

	return item, res
}

func (module *engineModule) Invokes(meta *Meta, name string, value Map, settings ...Map) ([]Map, Res) {
	data, res, _ := module.Call(meta, name, value, settings...)

	if res != nil && res.Fail() {
		return []Map{}, res
	}

	if results, ok := data["items"].([]Map); ok {
		return results, res
	} else if results, ok := data["items"].([]Any); ok {
		items := []Map{}
		for _, result := range results {
			if item, ok := result.(Map); ok {
				items = append(items, item)
			}
		}
		return items, res
	}
	if data != nil {
		return []Map{data}, res
	}
	return nil, res
}
func (module *engineModule) Invoked(meta *Meta, name string, value Map, settings ...Map) (bool, Res) {
	_, res, _ := module.Call(meta, name, value, settings...)
	if res == nil || res.OK() {
		return true, res
	}
	return false, res
}
func (module *engineModule) Invoking(meta *Meta, name string, offset, limit int64, value Map, settings ...Map) (int64, []Map, Res) {
	if value == nil {
		value = Map{}
	}
	value["offset"] = offset
	value["limit"] = limit

	data, res, _ := module.Call(meta, name, value, settings...)
	if res != nil && res.Fail() {
		return 0, nil, res
	}

	count, countOK := data["count"].(int64)
	items, itemsOK := data["items"].([]Map)
	if countOK && itemsOK {
		return count, items, res
	}

	return 0, []Map{data}, res
}

func (module *engineModule) Invoker(meta *Meta, name string, value Map, settings ...Map) (Map, []Map, Res) {
	data, res, _ := module.Call(meta, name, value, settings...)
	if res != nil && res.Fail() {
		return nil, nil, res
	}

	item, itemOK := data["item"].(Map)
	items, itemsOK := data["items"].([]Map)

	if itemOK && itemsOK {
		return item, items, res
	}

	return data, []Map{data}, res
}

func (module *engineModule) Invokee(meta *Meta, name string, value Map, settings ...Map) (float64, Res) {
	data, res, _ := module.Call(meta, name, value, settings...)
	if res != nil && res.Fail() {
		return 0, res
	}

	if vv, ok := data["count"].(float64); ok {
		return vv, res
	} else if vv, ok := data["count"].(int64); ok {
		return float64(vv), res
	}

	return 0, res
}

func (module *engineModule) Library(name string, cardinals ...int) *engineLibrary {
	cardinal := 1000
	if len(cardinals) > 0 {
		cardinal = cardinals[0]
	}
	return &engineLibrary{module, name, cardinal}
}
func (module *engineModule) Logic(meta *Meta, name string, settings ...Map) *Logic {
	setting := make(Map)
	if len(settings) > 0 {
		setting = settings[0]
	}
	return &Logic{meta, module, name, setting}
}

// 获取参数定义
// 支持远程获取
// 待优化
func (module *engineModule) Arguments(name string, extends ...Vars) Vars {
	args := Vars{}

	if config, ok := module.methods[name]; ok {
		for k, v := range config.Args {
			args[k] = v
		}
	} else {

		//去集群找定义，待处理

		//停用，因为注册路由的时候，集群还没有初始化，自然拿不到定义
		// vvv, err := module.core.Cluster.arguments(name)
		// if err == nil {
		// 	args = vvv
		// }
	}

	return VarsExtend(args, extends...)
}

//------------ library ----------------

func (lib *engineLibrary) Name() string {
	return lib.name
}
func (lib *engineLibrary) Register(name string, value Any, overrides ...bool) {
	override := true
	if len(overrides) > 0 {
		override = overrides[0]
	}

	if !strings.HasPrefix(name, lib.name+".") && lib.name != "" {
		name = lib.name + "." + name
	}

	mEngine.Register(name, value, override)
}

func (lib *engineLibrary) Result(ok bool, state string, text string, overrides ...bool) Res {
	code := 0
	if ok == false {
		code = lib.cardinal
		lib.cardinal++
	}

	if !strings.HasPrefix(state, lib.name+".") && lib.name != "" {
		state = lib.name + "." + state
	}
	return Result(code, state, text, overrides...)
}

//------- logic 方法 -------------
func (lgc *Logic) naming(name string) string {
	return lgc.Name + "." + name
}

func (lgc *Logic) Invoke(name string, values ...Any) Map {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	vvv, res := lgc.engine.Invoke(lgc.meta, lgc.naming(name), value, lgc.Setting)
	lgc.meta.result = res
	return vvv
}

func (logic *Logic) Invokes(name string, values ...Any) []Map {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	vvs, res := logic.engine.Invokes(logic.meta, logic.naming(name), value, logic.Setting)
	logic.meta.result = res
	return vvs
}
func (logic *Logic) Invoked(name string, values ...Any) bool {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	vvv, res := logic.engine.Invoked(logic.meta, logic.naming(name), value, logic.Setting)
	logic.meta.result = res
	return vvv
}
func (logic *Logic) Invoking(name string, offset, limit int64, values ...Any) (int64, []Map) {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	count, items, res := logic.engine.Invoking(logic.meta, logic.naming(name), offset, limit, value, logic.Setting)
	logic.meta.result = res
	return count, items
}

// gob之后，不需要再定义data模型
func (logic *Logic) Invoker(name string, values ...Any) (Map, []Map) {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	item, items, res := logic.engine.Invoker(logic.meta, logic.naming(name), value, logic.Setting)
	logic.meta.result = res
	return item, items
}

func (logic *Logic) Invokee(name string, values ...Any) float64 {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	count, res := logic.engine.Invokee(logic.meta, logic.naming(name), value, logic.Setting)
	logic.meta.result = res
	return count
}

//---------------------------- engine config data

func invokingArgsConfig(offset, limit int64, extends ...Vars) Vars {
	config := Vars{
		"offset": Var{
			Type: "int", Required: true, Default: offset, Name: "offset", Text: "offset",
		},
		"limit": Var{
			Type: "int", Required: true, Default: limit, Name: "limit", Text: "limit",
		},
	}

	return VarsExtend(config, extends...)
}
func invokingDataConfig(childrens ...Vars) Vars {
	var children Vars
	if len(childrens) > 0 {
		children = childrens[0]
	}
	config := Vars{
		"count": Var{
			Type: "int", Required: true, Default: 0, Name: "统计数", Text: "统计数",
		},
		"items": Var{
			Type: "[json]", Required: true, Name: "数据列表", Text: "数据列表",
			Children: children,
		},
	}
	return config
}

func invokesDataConfig(childrens ...Vars) Vars {
	var children Vars
	if len(childrens) > 0 {
		children = childrens[0]
	}
	config := Vars{
		"items": Var{
			Type: "[json]", Required: true, Name: "数据列表", Text: "数据列表",
			Children: children,
		},
	}
	return config
}

func invokeeDataConfig() Vars {
	config := Vars{
		"count": Var{
			Type: "float", Required: true, Default: 0, Name: "统计数", Text: "统计数",
		},
	}
	return config
}

//待处理，返回模型有点不好定义
func invokerDataConfig() Vars {
	config := Vars{
		"item": Var{
			Type: "json", Required: true, Name: "数据", Text: "数据",
			// Children: children,
		},
		"items": Var{
			Type: "[json]", Required: true, Name: "数据列表", Text: "数据列表",
			// Children: children,
		},
	}
	return config
}

//-------------------------------------------------------------------------------------------------------

func Library(name string, cardinals ...int) *engineLibrary {
	return mEngine.Library(name, cardinals...)
}

//方法参数
func Arguments(name string, extends ...Vars) Vars {
	return mEngine.Arguments(name, extends...)
}

//直接执行，同步，本地
func Execute(name string, values ...Any) (Map, Res) {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	return mEngine.Execute(nil, name, value)
}

//触发执行，异步，本地
func Trigger(name string, values ...Any) {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	mEngine.Trigger(nil, name, value)
}
