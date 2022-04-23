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
