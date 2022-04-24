package chef

import (
	"flag"
	"path"
	"strings"
	"time"

	. "github.com/chefsgo/base"
	"github.com/chefsgo/util"

	"github.com/BurntSushi/toml"
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

func parseDurationFromMap(config Map, field string) time.Duration {
	if expiry, ok := config[field].(string); ok {
		dur, err := util.ParseDuration(expiry)
		if err == nil {
			return dur
		}
	}
	if expiry, ok := config[field].(int); ok {
		return time.Second * time.Duration(expiry)
	}
	if expiry, ok := config[field].(int64); ok {
		return time.Second * time.Duration(expiry)
	}

	return -1
}
