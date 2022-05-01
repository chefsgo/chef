package chef

import (
	"fmt"
	"strings"

	. "github.com/chefsgo/base"
)

var (
	OK           = Result(0, "ok", "成功")
	Fail         = Result(1, "fail", "失败")
	Retry        = Result(2, "retry", "重试")
	Invalid      = Result(3, "invalid", "无效请求或数据")
	Nothing      = Result(4, "nothing", "无效对象")
	Unauthorized = Result(5, "unauthorized", "无权访问")
)

type (
	result struct {
		// code 状态码
		// 0 表示成功，其它表示失败
		code int
		// state 对应的状态
		state string
		//携带的参数
		args []Any
	}
)

// OK 表示Res是否成功
func (res *result) OK() bool {
	if res == nil {
		return true
	}
	return res.code == 0
}

// Fail 表示Res是否失败
func (res *result) Fail() bool {
	return !res.OK()
}

// Code 返回Res的状态码
func (res *result) Code() int {
	return res.code
}

// State 返回Res的信息
func (res *result) State() string {
	return res.state
}

// Args 返回Res携带的参数
func (res *result) Args() []Any {
	return res.args
}

// With 使用当前Res加上参数生成一个新的Res并返回
// 因为result都是预先定义好的，所以如果直接修改args，会修改本来已经定义好的result
func (res *result) With(args ...Any) Res {
	if len(args) > 0 {
		return &result{res.code, res.state, args}
	}
	return res
}

// Error 返回Res的信息以符合error接口的定义
func (res *result) Error() string {
	if res == nil {
		return ""
	}

	text := String(DEFAULT, res.state)

	if res.args != nil && len(res.args) > 0 {
		ccc := strings.Count(text, "%") - strings.Count(text, "%%")
		if ccc == len(res.args) {
			return fmt.Sprintf(text, res.args...)
		}
	}
	return text
}

func newResult(code int, text string, args ...Any) Res {
	return &result{code, text, args}
}
func codeResult(code int, args ...Any) Res {
	return &result{code, "", args}
}
func textResult(text string, args ...Any) Res {
	return &result{-1, text, args}
}
func errorResult(err error) Res {
	return &result{-1, err.Error(), []Any{}}
}

// Result 定义一个result，并自动注册state
// state 表示状态key
// text 表示状态对应的默认文案
func Result(code int, state string, text string, overrides ...bool) Res {
	override := true
	if len(overrides) > 0 {
		override = overrides[0]
	}

	//自动注册状态和字串
	mBasic.State(state, State(code), override)
	mBasic.Strings(DEFAULT, Strings{state: text}, override)

	// result只携带state，而不携带string
	// 具体的string需要配置context拿到lang之后生成
	// 而实现多语言的状态反馈
	return newResult(code, state)
}
