package chef

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	. "github.com/chefsgo/base"
	"github.com/chefsgo/util"
)

var (
	mCodec = &codecModule{
		config: codecConfig{
			Text:  "01234AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz56789-_",
			Digit: "abcdefghijkmnpqrstuvwxyz123456789ACDEFGHJKLMNPQRSTUVWXYZ",
			Salt:  CHEF, Length: 7,

			Start:    time.Date(2022, 5, 1, 0, 0, 0, 0, time.Local),
			Timebits: 42, Nodebits: 7, Stepbits: 14,
			// 42bit=128年
		},
		codecs: make(map[string]Codec, 0),
	}
	errInvalidCodec     = errors.New("Invalid codec.")
	errInvalidCodecData = errors.New("Invalid codec data.")
)

const (
	jsonCodec   = "json"
	xmlCodec    = "xml"
	gobCodec    = "gob"
	digitCodec  = "digit"
	digitsCodec = "digits"
	textCodec   = "text"
	textsCodec  = "text"
)

type (
	codecConfig struct {
		// Text Text 文本加密字母表
		Text string
		// Digit Digit 数字加密字母表
		Digit string
		// Salt 数字加密，加盐
		Salt string
		// Length 数字加密，最小长度
		Length int

		//雪花ID 开始时间
		Start time.Time
		//时间位
		Timebits uint
		//节点位
		Nodebits uint
		//序列位
		Stepbits uint
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

		fastid *util.FastID
	}
)

