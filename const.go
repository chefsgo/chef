package chef

type (
	env = int
)

const (
	_ env = iota
	developing
	testing
	production
	//
)

const (
	CHEF    = "chef"
	CHEFSGO = "Chefs.go"

	DEFAULT = "default"

	UTF8   = "utf-8"
	GB2312 = "gb2312"
	GBK    = "gbk"
)
