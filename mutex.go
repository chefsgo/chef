package chef

import (
	"errors"
	"sync"
	"time"

	. "github.com/chefsgo/base"
	"github.com/chefsgo/util"
)

var (
	mMutex = &mutexModule{
		config:    make(map[string]MutexConfig, 0),
		drivers:   make(map[string]MutexDriver, 0),
		instances: make(map[string]mutexInstance, 0),
	}

	errInvalidMutexConnection = errors.New("Invalid mutex connection.")
)

type (
	MutexConfig struct {
		Driver  string        `toml:"driver"`
		Weight  int           `toml:"weight"`
		Prefix  string        `toml:"prefix"`
		Expiry  time.Duration `toml:"expiry"`
		Setting Map           `toml:"setting"`
	}
	// MutexDriver 日志驱动
	MutexDriver interface {
		Connect(name string, config MutexConfig) (MutexConnect, error)
	}
	// MutexConnect 日志连接
	MutexConnect interface {
		//打开、关闭
		Open() error
		Close() error

		Lock(key string, expiry time.Duration) error
		Unlock(key string) error
	}

	// mutexInstance 连接的实例
	mutexInstance struct {
		config  MutexConfig
		connect MutexConnect
	}

	mutexModule struct {
		mutex  sync.Mutex
		config map[string]MutexConfig

		drivers   map[string]MutexDriver
		instances map[string]mutexInstance

		weights  map[string]int
		hashring *util.HashRing
	}
)

// register 模块注册中心
func (module *mutexModule) register(name string, value Any, override bool) {
	switch config := value.(type) {
	case MutexDriver:
		module.Driver(name, config, override)
	}
}

// configure 配置
func (module *mutexModule) configure(config Map) {
	var confs Map
	if vvv, ok := config["mutex"].(Map); ok {
		confs = vvv
	}

	for key, val := range confs {
		if conf, ok := val.(Map); ok {

			cfg := MutexConfig{
				Driver: DEFAULT, Weight: 1, Expiry: time.Second,
			}

			if driver, ok := conf["driver"].(string); ok {
				cfg.Driver = driver
			}

			//分配权重
			if weight, ok := conf["weight"].(int); ok {
				cfg.Weight = weight
			}
			if weight, ok := conf["weight"].(int64); ok {
				cfg.Weight = int(weight)
			}
			if weight, ok := conf["weight"].(float64); ok {
				cfg.Weight = int(weight)
			}

			//默认过期时间，单位秒
			if expiry, ok := conf["expiry"].(string); ok {
				dur, err := util.ParseDuration(expiry)
				if err == nil {
					cfg.Expiry = dur
				}
			}
			if expiry, ok := conf["expiry"].(int); ok {
				cfg.Expiry = time.Second * time.Duration(expiry)
			}
			if expiry, ok := conf["expiry"].(float64); ok {
				cfg.Expiry = time.Second * time.Duration(expiry)
			}

			//保存配置
			module.config[key] = cfg
		}
	}
}

// Driver 注册驱动
func (module *mutexModule) Driver(name string, driver MutexDriver, overrides ...bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	if driver == nil {
		panic("Invalid mutex driver: " + name)
	}

	override := true
	if len(overrides) > 0 {
		override = overrides[0]
	}

	if override {
		module.drivers[name] = driver
	} else {
		if module.drivers[name] == nil {
			module.drivers[name] = driver
		}
	}
}

// initialize 初始化
func (module *mutexModule) initialize() {

	// 如果没有配置任何连接时，默认一个
	if len(module.config) == 0 {
		module.config[DEFAULT] = MutexConfig{
			Driver: DEFAULT, Weight: 1, Expiry: time.Second,
		}
	}

	//记录要参与分布的连接和权重
	weights := make(map[string]int)

	for name, config := range module.config {
		driver, ok := module.drivers[config.Driver]
		if ok == false {
			panic("Invalid mutex driver: " + config.Driver)
		}

		// 建立连接
		connect, err := driver.Connect(name, config)
		if err != nil {
			panic("Failed to connect to mutex: " + err.Error())
		}

		// 打开连接
		err = connect.Open()
		if err != nil {
			panic("Failed to open mutex connect: " + err.Error())
		}

		//保存实例
		module.instances[name] = mutexInstance{
			config, connect,
		}

		//只有设置了权重的才参与分布
		if config.Weight > 0 {
			weights[name] = config.Weight
		}
	}

	//hashring分片
	module.weights = weights
	module.hashring = util.NewHashRing(weights)
}

// launch mutex模块launch暂时没有用
func (module *mutexModule) launch() {
	// fmt.Println("mutex launched")
}

// terminate 结束模块
func (module *mutexModule) terminate() {
	for _, inst := range module.instances {
		inst.connect.Close()
	}
}

// Lock 加锁
func (module *mutexModule) Lock(key string, expiries ...time.Duration) error {
	locate := module.hashring.Locate(key)

	if inst, ok := module.instances[locate]; ok {

		expiry := inst.config.Expiry
		if len(expiries) > 0 {
			expiry = expiries[0]
		}

		// 加上前缀
		key := inst.config.Prefix + key

		return inst.connect.Lock(key, expiry)
	}

	return errInvalidMutexConnection
}

// LockTo 加锁到指定的连接
func (module *mutexModule) LockTo(conn string, key string, expiries ...time.Duration) error {
	if inst, ok := module.instances[conn]; ok {

		//默认过期时间
		expiry := inst.config.Expiry
		if len(expiries) > 0 {
			expiry = expiries[0]
		}

		// 加上前缀
		key := inst.config.Prefix + key

		return inst.connect.Lock(key, expiry)
	}

	return errInvalidMutexConnection
}

// Unlock 解锁
func (module *mutexModule) Unlock(key string) error {
	locate := module.hashring.Locate(key)

	if inst, ok := module.instances[locate]; ok {
		key := inst.config.Prefix + key //加上前缀
		return inst.connect.Unlock(key)
	}

	return errInvalidMutexConnection
}

// UnlockFrom 从指定的连接解锁
func (module *mutexModule) UnlockFrom(locate string, key string) error {
	if inst, ok := module.instances[locate]; ok {
		key := inst.config.Prefix + key //加上前缀
		return inst.connect.Unlock(key)
	}

	return errInvalidMutexConnection
}

func Lock(key string, expiries ...time.Duration) error {
	return mMutex.Lock(key, expiries...)
}
func Unlock(key string) error {
	return mMutex.Unlock(key)
}
func LockTo(conn string, key string, expiries ...time.Duration) error {
	return mMutex.Lock(key, expiries...)
}
func UnlockFrom(conn string, key string) error {
	return mMutex.UnlockFrom(conn, key)
}

func Locked(key string, expiries ...time.Duration) bool {
	return mMutex.Lock(key, expiries...) != nil
}
