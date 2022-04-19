package chef

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/chefsgo/base"
)

//待处理 json 格式的输出
//待处理 format 支持更多字段

var (
	mLog = &logModule{
		config: LogConfig{
			Driver: DEFAULT, Level: LogDebug,
			Json: false, Sync: false, Pool: 1000,
			Format: "%time% [%level%] %body%",
		},
		drivers: map[string]LogDriver{},
	}
)

const (
	//LogLevel 日志级别，从小到大，数字越小越严重
	LogFatal   LogLevel = iota //错误级别，产生了严重错误，程序将退出
	LogPanic                   //恐慌级别，产生了恐慌，会调用panic
	LogWarning                 //警告级别，一般是记录调用错误
	LogNotice                  //注意级别，需要特别留意的信息
	LogInfo                    //普通级别，普通信息
	LogTrace                   //追踪级别，主要是请求日志，调用追踪等
	LogDebug                   //调试级别，开发时输出调试时用，生产环境不建议
)

var (
	logLevels = map[LogLevel]string{
		LogFatal:   "ERROR",
		LogPanic:   "PANIC",
		LogWarning: "WARNING",
		LogNotice:  "NOTICE",
		LogInfo:    "INFO",
		LogTrace:   "TRACE",
		LogDebug:   "DEBUG",
	}
)

type (
	// LogLevel 日志级别，从小到大，数字越小越严重
	LogLevel = int

	// LogConfig 日志模块配置
	LogConfig struct {
		// Driver 日志驱动，默认为 default
		Driver string

		// Level 输出的日志级别
		// fatal, panic, warning, notice, info, trace, debug
		Level LogLevel

		// Json 是否开启json输出模式
		// 开启后，所有日志 body 都会被包装成json格式输出
		Json bool

		// Sync 是否开启同步输出，默认为false，表示异步输出
		// 注意：如果开启同步输出，有可能影响程序性能
		Sync bool

		// Pool 异步缓冲池大小
		Pool int

		// Format 日志输出格式，默认格式为 %time% [%level%] %body%
		// 可选参数，参数使用 %% 包裹，如 %time%
		// time		格式化后的时间，如：2006-01-02 15:03:04.000
		// unix		unix时间戳，如：1650271473
		// level	日志级别，如：TRACE
		// body		日志内容
		Format string `toml:"format"`

		// Setting 是为不同驱动准备的自定义参数
		// 具体参数表，请参考各不同的驱动
		Setting Map `toml:"setting"`
	}

	// LogDriver log驱动
	LogDriver interface {
		// 连接到驱动
		Connect(config LogConfig) (LogConnect, error)
	}
	// LogConnect 日志连接
	LogConnect interface {
		// Open 打开连接
		Open() error

		// Close 关闭结束
		Close() error

		// Write 写入日志
		Write(Log) error

		// Flush 冲马桶
		Flush()
	}

	Log struct {
		format string   `json:"-"`
		Time   int64    `json:"time"`
		Level  LogLevel `json:"level"`
		Body   string   `json:"body"`
	}

	//logModule 日志模块定义
	logModule struct {
		//mutex 锁
		mutex sync.Mutex

		//config 日志配置
		config LogConfig

		//drivers 驱动注册表
		drivers map[string]LogDriver

		// connect 日志连接
		connect LogConnect

		waiter sync.WaitGroup

		// logger 日志发送管道
		logger chan Log

		// signal 信号管道，用于flush缓存区，或是结束循环
		// false 表示flush缓存区
		// true 表示结束关闭循环
		signal chan bool
	}
)

// configure 为log模块加载配置
func (module *logModule) configure(data Map) {
	if log, ok := data["log"].(Map); ok {
		//设置驱动
		if driver, ok := log["driver"].(string); ok {
			module.config.Driver = driver
		}
		//设置级别
		if level, ok := log["level"].(string); ok {
			for l, s := range logLevels {
				if strings.ToUpper(level) == s {
					module.config.Level = l
				}
			}
		}
		//是否json
		if json, ok := log["json"].(bool); ok {
			module.config.Json = json
		}
		//设置是否同步
		if sync, ok := log["sync"].(bool); ok {
			module.config.Sync = sync
		}
		// 设置输出格式
		if format, ok := log["format"].(string); ok {
			module.config.Format = format
		}

		// 设置缓存池大小
		if pool, ok := log["pool"].(int64); ok && pool > 0 {
			module.config.Pool = int(pool)
		}
		if pool, ok := log["pool"].(int); ok && pool > 0 {
			module.config.Pool = pool
		}
	}

	// fmt.Println("log configured", module.config)
}

// register 为log模块注册内容
func (module *logModule) register(key string, val Any, override bool) {
	switch obj := val.(type) {
	case LogDriver:
		module.Driver(key, obj, override)
	}
	// fmt.Println("log registered", key)
}

// initialize 初始化日志模块
func (module *logModule) initialize() {
	driver, ok := module.drivers[module.config.Driver]
	if ok == false {
		panic("Invalid log driver: " + module.config.Driver)
	}

	// 建立连接
	connect, err := driver.Connect(module.config)
	if err != nil {
		panic("Failed to connect to log: " + err.Error())
	}

	// 打开连接
	err = connect.Open()
	if err != nil {
		panic("Failed to open log connect: " + err.Error())
	}

	// 保存连接，设置管道大小
	module.connect = connect
	module.logger = make(chan Log, 100)
	module.signal = make(chan bool, 1)

	// 如果非同步模式，就开始异步循环
	if false == module.config.Sync {
		go module.eventLoop()
	}

	fmt.Println("log initialized", module.config.Sync)
}

// launch 日志模块launch暂时没有用
func (module *logModule) launch() {
	// fmt.Println("log launched")
}

