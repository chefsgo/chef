package chef

import (
	"io/ioutil"
	"os"
	"sync"
	"time"

	. "github.com/chefsgo/base"
)

type (
	Meta struct {
		name    string
		payload Map

		retries  int
		language string
		timezone int
		token    string
		trace    string

		mutex     sync.RWMutex
		result    Res
		tempfiles []string

		verify *Token
	}
	Metadata struct {
		Name     string `json:"n,omitempty"`
		Payload  Map    `json:"p,omitempty"`
		Retries  int    `json:"r,omitempty"`
		Language string `json:"l,omitempty"`
		Timezone int    `json:"z,omitempty"`
		Token    string `json:"t,omitempty"`
		Trace    string `json:"i,omitempty"`
	}
)

//最终的清理工作
func (meta *Meta) close() {
	for _, file := range meta.tempfiles {
		os.Remove(file)
	}
}

func (meta *Meta) Metadata(datas ...Metadata) Metadata {
	if len(datas) > 0 {
		data := datas[0]
		meta.name = data.Name
		meta.payload = data.Payload
		meta.retries = data.Retries
		meta.language = data.Language
		meta.timezone = data.Timezone
		meta.token = data.Token
		meta.trace = data.Trace

		if data.Token != "" {
			meta.Verify(data.Token)
		}
	}

	return Metadata{
		meta.name, meta.payload, meta.retries, meta.language, meta.timezone, meta.token, meta.trace,
	}
}

// Language 设置的时候，做一下langs的匹配
func (meta *Meta) Language(langs ...string) string {
	if len(langs) > 0 {
		meta.language = langs[0]
	}
	if meta.language == "" {
		return DEFAULT
	}
	return meta.language
}

// Timezone 获取当前时区
func (meta *Meta) Timezone(zones ...*time.Location) *time.Location {
	if len(zones) > 0 {
		_, offset := time.Now().In(zones[0]).Zone()
		meta.timezone = offset
	}
	if meta.timezone == 0 {
		return time.Local
	}
	return time.FixedZone("", meta.timezone)
}

// Retries 重试次数 0 为还没重试
func (meta *Meta) Retries() int {
	return meta.retries
}

// Trace 追踪ID
func (meta *Meta) Trace(traces ...string) string {
	if len(traces) > 0 {
		meta.trace = traces[0]
	}
	return meta.trace
}

// Token 令牌
func (meta *Meta) Token(tokens ...string) string {
	if len(tokens) > 0 {
		meta.token = tokens[0]
	}
	return meta.token
}

//返回最后的错误信息
//获取操作结果
func (meta *Meta) Result(res ...Res) Res {
	if len(res) > 0 {
		err := res[0]
		meta.result = err
		return err
	} else {
		if meta.result == nil {
			return nil //nil 也要默认是成功
		}
		err := meta.result
		meta.result = nil
		return err
	}
}

//获取langString
func (meta *Meta) String(key string, args ...Any) string {
	return mBasic.String(meta.Language(), key, args...)
}

//----------------------- 签名系统 end ---------------------------------

// ------- 服务调用 -----------------
func (meta *Meta) Invoke(name string, values ...Any) Map {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	vvv, res := mEngine.Invoke(meta, name, value)
	meta.result = res

	return vvv
}

func (meta *Meta) Invokes(name string, values ...Any) []Map {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	vvs, res := mEngine.Invokes(meta, name, value)
	meta.result = res
	return vvs
}
func (meta *Meta) Invoked(name string, values ...Any) bool {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	vvv, res := mEngine.Invoked(meta, name, value)
	meta.result = res
	return vvv
}
func (meta *Meta) Invoking(name string, offset, limit int64, values ...Any) (int64, []Map) {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	count, items, res := mEngine.Invoking(meta, name, offset, limit, value)
	meta.result = res
	return count, items
}

