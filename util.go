package chef

import (
	"flag"
	"path"
	"strings"

	"github.com/BurntSushi/toml"
	. "github.com/chefsgo/base"
)

type (
	flagSlice []string
)

var _ flag.Value = new(flagSlice)

func (s *flagSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *flagSlice) Set(value string) error {
	if *s == nil {
		*s = make([]string, 0, 1)
	}

	*s = append(*s, value)
	return nil
}

// getBaseWithoutExt 获取路径中的的文件名，去掉扩展名
func getBaseWithoutExt(p string) string {
	name := path.Base(p)
	ext := path.Ext(name)
	return strings.TrimSuffix(name, ext)
}

// parseTOML 解析toml文本得到配置
func parseTOML(s string) Map {
	var config Map
	toml.Decode(s, &config)
	return config
}
