package chef

import (
	"errors"
	"strings"
	"sync"
	"time"

	. "github.com/chefsgo/base"
)

var (
	mCodec = &codecModule{
		config: codecConfig{
			Numbers: "abcdefghijkmnpqrstuvwxyz123456789ACDEFGHJKLMNPQRSTUVWXYZ",
			Strings: "01234AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz56789-_",
			Salt:    CHEF,
			Length:  7,
		},
		codecs: make(map[string]Codec, 0),
	}
	errInvalidCodec = errors.New("Invalid codec.")
)

const (
	jsonCodec    = "json"
	xmlCodec     = "xml"
	gobCode      = "gob"
	numberCodec  = "number"
	numbersCodec = "numbers"
	stringCodec  = "string"
	stringsCodec = "strings"
)

type (
	codecConfig struct {
		Strings string
		Numbers string
		Salt    string
		Length  int

		Node     int
		Start    time.Time
		TimeBits uint
		NodeBits uint
		SeqBits  uint
	}

	Codec struct {
		Name    string     `json:"name"`
		Desc    string     `json:"desc"`
		Alias   []string   `json:"alias"`
		Setting Map        `json:"setting"`
		Encode  EncodeFunc `json:"-"`
		Decode  DecodeFunc `json:"-"`
	}
	EncodeFunc func(v Any) (Any, error)
	DecodeFunc func(d Any, v Any) (Any, error)

	// codecModule 是编解码模块
	// 主要用功能是 状态、多语言字串、MIME类型、正则表达式等等
	codecModule struct {
		mutex  sync.Mutex
		config codecConfig

		// codecs 编解码器集合
		codecs map[string]Codec
	}
)

func (module *codecModule) Configure(global Map) {
	var config Map
	if vv, ok := global["codec"].(Map); ok {
		config = vv
	}

	//字串字符表
	if strings, ok := config["strings"].(string); ok {
		module.config.Strings = strings
	}

	//数字字母表
	if numbers, ok := config["numbers"].(string); ok {
		module.config.Numbers = numbers
	}
	if salt, ok := config["salt"].(string); ok {
		module.config.Salt = salt
	}
	if length, ok := config["length"].(int64); ok {
		module.config.Length = int(length)
	}
	if length, ok := config["length"].(int); ok {
		module.config.Length = int(length)
	}

}

func (module *codecModule) Register(name string, value Any, override bool) {
	switch val := value.(type) {
	case Codec:
		module.Codec(name, val, override)
		// case Crypto:
		// 	module.Crypto(key, val, overrides...)
	}

	// fmt.Println("codec registered", name)
}

func (module *codecModule) Initialize() {
	// fmt.Println("codec initialized")
}

func (module *codecModule) Launch() {
	// fmt.Println("codec launched")
}

func (module *codecModule) Terminate() {
	// fmt.Println("codec terminated")
}

// Codec 注册编解码器
func (module *codecModule) Codec(name string, config Codec, override bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	alias := make([]string, 0)
	if name != "" {
		alias = append(alias, name)
	}
	if config.Alias != nil {
		alias = append(alias, config.Alias...)
	}

	for _, key := range alias {
		if override {
			module.codecs[key] = config
		} else {
			if _, ok := module.codecs[key]; ok == false {
				module.codecs[key] = config
			}
		}
	}

}

// Codecs 获取所有编解码器
func (module *codecModule) Codecs() map[string]Codec {
	codecs := map[string]Codec{}
	for k, v := range module.codecs {
		codecs[k] = v
	}
	return codecs
}

// Encode 编码
func (module *codecModule) Encode(codec string, v Any) (Any, error) {
	codec = strings.ToLower(codec)
	if ccc, ok := module.codecs[codec]; ok {
		return ccc.Encode(v)
	}
	return nil, errInvalidCodec
}

// Decode 解码
func (module *codecModule) Decode(codec string, d Any, v Any) (Any, error) {
	codec = strings.ToLower(codec)
	if ccc, ok := module.codecs[codec]; ok {
		return ccc.Decode(d, v)
	}
	return nil, errInvalidCodec
}

// EncodeNumber 编码数字
func (module *codecModule) EncodeNumber(n int64) (string, error) {
	val, err := module.Encode(numbersCodec, n)
	if err != nil {
		return "", err
	}
	if str, ok := val.(string); ok {
		return str, nil
	}

	return "", errInvalidCodec
}

// EncodeNumber 编码数字
func (module *codecModule) EncodeNumbers(ns []int64) (string, error) {
	val, err := module.Encode(numbersCodec, ns)
	if err != nil {
		return "", err
	}
	if str, ok := val.(string); ok {
		return str, nil
	}

	return "", errInvalidCodec
}

// DecodeNumber 解码数字
func (module *codecModule) DecodeNumber(s string) (int64, error) {
	val, err := module.Decode(numberCodec, s, nil)
	if err != nil {
		return -1, err
	}

	if num, ok := val.(int64); ok {
		return num, nil
	}

	return -1, errInvalidCodec
}

// DecodeNumbers 解码数字列表
func (module *codecModule) DecodeNumbers(s string) ([]int64, error) {
	val, err := module.Decode(numberCodec, s, nil)
	if err != nil {
		return nil, err
	}

	if nums, ok := val.([]int64); ok {
		return nums, nil
	}

	return nil, errInvalidCodec
}

// CodecStrings
func CodecConfig() codecConfig {
	return mCodec.config
}

// Encode 对象公开的编码
func Encode(codec string, v Any) (Any, error) {
	return mCodec.Encode(codec, v)
}
func Decode(codec string, d Any, v Any) (Any, error) {
	return mCodec.Decode(codec, d, v)
}

// JSONEncode
func JSONEncode(v Any) (Any, error) {
	return mCodec.Encode(jsonCodec, v)
}

// JSONDecode
func JSONDecode(d Any, v Any) (Any, error) {
	return mCodec.Decode(jsonCodec, d, v)
}

// XMLEncode
func XMLEncode(v Any) (Any, error) {
	return mCodec.Encode(xmlCodec, v)
}

// XMLDecode
func XMLDecode(d Any, v Any) (Any, error) {
	return mCodec.Decode(xmlCodec, d, v)
}

// GOBEncode
func GOBEncode(v Any) (Any, error) {
	return mCodec.Encode(gobCode, v)
}

// GOBDecode
func GOBDecode(d Any, v Any) (Any, error) {
	return mCodec.Decode(gobCode, d, v)
}

// EncodeNumber 编码数字
func EncodeNumber(n int64) (string, error) {
	return mCodec.EncodeNumber(n)
}

// EncodeNumber 编码数字
func EncodeNumbers(ns []int64) (string, error) {
	return mCodec.EncodeNumbers(ns)
}

// DecodeNumber 解码数字
func DecodeNumber(s string) (int64, error) {
	return mCodec.DecodeNumber(s)
}

// DecodeNumbers 解码数字列表
func DecodeNumbers(s string) ([]int64, error) {
	return mCodec.DecodeNumbers(s)
}
