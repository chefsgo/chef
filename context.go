package chef

import (
	"os"
	"sync"
	"time"

	. "github.com/chefsgo/base"
)

type (
	context struct {
		mutex     sync.RWMutex
		lastError Res

		rayid  string
		token  string
		verify *Token

		lang     string
		zone     *time.Location
		attempts int

		tempfiles []string
		databases map[string]DataBase
	}
)

func newcontext(rayids ...string) *context {
	ctx := &context{
		lang:     DEFAULT,
		zone:     time.Local,
		attempts: 1,

		tempfiles: make([]string, 0),
		databases: make(map[string]DataBase),
	}

	if len(rayids) > 0 && rayids[0] != "" {
		ctx.rayid = rayids[0]
	} else {
		ctx.rayid = mCodec.Generate()
	}

	return ctx
}

// // rayId 上下文追踪用的，服务调用的时候传递
func (ctx *context) RayId() string {
	if ctx == nil {
		return ""
	}
	//20211108关闭外部修改
	// if len(traces) > 0 && traces[0] != "" {
	// 	ctx.trace = traces[0]
	// }
	return ctx.rayid
}

// // Token 令牌，
func (ctx *context) Token() string {
	if ctx == nil {
		return ""
	}
	//20211108关闭外部修改
	// if len(tokens) > 0 && tokens[0] != "" {
	// 	ctx.token = tokens[0]
	// }
	return ctx.token
}

// Lang 获取或设置当前上下文的语言
func (ctx *context) Lang(langs ...string) string {
	if ctx == nil {
		return DEFAULT
	}
	if len(langs) > 0 && langs[0] != "" {
		//待处理：加上配置中的语言判断，否则不修改
		ctx.lang = langs[0]
	}
	return ctx.lang
}

// Zone 获取或设置当前上下文的时区
func (ctx *context) Zone(zones ...*time.Location) *time.Location {
	if ctx == nil {
		return time.Local
	}

	if len(zones) > 0 && zones[0] != nil {
		ctx.zone = zones[0]
	}

	if ctx.zone == nil {
		ctx.zone = time.Local
	}

	return ctx.zone
}

// Attempts 尝试次数
func (ctx *context) Attempts() int {
	if ctx == nil {
		return 0
	}
	return ctx.attempts
}

//最终的清理工作
func (ctx *context) terminal() {
	for _, file := range ctx.tempfiles {
		os.Remove(file)
	}
	for _, base := range ctx.databases {
		base.Close()
	}
}

//待处理
func (ctx *context) dataBase(bases ...string) DataBase {
	inst := mData.Instance(bases...)

	if _, ok := ctx.databases[inst.name]; ok == false {
		ctx.databases[inst.name] = inst.connect.Base()
	}
	return ctx.databases[inst.name]
}

//返回最后的错误信息
//获取操作结果
func (ctx *context) Result(res ...Res) Res {
	if len(res) > 0 {
		err := res[0]
		ctx.lastError = err
		return err
	} else {
		if ctx.lastError == nil {
			return OK
		}
		err := ctx.lastError
		ctx.lastError = nil
		return err
	}
}

//获取langString
func (ctx *context) String(key string, args ...Any) string {
	return mBasic.String(ctx.Lang(), key, args...)
}

//----------------------- 签名系统 end ---------------------------------

// ------- 服务调用 -----------------
func (ctx *context) Invoke(name string, values ...Any) Map {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	vvv, res := mEngine.Invoke(ctx, name, value)
	ctx.lastError = res

	return vvv
}

func (ctx *context) Invokes(name string, values ...Any) []Map {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	vvs, res := mEngine.Invokes(ctx, name, value)
	ctx.lastError = res
	return vvs
}
func (ctx *context) Invoked(name string, values ...Any) bool {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	vvv, res := mEngine.Invoked(ctx, name, value)
	ctx.lastError = res
	return vvv
}
func (ctx *context) Invoking(name string, offset, limit int64, values ...Any) (int64, []Map) {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	count, items, res := mEngine.Invoking(ctx, name, offset, limit, value)
	ctx.lastError = res
	return count, items
}

//集群后，此方法data不好定义，
//使用gob编码内容后，就不再需要定义data了
func (ctx *context) Invoker(name string, values ...Any) (Map, []Map) {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	item, items, res := mEngine.Invoker(ctx, name, value)
	ctx.lastError = res
	return item, items
}

func (ctx *context) Invokee(name string, values ...Any) float64 {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	count, res := mEngine.Invokee(ctx, name, value)
	ctx.lastError = res
	return count
}

func (ctx *context) Logic(name string, settings ...Map) *Logic {
	return mEngine.Logic(ctx, name, settings...)
}

//------- 服务调用 end-----------------

// //语法糖
func (ctx *context) Locked(key string, expiry time.Duration) bool {
	return mMutex.Lock(key, expiry) != nil
}
func (ctx *context) Lock(key string, expiry time.Duration) error {
	return mMutex.Lock(key, expiry)
}
func (ctx *context) Unlock(key string) error {
	return mMutex.Unlock(key)
}

//生成临时文件
func (ctx *context) TempFile(patterns ...string) (*os.File, error) {
	file, err := TempFile(patterns...)

	//记录临时文件
	ctx.mutex.Lock()
	ctx.tempfiles = append(ctx.tempfiles, file.Name())
	ctx.mutex.Unlock()

	return file, err
}
func (ctx *context) TempDir(patterns ...string) (string, error) {
	name, err := TempDir(patterns...)

	if err == nil {
		//记录临时文件
		ctx.mutex.Lock()
		ctx.tempfiles = append(ctx.tempfiles, name)
		ctx.mutex.Unlock()
	}

	return name, err
}

// logger begin -------

// log begin

// //语法糖
func (ctx *context) Debug(args ...Any) {
	mLog.Debug(args...)
}
func (ctx *context) Trace(args ...Any) {
	mLog.Trace(args...)
}
func (ctx *context) Info(args ...Any) {
	mLog.Info(args...)
}
func (ctx *context) Notice(args ...Any) {
	mLog.Notice(args...)
}
func (ctx *context) Warning(args ...Any) {
	mLog.Warning(args...)
}
func (ctx *context) Panic(args ...Any) {
	mLog.Panic(args...)
}
func (ctx *context) Fatal(args ...Any) {
	mLog.Fatal(args...)
}

// log end

// logger end ----------

//token相关

//actId
func (ctx *context) ActId() string {
	if ctx.verify != nil {
		return ctx.verify.ActId
	}
	return ""
}

//是否有合法的token
func (ctx *context) Tokenized() bool {
	if ctx.verify != nil {
		return true
	}
	return false
}

//是否通过验证
func (ctx *context) Authorized() bool {
	if ctx.verify != nil {
		return ctx.verify.Authorized
	}
	return false
}

//登录的身份信息
func (ctx *context) Identity() string {
	if ctx.verify != nil {
		return ctx.verify.Identity
	}
	return ""
}

//登录的身份信息
func (ctx *context) Payload() Map {
	if ctx.verify != nil {
		return ctx.verify.Payload
	}
	return nil
}