// terminate 关闭日志模块
func (module *logModule) terminate() {
	if module.connect != nil {
		module.Flush()
		module.connect.Close()
	}
}

//Driver 为log模块注册驱动
func (module *logModule) Driver(name string, driver LogDriver, override bool) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	if driver == nil {
		panic("[log]驱动不可为空")
	}

	if override {
		module.drivers[name] = driver
	} else {
		if module.drivers[name] == nil {
			module.drivers[name] = driver
		}
	}
}

//Log format
func (log *Log) Format() string {
	message := log.format

	// message := strings.Replace(format, "%flag%", log.Flag, -1)
	message = strings.Replace(message, "%nano%", strconv.FormatInt(log.Time, 10), -1)
	message = strings.Replace(message, "%time%", time.Unix(0, log.Time).Format("2006-01-02 15:04:05.000"), -1)
	message = strings.Replace(message, "%level%", logLevels[log.Level], -1)
	// message = strings.Replace(message, "%file%", log.File, -1)
	// message = strings.Replace(message, "%line%", strconv.Itoa(log.Line), -1)
	// message = strings.Replace(message, "%func%", log.Func, -1)
	message = strings.Replace(message, "%body%", log.Body, -1)

	return message
}

// flush 调用连接在write
func (module *logModule) write(msg Log) error {
	if module.connect == nil {
		return errors.New("[日志]无效连接")
	}

	//格式传过去
	msg.format = module.config.Format

	return module.connect.Write(msg)
}

//flush 真flush
func (module *logModule) flush() {
	if false == module.config.Sync {
		for {
			if len(module.logger) > 0 {
				log := <-module.logger
				module.write(log)
				module.waiter.Done()
			} else {
				break
			}
		}
	}
	module.connect.Flush()
}

// Write 写入日志，对外的，需要处理逻辑
func (module *logModule) Write(msg Log) error {
	if module.config.Level < msg.Level {
		return nil
	}
	if msg.Time <= 0 {
		msg.Time = time.Now().UnixNano()
	}

	if module.config.Sync {
		// 同步模式下 直接写消息
		return module.write(msg)
	} else {
		//异步模式写入管道
		module.waiter.Add(1)
		module.logger <- msg
		return nil
	}
}

func (module *logModule) Flush() {
	if false == module.config.Sync {
		module.signal <- false
		module.waiter.Wait()
	} else {
		module.flush()
	}
}

// Logging 对外按日志级写日志的方法
func (module *logModule) Logging(level LogLevel, body string) error {
	msg := Log{Time: time.Now().UnixNano(), Level: level, Body: body}
	return module.Write(msg)
}

// asyncLoop 异步循环
func (module *logModule) eventLoop() {
	for {
		select {
		case log := <-module.logger:
			module.write(log)
			module.waiter.Done()
		case signal := <-module.signal:
			if signal {
				module.flush()
			} else {
				module.flush()
			}
		}
	}
}

//string 把对象转成字串
func (module *logModule) string(args ...Any) string {
	vs := []string{}
	for _, v := range args {
		s := fmt.Sprintf("%v", v)
		vs = append(vs, s)
	}
	return strings.Join(vs, " ")
}
func (module *logModule) parse(args ...Any) string {
	ls := len(args)
	if ls == 0 {
		return ""
	}
	if ls == 1 {
		return module.string(args...)
	} else {
		if format, ok := args[0].(string); ok {
			ccc := strings.Count(format, "%") - strings.Count(format, "%%")
			if ccc > 0 && ccc == (len(args)-1) {
				return fmt.Sprintf(format, args[1:]...)
			}
		}
		return module.string(args...)
	}
}

// output 是为了直接输出到控制台，不管是否启用控制台
func (module *logModule) output(args ...Any) {
	body := module.parse(args...)
	log.Println(body)
}

//调试
func (module *logModule) Debug(args ...Any) {
	if module.connect == nil {
		module.output(args...)
	} else {
		module.Logging(LogDebug, module.parse(args...))
	}
}
func (module *logModule) Trace(args ...Any) {
	if module.connect == nil {
		module.output(args...)
	} else {
		module.Logging(LogTrace, module.parse(args...))
	}
}
func (module *logModule) Info(args ...Any) {
	if module.connect == nil {
		module.output(args...)
	} else {
		module.Logging(LogInfo, module.parse(args...))
	}
}
func (module *logModule) Notice(args ...Any) {
	if module.connect == nil {
		module.output(args...)
	} else {
		module.Logging(LogNotice, module.parse(args...))
	}
}
func (module *logModule) Warning(args ...Any) {
	if module.connect == nil {
		module.output(args...)
	} else {
		module.Logging(LogWarning, module.parse(args...))
	}
}
func (module *logModule) Panic(args ...Any) {
	if module.connect == nil {
		module.output(args...)
	} else {
		module.Logging(LogPanic, module.parse(args...))
	}
	panic(module.parse(args...))
}
func (module *logModule) Fatal(args ...Any) {
	if module.connect == nil {
		module.output(args...)
	} else {
		module.Logging(LogFatal, module.parse(args...))
	}
	//待处理，发送退出信号
}

//定义列表
func LogLevels() map[LogLevel]string {
	return logLevels
}

// //语法糖
func Debug(args ...Any) {
	mLog.Debug(args...)
}
func Trace(args ...Any) {
	mLog.Trace(args...)
}
func Info(args ...Any) {
	mLog.Info(args...)
}
func Notice(args ...Any) {
	mLog.Notice(args...)
}
func Warning(args ...Any) {
	mLog.Warning(args...)
}
func Panic(args ...Any) {
	mLog.Panic(args...)
}
func Fatal(args ...Any) {
	mLog.Fatal(args...)
}
