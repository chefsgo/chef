package chef

import (
	"sync"
	"time"

	. "github.com/chefsgo/base"
)

type (
	// Meta 元数据
	// methet, service, queue, event
	// 调用的时候传输要的，在 context 里面用
	Meta struct {
		Name     string `json:"n,omitempty"`
		Payload  Map    `json:"p,omitempty"`
		Retries  int    `json:"r,omitempty"`
		Language string `json:"l,omitempty"`
		Timezone int    `json:"o,omitempty"`
		Token    string `json:"t,omitempty"`
		Trace    string `json:"i,omitempty"`
	}
	Context struct {
		mutex  sync.RWMutex
		result Res

		//待处理
		// tempfiles []string
		// databases map[string]DataBase

		// 调用时的header数据
		meta     Meta
		language string
		timezone *time.Location

		// token *Token	//待处理

		// Name 调用的方法名
		Name string

		// Setting调用时的setting，不是method.setting
		// 主要是Library和Logic时，传入setting
		Setting Map

		// Value 调用时传过来的值
		Value Map

		// Args 调用时解析后的参数
		Args Map
	}
)

func newContext(metas ...Meta) *Context {
	ctx := &Context{
		language: DEFAULT,
		timezone: time.Local,

		//待处理，meta并且拥有 language, timezone，要不太乱了

		//待处理
		// tempfiles: make([]string, 0),
		// databases: make(map[string]DataBase),
	}
	if len(metas) > 0 {
		meta := metas[0]

		if meta.Language != "" {
			ctx.SetLanguage(meta.Language)
		}
		if meta.Timezone > 0 {
			zone := time.FixedZone("", meta.Timezone)
			ctx.SetTimezone(zone)
		}
		if meta.Payload != nil {
			ctx.Value = meta.Payload
		}

		ctx.meta = meta
	}

	if ctx.meta.Trace == "" {
		ctx.meta.Trace = Generate()
	}

	return ctx
}

// Meta 获取Meta
func (ctx *Context) Meta() Meta {
	return ctx.meta
}

// Language 获取当前语言
func (ctx *Context) Language() string {
	return ctx.language
}

// SetLanguage 修改当前语言
// 待优化，从langs配置里做匹配，不匹配的不让设置
// 最好是按accepts匹配
func (ctx *Context) SetLanguage(lang string) {
	ctx.language = lang
}

// Timezone 获取当前时区
func (ctx *Context) Timezone() *time.Location {
	if ctx.timezone == nil {
		ctx.timezone = time.Local
	}
	return ctx.timezone
}

// SetTimezone 设置当前时区
func (ctx *Context) SetTimezone(zone *time.Location) {
	ctx.timezone = zone
}

// Retries 重试次数 0 为还没重试
func (ctx *Context) Retries() int {
	return ctx.meta.Retries
}

// Trace 追踪ID
func (ctx *Context) Trace() string {
	if ctx == nil {
		return ""
	}
	return ctx.meta.Trace
}

// Token 令牌
func (ctx *Context) Token() string {
	if ctx == nil {
		return ""
	}
	//20211108关闭外部修改
	// if len(tokens) > 0 && tokens[0] != "" {
	// 	ctx.token = tokens[0]
	// }
	return ctx.meta.Token
}

//最终的清理工作
// 待处理
func (ctx *Context) End() {
	// for _, file := range ctx.tempfiles {
	// 	os.Remove(file)
	// }
	// for _, base := range ctx.databases {
	// 	base.Close()
	// }
}

//待处理
// func (ctx *Context) dataBase(bases ...string) DataBase {
// 	inst := mData.Instance(bases...)

// 	if _, ok := ctx.databases[inst.name]; ok == false {
// 		ctx.databases[inst.name] = inst.connect.Base()
// 	}
// 	return ctx.databases[inst.name]
// }

//返回最后的错误信息
//获取操作结果
func (ctx *Context) Result(res ...Res) Res {
	if len(res) > 0 {
		err := res[0]
		ctx.result = err
		return err
	} else {
		if ctx.result == nil {
			return nil //nil 也要默认是成功
		}
		err := ctx.result
		ctx.result = nil
		return err
	}
}

//获取langString
func (ctx *Context) String(key string, args ...Any) string {
	return mBasic.String(ctx.Language(), key, args...)
}

//----------------------- 签名系统 end ---------------------------------

// ------- 服务调用 -----------------
func (ctx *Context) Invoke(name string, values ...Any) Map {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	vvv, res := mEngine.Invoke(ctx, name, value)
	ctx.result = res

	return vvv
}

func (ctx *Context) Invokes(name string, values ...Any) []Map {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	vvs, res := mEngine.Invokes(ctx, name, value)
	ctx.result = res
	return vvs
}
func (ctx *Context) Invoked(name string, values ...Any) bool {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	vvv, res := mEngine.Invoked(ctx, name, value)
	ctx.result = res
	return vvv
}
func (ctx *Context) Invoking(name string, offset, limit int64, values ...Any) (int64, []Map) {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	count, items, res := mEngine.Invoking(ctx, name, offset, limit, value)
	ctx.result = res
	return count, items
}

//集群后，此方法data不好定义，
//使用gob编码内容后，就不再需要定义data了
func (ctx *Context) Invoker(name string, values ...Any) (Map, []Map) {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	item, items, res := mEngine.Invoker(ctx, name, value)
	ctx.result = res
	return item, items
}

func (ctx *Context) Invokee(name string, values ...Any) float64 {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	count, res := mEngine.Invokee(ctx, name, value)
	ctx.result = res
	return count
}

func (ctx *Context) Logic(name string, settings ...Map) *Logic {
	return mEngine.Logic(ctx, name, settings...)
}

//------- 服务调用 end-----------------

//待处理

// //生成临时文件
// func (ctx *Context) TempFile(patterns ...string) (*os.File, error) {
// 	file, err := TempFile(patterns...)

// 	//记录临时文件
// 	ctx.mutex.Lock()
// 	ctx.tempfiles = append(ctx.tempfiles, file.Name())
// 	ctx.mutex.Unlock()

// 	return file, err
// }
// func (ctx *Context) TempDir(patterns ...string) (string, error) {
// 	name, err := TempDir(patterns...)

// 	if err == nil {
// 		//记录临时文件
// 		ctx.mutex.Lock()
// 		ctx.tempfiles = append(ctx.tempfiles, name)
// 		ctx.mutex.Unlock()
// 	}

// 	return name, err
// }

//token相关

// 待处理
// //actId
// func (ctx *Context) ActId() string {
// 	if ctx.verify != nil {
// 		return ctx.verify.ActId
// 	}
// 	return ""
// }

// //是否有合法的token
// func (ctx *Context) Tokenized() bool {
// 	if ctx.verify != nil {
// 		return true
// 	}
// 	return false
// }

// //是否通过验证
// func (ctx *Context) Authorized() bool {
// 	if ctx.verify != nil {
// 		return ctx.verify.Authorized
// 	}
// 	return false
// }

// //登录的身份信息
// func (ctx *Context) Identity() string {
// 	if ctx.verify != nil {
// 		return ctx.verify.Identity
// 	}
// 	return ""
// }

// //登录的身份信息
// func (ctx *Context) Payload() Map {
// 	if ctx.verify != nil {
// 		return ctx.verify.Payload
// 	}
// 	return nil
// }

//------------------- Process 方法 --------------------

// func (process *Context) Base(bases ...string) DataBase {
// 	return process.dataBase(bases...)
// }

func NewContext(metas ...Meta) *Context {
	return newContext(metas...)
}
