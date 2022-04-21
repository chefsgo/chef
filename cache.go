package chef

import (
	"errors"
	"sync"
	"time"

	. "github.com/chefsgo/base"
	"github.com/chefsgo/util"
)

var (
	mCache = &cacheModule{
		configs:   make(map[string]CacheConfig, 0),
		drivers:   make(map[string]CacheDriver, 0),
		instances: make(map[string]cacheInstance, 0),
	}

	errInvalidCacheConnection = errors.New("Invalid cache connection.")
)

type (
	CacheConfig struct {
		Driver  string
		Weight  int
		Prefix  string
		Expiry  time.Duration
		Setting Map
	}
	// CacheDriver 数据驱动
	CacheDriver interface {
		Connect(name string, config CacheConfig) (CacheConnect, error)
	}

	// CacheConnect 缓存连接
	CacheConnect interface {
		Open() error
		Close() error

		Read(string) (Any, error)
		Write(key string, val Any, expiry time.Duration) error
		Exists(key string) (bool, error)
		Delete(key string) error
		Serial(key string, start, step int64) (int64, error)
		Keys(prefix string) ([]string, error)
		Clear(prefix string) error
	}

	// cacheInstance 缓存实例
	cacheInstance struct {
		config  CacheConfig
		connect CacheConnect
	}

	cacheModule struct {
		mutex   sync.Mutex
		configs map[string]CacheConfig
		drivers map[string]CacheDriver

		instances map[string]cacheInstance

		weights  map[string]int
		hashring *util.HashRing
	}
)

// register 模块注册中心
func (module *cacheModule) Register(name string, value Any, override bool) {
	switch config := value.(type) {
	case CacheDriver:
		module.Driver(name, config, override)
	}
}

// 处理单个配置
func (module *cacheModule) configure(name string, config Map) {
	cfg := CacheConfig{
		Driver: DEFAULT, Weight: 1, Expiry: time.Hour * 24,
	}
	//如果已经存在了，用现成的改写
	if vv, ok := module.configs[name]; ok {
		cfg = vv
	}

	if driver, ok := config["driver"].(string); ok {
		cfg.Driver = driver
	}

	//分配权重
	if weight, ok := config["weight"].(int); ok {
		cfg.Weight = weight
	}
	if weight, ok := config["weight"].(int64); ok {
		cfg.Weight = int(weight)
	}
	if weight, ok := config["weight"].(float64); ok {
		cfg.Weight = int(weight)
	}

	//默认过期时间，单位秒
	if expiry, ok := config["expiry"].(string); ok {
		dur, err := util.ParseDuration(expiry)
		if err == nil {
			cfg.Expiry = dur
		}
	}
	if expiry, ok := config["expiry"].(int); ok {
		cfg.Expiry = time.Second * time.Duration(expiry)
	}
	if expiry, ok := config["expiry"].(float64); ok {
		cfg.Expiry = time.Second * time.Duration(expiry)
	}

	// cache 全部参与分布
	if cfg.Weight <= 0 {
		cfg.Weight = 1
	}

	//保存配置
	module.configs[name] = cfg
}

func (module *cacheModule) Configure(config Map) {
	var confs Map
	if vvv, ok := config["cache"].(Map); ok {
		confs = vvv
	}

	//记录上一层的配置，如果有的话
	defConfig := Map{}

	for key, val := range confs {
		if conf, ok := val.(Map); ok {
			//直接注册，然后删除当前key
			module.configure(key, conf)
		} else {
			//记录上一层的配置，如果有的话
			defConfig[key] = val
		}
	}

	if len(defConfig) > 0 {
		module.configure(DEFAULT, defConfig)
	}
}

// Driver 注册驱动
func (module *cacheModule) Driver(name string, driver CacheDriver, overrides ...bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	if driver == nil {
		panic("Invalid cache driver: " + name)
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
func (module *cacheModule) Initialize() {

	// 如果没有配置任何连接时，默认一个
	if len(module.configs) == 0 {
		module.configs[DEFAULT] = CacheConfig{
			Driver: DEFAULT, Weight: 1, Expiry: time.Hour * 24,
		}
	}

	//记录要参与分布的连接和权重
	weights := make(map[string]int)

	for name, config := range module.configs {
		driver, ok := module.drivers[config.Driver]
		if ok == false {
			panic("Invalid cache driver: " + config.Driver)
		}

		// 建立连接
		connect, err := driver.Connect(name, config)
		if err != nil {
			panic("Failed to connect to cache: " + err.Error())
		}

		// 打开连接
		err = connect.Open()
		if err != nil {
			panic("Failed to open cache connect: " + err.Error())
		}

		//保存连接
		module.instances[name] = cacheInstance{
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

// launch cache模块launch暂时没有用
func (module *cacheModule) Launch() {
	// fmt.Println("cache launched")
}

// terminate 结束模块
func (module *cacheModule) Terminate() {
	for _, ins := range module.instances {
		ins.connect.Close()
	}
}

func (module *cacheModule) Read(id string) (Any, error) {
	locate := module.hashring.Locate(id)

	if inst, ok := module.instances[locate]; ok {
		key := inst.config.Prefix + id //加前缀
		return inst.connect.Read(key)
	}

	return nil, errInvalidCacheConnection
}

func (module *cacheModule) Exists(id string) (bool, error) {
	locate := module.hashring.Locate(id)

	if inst, ok := module.instances[locate]; ok {
		key := inst.config.Prefix + id //加前缀
		return inst.connect.Exists(key)
	}

	return false, errInvalidCacheConnection
}

// Write 写缓存
func (module *cacheModule) Write(key string, val Map, expiries ...time.Duration) error {
	locate := module.hashring.Locate(key)

	if inst, ok := module.instances[locate]; ok {
		expiry := inst.config.Expiry
		if len(expiries) > 0 {
			expiry = expiries[0]
		}

		//KEY加上前缀
		key := inst.config.Prefix + key

		return inst.connect.Write(key, val, expiry)
	}

	return errInvalidCacheConnection
}

// Delete 删除缓存
func (module *cacheModule) Delete(key string) error {
	locate := module.hashring.Locate(key)

	if inst, ok := module.instances[locate]; ok {
		key := inst.config.Prefix + key
		return inst.connect.Delete(key)
	}

	return errInvalidCacheConnection
}

// Serial 生成序列编号
func (module *cacheModule) Serial(key string, start, step int64) (int64, error) {
	locate := module.hashring.Locate(key)

	if inst, ok := module.instances[locate]; ok {
		key := inst.config.Prefix + key
		return inst.connect.Serial(key, start, step)
	}

	return -1, errInvalidCacheConnection
}

// Keys 获取所有前缀的KEYS
func (module *cacheModule) Keys(prefix string) ([]string, error) {
	keys := make([]string, 0)

	for _, inst := range module.instances {
		prefix := inst.config.Prefix + prefix
		temps, err := inst.connect.Keys(prefix)
		if err == nil {
			keys = append(keys, temps...)
		}
	}

	return keys, nil
}

// Clear 按前缀清理缓存
func (module *cacheModule) Clear(prefix string) error {
	for _, inst := range module.instances {
		prefix := inst.config.Prefix + prefix
		inst.connect.Clear(prefix)
	}

	return errInvalidCacheConnection
}
