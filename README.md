# Cache

### 实现的效果

1. 支持多维度自定义查询逻辑, 比如通过channelCache.Get("true", ">10"), 可以通过定义自定义查询逻辑来缓存"可用的且id大于10的channel"
2. 任何情况下对应指定维度的查询，最多只有一次撞库(1级缓存失效的情况下)
3. 1级缓存失效，如果没开启2级缓存，除了拿到撞库权限的线程以外，其他都返回空，如果开启2级缓存且没过期的话，则落在2级缓存上
4. 对于不存在的数据依旧可以通过缓存拦截

### 使用方法

1. `func(...string) string` 按这个规则定义数据查询逻辑，通过 cache.Get(...string) 传入的参数都会传进去。缓存失效就会通过这个方法更新数据。里面可以定义自己逻辑，可以用mysql，可以用mogodb等等，根据具体业务可以灵活定义

2. `cache.NewCache(&cache.ClientGoCache{}, keyPrefix, funcReadData, expireTime, use2Cache)`，
  - 创建cache实例。注册你要使用的缓存服务。目前实现了本地(GoCache)和Redis，扩展很方便。
  - keyPrefix 定义键名前缀防止键值冲突
    - 1级缓存键名 `{keyPrefix}_{f1}_{f2}.._1`
    - 2级缓存键名 `{keyPrefix}_{f1}_{f2}.._2`
    - 全局锁键名 `{keyPrefix}_{f1}_{f2}.._lock`
  - funcReadData 自定义数据查询逻辑
  - expireTime 1级缓存过期时间，2级缓存过期时间 = 1级缓存过期时间 + 5分钟
  - use2Cache 是否使用2级缓存

3. 上面这些都做好之后，你要做的就只是使用 `cache.Get(...string)` 获取你要的数据就可以了

```
import "subscription_api_console_backend/utils/cache"

func main() {

  // 实现从数据源读取数据 支持多字段
  channelReadData := func(fs ...string) string {
    channelId, _ := strconv.Atoi(fs[0])
    // 从数据库读取channel并格式化成字符串
    data := db.get(channelId)
    return data
  }
  
  cacheClient := &cache.ClientGoCache{} // 使用本地缓存
  //cacheClient := &cache.ClientRedis{} // 使用 Redis 缓存
  expireTime := time.Minute * 5 // 缓存过期时间
  keyPrefix := "cChannel" // 换成键名前缀
  use2Cache := true // 是否使用2级缓存

  channelCache := cache.NewCache(cacheClient, keyPrefix, funcReadData, expireTime, use2Cache)

  channelCache.Get(1) // 读取 id = 1 的 channel
}

```

### 压测试情况

使用 Redis 缓存

```
BenchmarkCache1B-4                 30000             53967 ns/op             224 B/op          9 allocs/op
BenchmarkMid1B-4                   20000             64072 ns/op             240 B/op         12 allocs/op
BenchmarkCache1KB-4                20000             51530 ns/op             580 B/op          9 allocs/op
BenchmarkMid1KB-4                  30000             52481 ns/op            1280 B/op         12 allocs/op
BenchmarkCache500KB-4               5000            244467 ns/op          210840 B/op          9 allocs/op
BenchmarkMid500KB-4                 3000            424048 ns/op          508711 B/op         12 allocs/op
```

使用本地缓存

```
BenchmarkCache1B-4               5000000               271 ns/op              35 B/op          4 allocs/op
BenchmarkMid1B-4                 5000000               298 ns/op              48 B/op          5 allocs/op
BenchmarkCache1KB-4              5000000               286 ns/op              52 B/op          4 allocs/op
BenchmarkMid1KB-4                5000000               314 ns/op              64 B/op          5 allocs/op
BenchmarkCache500KB-4            5000000               287 ns/op              52 B/op          4 allocs/op
BenchmarkMid500KB-4              5000000               314 ns/op              64 B/op          5 allocs/op
```

> 以上测试是分别在存储数据在 1B/1KB/500KB 情况下的性能表现，个人建议缓存如果不需要持久化切不需要分布式的话 用本地缓存就可以了