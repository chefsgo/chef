package chef

import (
	"io/ioutil"
	"os"

	. "github.com/chefsgo/base"
)

//定义Var
//def表示默认值，不写请传nil
func Define(tttt string, require bool, def Any, name string, extends ...Any) Var {
	config := Var{
		Type: tttt, Required: require, Name: name, Default: def,
	}

	return VarExtend(config, extends...)
}

func VarExtend(config Var, extends ...Any) Var {
	if len(extends) > 0 {
		ext := extends[0]

		if extend, ok := ext.(Var); ok {
			if extend.Text != "" {
				config.Text = extend.Text
			}

			if extend.Encode != "" {
				config.Encode = extend.Encode
			}
			if extend.Decode != "" {
				config.Decode = extend.Decode
			}
			if extend.Default != nil {
				config.Default = extend.Default
			}
			if extend.Empty != nil {
				config.Empty = extend.Empty
			}
			if extend.Error != nil {
				config.Error = extend.Error
			}
			if extend.Value != nil {
				config.Value = extend.Value
			}
			if extend.Valid != nil {
				config.Valid = extend.Valid
			}

			if extend.Children != nil {
				config.Children = extend.Children
			}
			if extend.Options != nil {
				config.Options = extend.Options
			}
			if extend.Setting != nil {
				if config.Setting == nil {
					config.Setting = Map{}
				}
				for k, v := range extend.Setting {
					config.Setting[k] = v
				}
			}

		} else if extend, ok := ext.(Map); ok {

			if vv, ok := extend["require"].(bool); ok {
				config.Required = vv
			}
			if vv, ok := extend["required"].(bool); ok {
				config.Required = vv
			}
			if vv, ok := extend["bixude"].(bool); ok {
				config.Required = vv
			}
			if vv, ok := extend["must"].(bool); ok {
				config.Required = vv
			}
			if vv, ok := extend["default"]; ok {
				config.Default = vv
			}
			if vv, ok := extend["auto"]; ok {
				config.Default = vv
			}
			if vv, ok := extend["children"].(Vars); ok {
				config.Children = vv
			}
			if vv, ok := extend["json"].(Vars); ok {
				config.Children = vv
			}
			if vv, ok := extend["option"].(Map); ok {
				config.Options = vv
			}
			if vv, ok := extend["options"].(Map); ok {
				config.Options = vv
			}
			if vv, ok := extend["enum"].(Map); ok {
				config.Options = vv
			}
			if vv, ok := extend["enums"].(Map); ok {
				config.Options = vv
			}
			if vv, ok := extend["setting"].(Map); ok {
				config.Setting = vv
			}
			if vv, ok := extend["desc"].(string); ok {
				config.Text = vv
			}
			if vv, ok := extend["text"].(string); ok {
				config.Text = vv
			}

			if vv, ok := extend["encode"].(string); ok {
				config.Encode = vv
			}
			if vv, ok := extend["decode"].(string); ok {
				config.Decode = vv
			}

			if vv, ok := extend["empty"].(Res); ok {
				config.Empty = vv
			}
			if vv, ok := extend["error"].(Res); ok {
				config.Error = vv
			}

			if vv, ok := extend["valid"].(func(Any, Var) bool); ok {
				config.Valid = vv
			}

			if vv, ok := extend["value"].(func(Any, Var) Any); ok {
				config.Value = vv
			}

			if config.Setting == nil {
				config.Setting = Map{}
			}

			//除了setting，全部写到setting里
			for k, v := range extend {
				if k != "setting" {
					config.Setting[k] = v
				}
			}
		}

	}

	return config
}

func VarsExtend(config Vars, extends ...Vars) Vars {
	if len(extends) > 0 {
		for k, v := range extends[0] {
			if v.Nil() {
				delete(config, k)
			} else {
				config[k] = v
			}
		}
	}
	return config
}

func TempFile(patterns ...string) (*os.File, error) {
	pattern := ""
	if len(patterns) > 0 {
		pattern = patterns[0]
	}

	dir := os.TempDir()
	//待处理
	// if mFile.config.TempDir != "" {
	// 	dir = mFile.config.TempDir
	// }

	return ioutil.TempFile(dir, pattern)
}

func TempDir(patterns ...string) (string, error) {
	pattern := ""
	if len(patterns) > 0 {
		pattern = patterns[0]
	}

	dir := os.TempDir()
	//待处理
	// if mFile.config.TempDir != "" {
	// 	dir = mFile.config.TempDir
	// }

	return ioutil.TempDir(dir, pattern)
}
