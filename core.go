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
			name: CHEF, role: CHEF, version: "0.0.0",
			setting: Map{},
		},

		names:   make([]string, 0),
		modules: make(map[string]Module, 0),
	}
)

func init() {
	//如果外部加载了，这里先builtin就没法执行了
	// core.builtin()
}
