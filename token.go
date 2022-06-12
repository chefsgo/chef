package chef

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	. "github.com/chefsgo/base"
	"github.com/chefsgo/util"
)

var (
	mToken = &tokenModule{
		config: tokenConfig{
			Secret: CHEFSGO,
		},
	}

	errInvalidToken = errors.New("Invalid token.")
)

type (
	tokenConfig struct {
		Tokenizer string
		Secret    string
		Expiry    time.Duration
	}

	tokenHeader struct {
		Id    string `json:"d,omitempty"`
		Begin int64  `json:"b,omitempty"`
		End   int64  `json:"e,omitempty"`
		Auth  bool   `json:"a,omitempty"`
	}
	Token struct {
		Header  tokenHeader
		Payload Map
	}

	tokenModule struct {
		mutex  sync.Mutex
		config tokenConfig
	}
)

// Register
func (module *tokenModule) Register(name string, value Any, override bool) {
	// switch val := value.(type) {
	// // case Tokenizer:
	// // 	module.Tokenizer(name, val, override)
	// }
}

// Configure
func (module *tokenModule) Configure(global Map) {
	var config Map
	if vv, ok := global["token"].(Map); ok {
		config = vv
	}

	if secret, ok := config["secret"].(string); ok {
		mToken.config.Secret = secret
	}

	//默认过期时间，单位秒
	if expiry, ok := config["expiry"].(string); ok {
		dur, err := util.ParseDuration(expiry)
		if err == nil {
			mToken.config.Expiry = dur
		}
	}
	if expiry, ok := config["expiry"].(int); ok {
		mToken.config.Expiry = time.Second * time.Duration(expiry)
	}
	if expiry, ok := config["expiry"].(float64); ok {
		mToken.config.Expiry = time.Second * time.Duration(expiry)
	}
}

func (this *tokenModule) Initialize() {
}
func (this *tokenModule) Connect() {
}
func (this *tokenModule) Launch() {
}
func (this *tokenModule) Terminate() {
}

//------------------------- 方法 ----------------------------

func (this *tokenModule) Sign(token *Token) (string, error) {
	header, payload := "{}", "{}"

	if vv, err := MarshalJSON(token.Header); err != nil {
		return "", err
	} else {
		if vvs, err := mCodec.EncryptTEXT(string(vv)); err != nil {
			return "", err
		} else {
			header = vvs
		}
	}

	if vv, err := MarshalJSON(token.Payload); err != nil {
		return "", err
	} else {
		payload = base64.URLEncoding.EncodeToString(vv)
	}

	tokenString := header + "." + payload

	//计算签名
	sign, err := hmacSign(tokenString, this.config.Secret)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.%s", tokenString, sign), nil
}

func (this *tokenModule) Verify(str string) (*Token, error) {

	alls := strings.Split(str, ".")
	if len(alls) != 3 {
		return nil, errInvalidToken
	}

	header := alls[0]
	payload := alls[1]
	sign := alls[2]

	tokenString := header + "." + payload

	//验证签名
	err := hmacVerify(tokenString, sign, this.config.Secret)
	if err != nil {
		return nil, err
	}

	token := &Token{}

	if vvs, err := mCodec.DecryptTEXT(header); err != nil {
		return nil, err
	} else {
		if err := UnmarshalJSON([]byte(vvs), &token.Header); err != nil {
			return nil, err
		}
	}

	if vvs, err := base64.URLEncoding.DecodeString(payload); err != nil {
		return nil, err
	} else {
		if err := UnmarshalJSON(vvs, &token.Payload); err != nil {
			return nil, err
		}
	}

	now := time.Now()

	//是否校验，并且在有效期以内
	if token.Header.Begin > 0 && now.Unix() < token.Header.Begin {
		token.Header.Auth = false
	}
	if token.Header.End > 0 && now.Unix() > token.Header.End {
		token.Header.Auth = false
	}

	return token, nil
}

// -------------  外部方法 -------

// Sign 生成签名
// 可以用在一些批量生成的场景
func Sign(auth bool, payload Map, ends ...time.Duration) string {
	verify := &Token{Payload: payload}
	verify.Header.Id = mCodec.Generate()
	verify.Header.Auth = auth

	now := time.Now()
	if len(ends) > 0 {
		verify.Header.End = now.Add(ends[0]).Unix()
	}
	token, err := mToken.Sign(verify)
	if err != nil {
		return ""
	}
	return token
}

// Verify
func Verify(token string) (*Token, error) {
	return mToken.Verify(token)
}
