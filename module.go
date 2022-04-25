package chef

import (
	. "github.com/chefsgo/base"
)

type (
	Module interface {
		Register(name string, value Any, override bool)
		Configure(Any)
		Initialize()
		Connect()
		Launch()
		Terminate()
	}
)

func init() {
	Register("basic", mBasic)
	Register("codec", mCodec)
	Register("engine", mEngine)
}
