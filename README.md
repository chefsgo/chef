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

    要不然就是直接拿Meta做为父类， 再考虑


-->