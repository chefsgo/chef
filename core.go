package chef

import (
	. "github.com/chefsgo/base"
)

var (
	core = &chef{
		parsed:      false,
		initialized: false,
		connected:   false,
		launched:    false,
		config: config{
			name: CHEF, role: CHEF, version: "v0.0.0",
			setting: Map{},
		},

		modules: make([]Module, 0),
	}
)

func init() {

}