//集群后，此方法data不好定义，
//使用gob编码内容后，就不再需要定义data了
func (meta *Meta) Invoker(name string, values ...Any) (Map, []Map) {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	item, items, res := mEngine.Invoker(meta, name, value)
	meta.result = res
	return item, items
}

func (meta *Meta) Invokee(name string, values ...Any) float64 {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	count, res := mEngine.Invokee(meta, name, value)
	meta.result = res
	return count
}

func (meta *Meta) Logic(name string, settings ...Map) *Logic {
	return mEngine.Logic(meta, name, settings...)
}

//------- 服务调用 end-----------------

//待处理

//生成临时文件
func (meta *Meta) TempFile(patterns ...string) (*os.File, error) {
	meta.mutex.Lock()
	defer meta.mutex.Unlock()

	if meta.tempfiles == nil {
		meta.tempfiles = make([]string, 0)
	}

	file, err := tempFile(patterns...)
	meta.tempfiles = append(meta.tempfiles, file.Name())

	return file, err
}
func (meta *Meta) TempDir(patterns ...string) (string, error) {
	meta.mutex.Lock()
	defer meta.mutex.Unlock()

	if meta.tempfiles == nil {
		meta.tempfiles = make([]string, 0)
	}

	name, err := tempDir(patterns...)
	if err == nil {
		meta.tempfiles = append(meta.tempfiles, name)
	}

	return name, err
}

//token相关

// Id 是token的ID，类似与 sessionId
func (meta *Meta) Id() string {
	if meta.verify != nil {
		return meta.verify.Header.Id
	}
	return ""
}

// Tokenized 是否有合法的token
func (meta *Meta) Signed() bool {
	return meta.verify != nil
}

//是否通过验证
func (meta *Meta) Authed() bool {
	if meta.verify != nil {
		return meta.verify.Header.Auth
	}
	return false
}

// Payload Token携带的负载
func (meta *Meta) Payload() Map {
	if meta.verify != nil {
		return meta.verify.Payload
	}
	return nil
}

// Sign 生成签名
// 此方法不会更新当前上下文中的token和verify
// 可以用在一些批量生成的场景
func (meta *Meta) Sign(auth bool, payload Map, ends ...time.Duration) string {
	verify := &Token{Payload: payload}
	if tid := meta.Id(); tid != "" {
		verify.Header.Id = tid
	} else {
		verify.Header.Id = mCodec.Generate()
	}

	verify.Header.Auth = auth

	now := time.Now()
	if len(ends) > 0 {
		verify.Header.End = now.Add(ends[0]).Unix()
	}

	token, err := mToken.Sign(verify)
	if err != nil {
		meta.Result(errorResult(err))
		return ""
	}

	//这里生成，就替换上下文里的了
	meta.token = token
	meta.verify = verify

	return token
}

// Verify 验证签名
func (meta *Meta) Verify(token string) error {
	verify, err := mToken.Verify(token)
	if verify != nil {
		meta.token = token
		meta.verify = verify
	}
	return err
}

//------------------- Process 方法 --------------------

// func (process *Meta) Base(bases ...string) DataBase {
// 	return process.dataBase(bases...)
// }

// CloseMeta 所有携带Meta的Context，必须在执行完成后
// 调用 CloseMeta 来给meta做收尾的工作，主要是删除临时文件，关闭连接之类的
func CloseMeta(meta *Meta) {
	meta.close()
}

func tempFile(patterns ...string) (*os.File, error) {
	pattern := ""
	if len(patterns) > 0 {
		pattern = patterns[0]
	}

	dir := os.TempDir()
	// if core.config.TempDir != "" {
	// 	dir = core.config.TempDir
	// }

	return ioutil.TempFile(dir, pattern)
}

func tempDir(patterns ...string) (string, error) {
	pattern := ""
	if len(patterns) > 0 {
		pattern = patterns[0]
	}

	dir := os.TempDir()
	// if mFile.config.TempDir != "" {
	// 	dir = mFile.config.TempDir
	// }

	return ioutil.TempDir(dir, pattern)
}
