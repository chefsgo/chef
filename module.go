package chef

import (
	. "github.com/chefsgo/base"
)

type (
	Module interface {
		Register(name string, value Any, override bool)
		Configure(Map)
		Initialize()
		Connect()
		Launch()
		Terminate()
	}
)

func init() {
	Register(mBasic)
	Register(mCodec)
	Register(mEngine)
}
