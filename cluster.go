package chef

import (
	"errors"
	"net"
	"strconv"
	"sync"

	. "github.com/chefsgo/base"
)

var (
	mCluster = newClusterModule()

	errInvalidClusterConnection = errors.New("Invalid cluster connection.")
)

func newClusterModule() *clusterModule {
	return &clusterModule{
		drivers: map[string]ClusterDriver{},
		config: ClusterConfig{
			Driver: DEFAULT, Host: "0.0.0.0", Port: 7000,
		},
	}
}

type (
	ClusterData   = map[string][]byte
	ClusterConfig struct {
		// Driver 集群驱动，默认default
		Driver string

		// Name 当前节点的角色
		// 注意：这只是在集群中的节点名称
		Name string

		// Host 监听主机，默认0.0.0.0
		Host string

		// Port 监听端口，默认7000
		Port int

		// Secret 密钥
		Secret string

		// Join 要加入的节点列表
		Join []string

		// Meta 当前节点携带的元数据
		Meta Map

		Setting Map
	}
	// ClusterDriver 集群驱动
	ClusterDriver interface {
		Connect(config ClusterConfig) (ClusterConnect, error)
	}
	// ClusterConnect 集群连接
	ClusterConnect interface {
		//打开、关闭
		Open() error
		Close() error

		Locate(key string) bool

		Read(key string) ([]byte, error)
		Write(key string, val []byte) error
		Delete(key string) error
		Clear(perfix string) error
		Batch(data ClusterData) error
		Fetch(prefix string) (ClusterData, error)

		Peers() []ClusterPeer
	}

	ClusterPeer struct {
		Name  string
		Host  string
		Port  int
		Meta  []byte
		State string
	}

	clusterModule struct {
		mutex  sync.Mutex
		config ClusterConfig

		drivers map[string]ClusterDriver
		connect ClusterConnect
	}
)

// configure 配置集群
func (module *clusterModule) configure(config Map) {
	var cluster = config
	if vv, ok := cluster["cluster"].(Map); ok {
		cluster = vv
	}

	//处理core中的配置
	if name, ok := cluster["name"].(string); ok {
		module.config.Name = name
	}
	if host, ok := cluster["host"].(string); ok {
		module.config.Host = host
	}

	//解析host+port
	if bind, ok := cluster["bind"].(string); ok {
		host, port, err := net.SplitHostPort(bind)
		if err == nil {
			module.config.Host = host
			pppp, e := strconv.Atoi(port)
			if e == nil {
				module.config.Port = pppp
			}
		}
	}
	if addr, ok := cluster["addr"].(string); ok {
		host, port, err := net.SplitHostPort(addr)
		if err == nil {
			module.config.Host = host
			pppp, e := strconv.Atoi(port)
			if e == nil {
				module.config.Port = pppp
			}
		}
	}

	//端口
	if port, ok := cluster["port"].(int64); ok {
		module.config.Port = int(port)
	}
	if port, ok := cluster["port"].(int); ok {
		module.config.Port = int(port)
	}
	if port, ok := cluster["port"].(float64); ok {
		module.config.Port = int(port)
	}

	// 集群安全key
	if key, ok := cluster["key"].(string); ok {
		module.config.Secret = key
	}
	if key, ok := cluster["secret"].(string); ok {
		module.config.Secret = key
	}
	if cert, ok := cluster["cert"].(string); ok {
		module.config.Secret = cert
	}

	// 要加入的节点
	if join, ok := cluster["join"].([]string); ok {
		module.config.Join = join
	}
	if peers, ok := cluster["peers"].([]string); ok {
		module.config.Join = peers
	}

	// 元数据
	if meta, ok := cluster["meta"].(Map); ok {
		module.config.Meta = meta
	}
}

// register 模块注册中心
func (module *clusterModule) register(key string, value Any, override bool) {
	switch val := value.(type) {
	case ClusterDriver:
		module.Driver(key, val, override)
	}
}

// initialize 初始化集群模块
func (module *clusterModule) initialize() {
	driver, ok := module.drivers[module.config.Driver]
	if ok == false {
		panic("Invalid cluster driver: " + module.config.Driver)
	}

	connect, err := driver.Connect(module.config)
	if err != nil {
		panic("Failed to connect to cluster: " + err.Error())
	}

	// 打开连接
	err = connect.Open()
	if err != nil {
		panic("Failed to open cluster connect: " + err.Error())
	}

	//保存连接
	module.connect = connect

	// fmt.Println("cluster initialized")
}

// launch 集群模块launch暂时没有用
func (module *clusterModule) launch() {
	// fmt.Println("cluster launched")
}

// terminate 关闭集群模块
func (module *clusterModule) terminate() {
	if module.connect != nil {
		module.connect.Close()
	}
}

// Driver 注册驱动
func (module *clusterModule) Driver(name string, driver ClusterDriver, override bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	if driver == nil {
		panic("Invalid cluster driver: " + name)
	}

	if override {
		module.drivers[name] = driver
	} else {
		if module.drivers[name] == nil {
			module.drivers[name] = driver
		}
	}
}

func (module *clusterModule) Locate(key string) bool {
	if module.connect == nil {
		return false
	}
	return module.connect.Locate(key)
}
func (module *clusterModule) Read(key string) ([]byte, error) {
	if module.connect == nil {
		return nil, errInvalidClusterConnection
	}
	return module.connect.Read(key)
}
func (module *clusterModule) Write(key string, val []byte) error {
	if module.connect == nil {
		return errInvalidClusterConnection
	}
	return module.connect.Write(key, val)
}
func (module *clusterModule) Delete(key string) error {
	if module.connect == nil {
		return errInvalidClusterConnection
	}
	return module.connect.Delete(key)
}
func (module *clusterModule) Peers() []ClusterPeer {
	if module.connect == nil {
		return []ClusterPeer{}
	}
	return module.connect.Peers()
}

// //获取服务列表
// func (module *clusterModule) Services() []Map {
// 	if module.connect == nil {
// 		return []Map{}
// 	}

// 	data, err := module.connect.Fetch("service-")
// 	if err != nil {
// 		return []Map{}
// 	}

// 	sss := make([]Map, 0)
// 	for _, val := range data {
// 		var srv Map
// 		err := mCodec.JsonDecode(val, &srv)
// 		if err == nil {
// 			sss = append(sss, srv)
// 		}
// 	}

// 	return sss

// }

// func ReadCluster(key string) ([]byte, error) {
// 	return mCluster.Read(key)
// }
// func WriteCluster(key string, val []byte) error {
// 	return mCluster.Write(key, val)
// }

func Peers() []ClusterPeer {
	return mCluster.Peers()
}

// func ClusterServices() []Map {
// 	return mCluster.Services()
// }