func (module *codecModule) Configure(global Map) {
	var config Map
	if vv, ok := global["codec"].(Map); ok {
		config = vv
	}

	//字串字符表
	if text, ok := config["text"].(string); ok {
		module.config.Text = text
	}

	//数字字母表
	if digit, ok := config["digit"].(string); ok {
		module.config.Digit = digit
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

	//雪花相关配置

	//开始时间
	if vv, ok := config["start"].(time.Time); ok {
		module.config.Start = vv
	}
	if vv, ok := config["start"].(int64); ok {
		module.config.Start = time.Unix(vv, 0)
	}
	//时间位
	if vv, ok := config["timebits"].(int); ok {
		module.config.Timebits = uint(vv)
	}
	if vv, ok := config["timebits"].(int64); ok {
		module.config.Timebits = uint(vv)
	}
	//节点位
	if vv, ok := config["nodebits"].(int); ok {
		module.config.Nodebits = uint(vv)
	}
	if vv, ok := config["nodebits"].(int64); ok {
		module.config.Nodebits = uint(vv)
	}
	//序列位
	if vv, ok := config["stepbits"].(int); ok {
		module.config.Stepbits = uint(vv)
	}
	if vv, ok := config["stepbits"].(int64); ok {
		module.config.Stepbits = uint(vv)
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
	module.fastid = util.NewFastID(module.config.Timebits, module.config.Nodebits, module.config.Stepbits, module.config.Start.Unix())
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

// Sequence 雪花ID
func (module *codecModule) Sequence() int64 {
	return module.fastid.NextID()
}

// Unique 雪花ID 转数字加密
func (module *codecModule) Generate(prefixs ...string) string {
	id := module.Sequence()
	ss, err := DigitEncrypt(id)
	if err != nil {
		return fmt.Sprintf("%v", id)
	}
	if len(prefixs) > 0 {
		return fmt.Sprintf("%s%s", prefixs[0], ss)
	} else {
		return ss
	}
}

// Encode 原始的编码
func (module *codecModule) Encode(codec string, v Any) (Any, error) {
	codec = strings.ToLower(codec)
	if ccc, ok := module.codecs[codec]; ok {
		return ccc.Encode(v)
	}
	return nil, errInvalidCodec
}

// Decode 原始的解码
func (module *codecModule) Decode(codec string, d Any, v Any) (Any, error) {
	codec = strings.ToLower(codec)
	if ccc, ok := module.codecs[codec]; ok {
		return ccc.Decode(d, v)
	}
	return nil, errInvalidCodec
}

// Marshal 序列化
// 如 json, xml, gob 等
func (module *codecModule) Marshal(codec string, v Any) ([]byte, error) {
	dat, err := module.Encode(codec, v)
	if err != nil {
		return nil, err
	}
	if bts, ok := dat.([]byte); ok {
		return bts, nil
	}

	return nil, errInvalidCodecData
}

// Unmarshal 反序列化
// 如 json, xml, gob 等
func (module *codecModule) Unmarshal(codec string, d []byte, v Any) error {
	_, err := mCodec.Decode(codec, d, v)
	return err
}

// Encrypt 数据加密
// 主要用类Var中的参数，数据
// 数据加密后，要返回明文可读的字串，方便传递
func (module *codecModule) Encrypt(codec string, v Any) (string, error) {
	dat, err := module.Encode(codec, v)
	if err != nil {
		return "", err
	}
	if bts, ok := dat.(string); ok {
		return bts, nil
	}
	if bts, ok := dat.([]byte); ok {
		return string(bts), nil
	}

	return "", errInvalidCodecData
}

// Decrypt 数据解密
// 主要用类Var中的参数，数据
func (module *codecModule) Decrypt(codec string, v Any) (Any, error) {
	return mCodec.Decode(codec, v, nil)
}

// Codecs 所有编解码器
func Codecs() map[string]Codec {
	return mCodec.Codecs()
}

// CodecStrings
func CodecConfig() codecConfig {
	return mCodec.config
}

// Sequence 雪花ID
func Sequence() int64 {
	return mCodec.Sequence()
}

// Unique 雪花ID 转数字加密
func Generate(prefixs ...string) string {
	return mCodec.Generate()
}

// Encode 对象公开的编码
// 原始先不公开方便了吧
// func Encode(codec string, v Any) (Any, error) {
// 	return mCodec.Encode(codec, v)
// }
// func Decode(codec string, d Any, v Any) (Any, error) {
// 	return mCodec.Decode(codec, d, v)
// }

// Marshal 序列化
// 如 json, xml, gob 等
func Marshal(codec string, v Any) ([]byte, error) {
	return mCodec.Marshal(codec, v)
}

// Unmarshal 反序列化
// 如 json, xml, gob 等
func Unmarshal(codec string, d []byte, v Any) error {
	return mCodec.Unmarshal(codec, d, v)
}

// Encrypt 数据加密
// 主要用类Var中的参数，数据
// 数据加密后，要返回明文可读的字串，方便传递
func Encrypt(codec string, v Any) (string, error) {
	return mCodec.Encrypt(codec, v)
}

// Decrypt 数据解密
// 主要用类Var中的参数，数据
func Decrypt(codec string, v Any) (Any, error) {
	return mCodec.Decrypt(codec, v)
}

// JSONMarshal
func JSONMarshal(v Any) ([]byte, error) {
	return mCodec.Marshal(jsonCodec, v)
}

// MarshalJSON alias for JSONMarshal
func MarshalJSON(v Any) ([]byte, error) {
	return mCodec.Marshal(jsonCodec, v)
}

// JSONUnmarshal
func JSONUnmarshal(d []byte, v Any) error {
	return mCodec.Unmarshal(jsonCodec, d, v)
}

// UnmarshalJSON alias for JSONUnmarshal
func UnmarshalJSON(d []byte, v Any) error {
	return mCodec.Unmarshal(jsonCodec, d, v)
}

// XMLMarshal
func XMLMarshal(v Any) ([]byte, error) {
	return mCodec.Marshal(xmlCodec, v)
}

// MarshalXML alias for XMLMarshal
func MarshalXML(v Any) ([]byte, error) {
	return mCodec.Marshal(xmlCodec, v)
}

// XMLUnmarshal
func XMLUnmarshal(d []byte, v Any) error {
	return mCodec.Unmarshal(xmlCodec, d, v)
}

// UnmarshalXML alias for XMLUnmarshal
func UnmarshalXML(d []byte, v Any) error {
	return mCodec.Unmarshal(xmlCodec, d, v)
}

// GOBMarshal
func GOBMarshal(v Any) ([]byte, error) {
	return mCodec.Marshal(gobCodec, v)
}

// MarshalGOB alias for GOBMarshal
func MarshalGOB(v Any) ([]byte, error) {
	return mCodec.Marshal(gobCodec, v)
}

// GOBUnmarshal
func GOBUnmarshal(d []byte, v Any) error {
	return mCodec.Unmarshal(gobCodec, d, v)
}

// UnmarshalGOB alias for GOBUnmarshal
func UnmarshalGOB(d []byte, v Any) error {
	return mCodec.Unmarshal(gobCodec, d, v)
}

// DigitEncrypt 加密数字
func DigitEncrypt(n int64) (string, error) {
	return mCodec.Encrypt(digitCodec, n)
}

// EncryptDigit alias for DigitEncrypt
func EncryptDigit(n int64) (string, error) {
	return mCodec.Encrypt(digitCodec, n)
}

// DigitsEncrypt 数字数组加密
func DigitsEncrypt(ns []int64) (string, error) {
	return mCodec.Encrypt(digitsCodec, ns)
}

// DigitsEncrypt alias for DigitsEncrypt
func EncryptDigits(ns []int64) (string, error) {
	return mCodec.Encrypt(digitsCodec, ns)
}

// DecryptDigit 解码数字
func DecryptDigit(s string) (int64, error) {
	val, err := mCodec.Decrypt(digitCodec, s)
	if err != nil {
		return -1, err
	}

	if num, ok := val.(int); ok {
		return int64(num), nil
	}
	if num, ok := val.(int64); ok {
		return num, nil
	}

	return -1, errInvalidCodec
}

// DigitDecrypt alias for DecryptDigit
func DigitDecrypt(s string) (int64, error) {
	return DecryptDigit(s)
}

// DecryptDigits 解码数字数组
func DecryptDigits(s string) ([]int64, error) {
	val, err := mCodec.Decrypt(digitCodec, s)
	if err != nil {
		return nil, err
	}

	if num, ok := val.(int); ok {
		return []int64{int64(num)}, nil
	}
	if num, ok := val.(int64); ok {
		return []int64{num}, nil
	}
	if num, ok := val.([]int64); ok {
		return num, nil
	}

	return nil, errInvalidCodec
}

// DigitsDecrypt alias for DecryptDigits
func DigitsDecrypt(s string) ([]int64, error) {
	return DecryptDigits(s)
}

// TextEncrypt 加密文本
func TextEncrypt(n string) (string, error) {
	return mCodec.Encrypt(textCodec, n)
}

// EncryptText alias for TextEncrypt
func EncryptText(n string) (string, error) {
	return mCodec.Encrypt(textCodec, n)
}

// TextsEncrypt 文本数组加密
func TextsEncrypt(ns []string) (string, error) {
	return mCodec.Encrypt(textsCodec, ns)
}

// TextsEncrypt alias for TextsEncrypt
func EncryptTexts(ns []string) (string, error) {
	return mCodec.Encrypt(textsCodec, ns)
}

// DecryptText 解码文本
func DecryptText(s string) (string, error) {
	val, err := mCodec.Decrypt(textCodec, s)
	if err != nil {
		return "", err
	}

	if sss, ok := val.(string); ok {
		return sss, nil
	}

	return "", errInvalidCodec
}

// TextDecrypt alias for DecryptText
func TextDecrypt(s string) (string, error) {
	return DecryptText(s)
}

// DecryptTexts 解码文本数组
func DecryptTexts(s string) ([]string, error) {
	val, err := mCodec.Decrypt(textCodec, s)
	if err != nil {
		return nil, err
	}

	if num, ok := val.(string); ok {
		return []string{num}, nil
	}
	if num, ok := val.([]string); ok {
		return num, nil
	}

	return nil, errInvalidCodec
}

// TextsDecrypt alias for DecryptTexts
func TextsDecrypt(s string) ([]string, error) {
	return DecryptTexts(s)
}
