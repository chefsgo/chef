package chef

import (
	"strings"

	. "github.com/chefsgo/base"
)

type (
	library struct {
		name     string
		cardinal int
	}
)

//------------ library ----------------

func (lib *library) Name() string {
	return lib.name
}
func (lib *library) Register(name string, value Any, overrides ...bool) {
	// override := true
	// if len(overrides) > 0 {
	// 	override = overrides[0]
	// }

	if lib.name != "" && !strings.HasPrefix(name, lib.name+".") && lib.name != "" {
		name = lib.name + "." + name
	}

	args := make([]Any, 0)
	args = append(args, name, value)
	if len(overrides) > 0 {
		args = append(args, overrides[0])
	}

	Register(args...)

	// mEngine.Register(name, value, override)
}

func (lib *library) Result(ok bool, state string, text string, overrides ...bool) Res {
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

func Library(name string, cardinals ...int) *library {

	cardinal := 1000
	if len(cardinals) > 0 {
		cardinal = cardinals[0]
	}
	return &library{name, cardinal}
}
