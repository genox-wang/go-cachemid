package gocachemid

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkCache1B(b *testing.B) {
	benchmarkCacheGet(b, 1, true)
}

func BenchmarkMid1B(b *testing.B) {
	benchmarkCacheGet(b, 1, false)
}

func BenchmarkCache1KB(b *testing.B) {
	benchmarkCacheGet(b, 1000, true)
}

func BenchmarkMid1KB(b *testing.B) {
	benchmarkCacheGet(b, 1000, false)
}

func BenchmarkCache500KB(b *testing.B) {
	benchmarkCacheGet(b, 500000, true)
}
func BenchmarkMid500KB(b *testing.B) {
	benchmarkCacheGet(b, 500000, false)
}

func benchmarkCacheGet(b *testing.B, size int64, useRedis bool) {
	//做一些初始化的工作,例如读取文件数据,数据库连接之类的,
	//这样这些时间不影响我们测试函数本身的性能
	funcReadData := func(fs ...string) (string, error, bool) {
		// time.Sleep(time.Second * 1)
		dst := make([]byte, size)
		return string(dst), nil, true
	}

	expireTime := time.Second * 1
	keyPrefix := fmt.Sprintf("bt_%d_", size)

	testCache := &Cache{
		CacheClient:      SelectedCache(),
		KeyPrefix:        keyPrefix,
		FuncReadData:     funcReadData,
		ExpireTime:       expireTime,
		Cache2Enabled:    true,
		Cache2ExpireTime: DefaultCache2ExpirePadding,
	}

	testCache.Get("1")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if useRedis {
			testCache.CacheClient.Get(testCache.GetCacheLayerKey(1, "1"))
		} else {
			testCache.Get("1")
		}
	}

	// b.SetParallelism(5)
	// b.RunParallel(func(pb *testing.PB) {
	// 	for pb.Next() {
	// 		testCache.Get("1")
	// 	}
	// })
}

// 测试代码
// const m1 = 0x5555555555555555
// const m2 = 0x3333333333333333
// const m4 = 0x0f0f0f0f0f0f0f0f
// const h01 = 0x0101010101010101

// func BenchmarkPopcnt(b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		x := i
// 		x -= (x >> 1) & m1
// 		x = (x & m2) + ((x >> 2) & m2)
// 		x = (x + (x >> 4)) & m4
// 		_ = (x * h01) >> 56
// 	}
// }
