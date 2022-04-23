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


-->