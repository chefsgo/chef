package chef

import (
	"errors"
	"sync"
	"time"

	. "github.com/chefsgo/base"
	"github.com/chefsgo/util"
)

var (
	mToken = &tokenModule{
		config: TokenConfig{
			Driver: DEFAULT,
		},
		drivers: map[string]TokenDriver{},
	}

	errInvalidTokenConnection = errors.New("Invalid token connection.")
)

type (
	// TokenConfig 令牌模块配置
	TokenConfig struct {
		// Driver 令牌驱动，默认为 default
		Driver string

		// Secret 密钥
		Secret string

		// Expiry 默认过期时间
		// 0 表示不过期
		Expiry time.Duration

		// Setting 是为不同驱动准备的自定义参数
		// 具体参数表，请参考各不同的驱动
		Setting Map
	}

	// TokenDriver token驱动
	TokenDriver interface {
		// 连接到驱动
		Connect(config TokenConfig) (TokenConnect, error)
	}
	// TokenConnect 令牌连接
	TokenConnect interface {
		// Open 打开连接
		Open() error
		// Close 关闭结束
		Close() error

		// Sign 签名
		Sign(*Token, time.Duration) (string, error)

		// Validate 验签
		Validate(token string) (*Token, error)
	}

	Token struct {
		ActId      string `json:"d,omitempty"`
		Authorized bool   `json:"a,omitempty"`
		Identity   string `json:"i,omitempty"`
		Payload    Map    `json:"l,omitempty"`
		Expiry     int64  `json:"e,omitempty"`
	}

	//tokenModule 令牌模块定义
	tokenModule struct {
		//mutex 锁
		mutex sync.Mutex

		//config 令牌配置
		config TokenConfig

		//drivers 驱动注册表
		drivers map[string]TokenDriver

		// connect 令牌连接
		connect TokenConnect
	}
)

// configure 为token模块加载配置
func (module *tokenModule) Configure(data Map) {
	if token, ok := data["token"].(Map); ok {
		//设置驱动
		if driver, ok := token["driver"].(string); ok {
			module.config.Driver = driver
		}

		// secret 密钥
		if secret, ok := token["secret"].(string); ok {
			module.config.Secret = secret
		}

		//默认过期时间，单位秒
		if expiry, ok := token["expiry"].(string); ok {
			dur, err := util.ParseDuration(expiry)
			if err == nil {
				module.config.Expiry = dur
			}
		}
		if expiry, ok := token["expiry"].(int); ok {
			module.config.Expiry = time.Second * time.Duration(expiry)
		}
		if expiry, ok := token["expiry"].(float64); ok {
			module.config.Expiry = time.Second * time.Duration(expiry)
		}

	}

	// fmt.Println("token configured", module.config)
}

// register 为token模块注册内容
func (module *tokenModule) Register(key string, val Any, override bool) {
	switch obj := val.(type) {
	case TokenDriver:
		module.Driver(key, obj, override)
	}
	// fmt.Println("token registered", key)
}

// initialize 初始化令牌模块
func (module *tokenModule) Initialize() {
	driver, ok := module.drivers[module.config.Driver]
	if ok == false {
		panic("Invalid token driver: " + module.config.Driver)
	}

	// 建立连接
	connect, err := driver.Connect(module.config)
	if err != nil {
		panic("Failed to connect to token: " + err.Error())
	}

	// 打开连接
	err = connect.Open()
	if err != nil {
		panic("Failed to open token connect: " + err.Error())
	}

	// 保存连接
	module.connect = connect
}

// launch 令牌模块launch暂时没有用
func (module *tokenModule) Launch() {
	// fmt.Println("token launched")
}

// terminate 关闭令牌模块
func (module *tokenModule) Terminate() {
	if module.connect != nil {
		module.connect.Close()
	}
}

//Driver 为token模块注册驱动
func (module *tokenModule) Driver(name string, driver TokenDriver, override bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	if driver == nil {
		panic("Invalid token driver: " + name)
	}

	if override {
		module.drivers[name] = driver
	} else {
		if module.drivers[name] == nil {
			module.drivers[name] = driver
		}
	}
}

// Sign 签名
func (module *tokenModule) Sign(token *Token, expiries ...time.Duration) (string, error) {
	if module.connect == nil {
		return "", errInvalidTokenConnection
	}

	expiry := module.config.Expiry
	if len(expiries) > 0 {
		expiry = expiries[0]
	}

	// 待完善
	// if data.ActId == "" {
	// 	data.ActId = mCodec.Unique()
	// }

	return module.connect.Sign(token, expiry)
}

// Validate 验签
func (module *tokenModule) Validate(token string) (*Token, error) {
	if module.connect == nil {
		return nil, errInvalidTokenConnection
	}

	return module.connect.Validate(token)
}
