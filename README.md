# chef
Chefs.go chef


<!-- 有些方法是需要定义成动态可替换的，比如，bodyParser -->


<!--

    各模块的configure没有处理setting
    各模块的 headth 还是要的， 像 data，就没法在框架内完成统计，因为不知道db是什么Close的

    所有模块间的调用，考虑使用 委托， 直接整到 chef 中， 做中转
    这样使所有模块独立， 不相互依赖
    

    cache 配置加 codec ，来指定加解密后段， 所以,Read的时候，要加 引用参数，用来 Unmar....

    默认的  cache, session, mutex 驱动，都要更新，因为，没有做自动过期处理

    log 模块的 管道 flush，结束，有问题，还没调

    queue redis 驱动， 多协程队列，关闭退出时，会有BUG，可能不会等待任务全部执行完成

    method调用的时候，必须newContext，只共享 meta 元数据，
    要不然连续调用的时候,name,config,valud,args会全部串线被修改
    method的调用，需要优化
    要不然就是直接拿Meta做为父类， 再考虑，
    拿meta做父类，或是，集成meta的一个父类， 或是 method调用的Context得用一个子类，不能直接用现在的context
    事实意义就是，meta是所有context的父类，

    event, queue 留着自定义Queue, Event 的名字功能，放到后续升级中
    event, queue Weight为-1的，不分布的，应该默认不注册， 除非指定连接
    event queue， 都支持，notice吧， 如果定义了 notice，那在publish的时候，要做参数解析

    全模块error替换为Res，每个模块定义自己的res列表，这样返回或是输出log的时候，就可以按自己定义的 语言，输出文案了


    chef 不公共end方法， 改成在 注册 模块 module 的时候，返回一个接口。  
    这个接口，可以在模块里访问，一些chef内容的方法，比如，end之类的，不直接能被包外调用的方法

    register去掉 override ，因为模块的 builtin，加载包的时候，就已经完成了
    就算是框架层面的builtin，也应该先引入， 可否被替换， 由模块自己决定

    event StartTrigger中，如果发一个 event，to redis
    第一个节点自己，会收不到，但是只要有其它已启动的节点，自己就能收到
    延迟100毫秒， 就可以收到， 说明 StartTrigger 被启动的同时
    event,redis还没初始化完成，因为监听是独立协池，所以
    解决方法：所有异步launch的模块，使用一个  WaitGroup 来同步等待 完成初始化

    event redis 驱动，暂时还没有好的分组想法或方案，待处理

    所有模块的configure 要检查，不直接从顶层map解析数据


    chef 各模块的委托方法

    http.bodyParser 要处理，老方法太垃圾了
    或者允许，被替换成自己想要的中间件


    http.Ctx.IPs()
    // X-Forwarded-For: proxy1, 127.0.0.1, proxy3
    c.IPs() // ["proxy1", "127.0.0.1", "proxy3"]

    ctx 方法，更多 动态参数处理，如 ctx.File 一样
    http ctx.Protocol() 方法
    http ctx Host Path 这些，函数化， 避免被外部修改。
    ctx.Uri(),  ctx.Host(),  ctx.Domain(),  ctx.Path() 等
    ctx.BodyParser, ctx.QueryParser 等解析方法，
    ctx.Routing，直接转向到另一个路由上去，响应
    ctx.File 考虑 compress 参数
    http 内置 gzip 啥的

    现成的middlewares，比如， limiter 啥的， 进一步简化开发工作
    比如，请求http,event,queue的请求log 中件间什么的
    gofiber 可参考
    cross 可以做成 中件间的方式提供
    cookies 加密，也可以走中间件？
    就是尽量，把所有功能都以中间件的方式提供，也更方便去替换


-->