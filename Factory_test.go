package gocachemid

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

type AssertModel struct {
	Expected interface{}
	Actual   interface{}
	Message  string
}

type CacheTestAsserts struct {
	CrossCnt   AssertModel
	SuccessCnt AssertModel
	EmptyCnt   AssertModel
}

func SelectedCache() ClientBase {
	// return &ClientRedis{}
	return &ClientGoCache{
		client: cache.New(DefaultCacheExpire, time.Minute*10),
	}
}

func TestLock(t *testing.T) {
	c := &Cache{
		CacheClient:      SelectedCache(),
		KeyPrefix:        "lock:test",
		ExpireTime:       time.Second * 10, // 过期时间
		Cache2Enabled:    true,
		Cache2ExpireTime: time.Second * 10,
		FuncReadData:     nil,
	}

	var success int32
	var fail int32

	testKey := "test"

	runnerNum := 5000

	runASync(func(runTimes *int32) {
		if c.Lock(testKey, time.Second*10) {
			atomic.AddInt32(&success, 1)
		} else {
			atomic.AddInt32(&fail, 1)
		}
		atomic.AddInt32(runTimes, 1)
	}, runnerNum)

	assert.Equal(t, success, int32(1), "同一键名只能被锁一次")

	c.UnLock(testKey)

	assert.Equal(t, c.Lock(testKey, time.Second*10), true, "解锁后应该可以再上锁")
}

func TestCache(t *testing.T) {
	var crossCounter int32

	funcReadData := func(fs ...string) (string, bool, error) {
		time.Sleep(time.Second * 1)
		atomic.AddInt32(&crossCounter, 1)
		return fs[0], true, nil
	}

	expireTime := time.Second * 2
	keyPrefix := "cacheTest_"

	testCache := &Cache{
		CacheClient:      SelectedCache(),
		KeyPrefix:        keyPrefix,
		FuncReadData:     funcReadData,
		ExpireTime:       expireTime,
		Cache2Enabled:    true,
		Cache2ExpireTime: DefaultCache2ExpirePadding,
	}

	runnerNum := 50

	// 还没有生成缓存
	doTestCache(t, testCache, runnerNum, &crossCounter, CacheTestAsserts{
		CrossCnt:   AssertModel{Expected: int32(1), Message: "并发只允许一次穿透"},
		SuccessCnt: AssertModel{Expected: int32(1), Message: "二级缓存生成前其他请求都应该失败"},
		EmptyCnt:   AssertModel{Expected: int32(runnerNum - 1), Message: "二级缓存生成前其他请求都应该返回空"},
	}, "1")

	// 生成缓存且没过期
	doTestCache(t, testCache, runnerNum, &crossCounter, CacheTestAsserts{
		CrossCnt:   AssertModel{Expected: int32(0), Message: "缓存在不应该穿透缓存"},
		SuccessCnt: AssertModel{Expected: int32(runnerNum), Message: "缓存存在应该全部打在缓存上"},
		EmptyCnt:   AssertModel{Expected: int32(0), Message: "缓存存在应该不存在空返回"},
	}, "1")

	// 等待 缓存过期
	time.Sleep(expireTime)

	// 缓存过期 二级缓存没过期，且使用二级缓存
	doTestCache(t, testCache, runnerNum, &crossCounter, CacheTestAsserts{
		CrossCnt:   AssertModel{Expected: int32(1), Message: "只允许一次缓存穿透"},
		SuccessCnt: AssertModel{Expected: int32(runnerNum), Message: "开启二级缓存应该是0失败"},
		EmptyCnt:   AssertModel{Expected: int32(0), Message: "开启二级缓存应该是0空返回"},
	}, "1")

	// 关闭 2级缓存
	testCache.SetCache2Enabled(false)
	// 等待 缓存过期
	time.Sleep(expireTime)

	// 缓存过期 二级缓存没过期，且不使用二级缓存
	doTestCache(t, testCache, runnerNum, &crossCounter, CacheTestAsserts{
		CrossCnt:   AssertModel{Expected: int32(1), Message: "只允许一次缓存穿透"},
		SuccessCnt: AssertModel{Expected: int32(1), Message: "不开启二级缓存只有穿透的能成功"},
		EmptyCnt:   AssertModel{Expected: int32(runnerNum - 1), Message: "不开启二级缓存应该剩余的全返回空"},
	}, "1")
}

func TestCacheNoExistData(t *testing.T) {
	var crossCounter int32

	funcReadData := func(fs ...string) (string, bool, error) {
		atomic.AddInt32(&crossCounter, 1)
		return "", true, nil
	}

	expireTime := time.Second * 4
	keyPrefix := "cacheNoExistTest_"

	testCache := &Cache{
		CacheClient:      SelectedCache(),
		KeyPrefix:        keyPrefix,
		FuncReadData:     funcReadData,
		ExpireTime:       expireTime,
		Cache2Enabled:    true,
		Cache2ExpireTime: DefaultCache2ExpirePadding,
	}

	runnerNum := 50

	testCache.Get("1")

	doTestCache(t, testCache, runnerNum, &crossCounter, CacheTestAsserts{
		CrossCnt:   AssertModel{Expected: int32(0), Message: "缓存存在不应该有穿透"},
		SuccessCnt: AssertModel{Expected: int32(runnerNum), Message: "走缓存应该都成功"},
		EmptyCnt:   AssertModel{Expected: int32(runnerNum), Message: "空数据全返回空"},
	}, "1")
}

// func TestCacheNoNeedToCache(t *testing.T) {
// 	var crossCounter int32

// 	funcReadData := func(fs ...string) (string, error, bool) {
// 		atomic.AddInt32(&crossCounter, 1)
// 		return "", nil, false
// 	}

// 	expireTime := time.Second * 4
// 	keyPrefix := "cacheNoNeedToCacheTest_"

// 	testCache := &Cache{
// 		CacheClient:      SelectedCache(),
// 		KeyPrefix:        keyPrefix,
// 		FuncReadData:     funcReadData,
// 		ExpireTime:       expireTime,
// 		Cache2Enabled:    true,
// 		Cache2ExpireTime: DefaultCache2ExpirePadding,
// 	}

// 	runnerNum := 50

// 	testCache.Get("1")

// 	doTestCache(t, testCache, runnerNum, &crossCounter, CacheTestAsserts{
// 		CrossCnt:   AssertModel{Expected: int32(runnerNum), Message: "应该不走缓存"},
// 		SuccessCnt: AssertModel{Expected: int32(runnerNum), Message: "走缓存应该都成功"},
// 		EmptyCnt:   AssertModel{Expected: int32(runnerNum), Message: "空数据全返回空"},
// 	}, "1")
// }

func doTestCache(t *testing.T, c *Cache, runnerNum int, crossCounter *int32, asserts CacheTestAsserts, fields ...string) {
	*crossCounter = 0
	// crossCounter.ToZero()
	success, empty := AsyncRequestCache(c, runnerNum, fields...)

	assert.Equal(t, asserts.CrossCnt.Expected.(int32), *crossCounter, asserts.CrossCnt.Message)
	assert.Equal(t, asserts.SuccessCnt.Expected.(int32), success, asserts.SuccessCnt.Message)
	assert.Equal(t, asserts.EmptyCnt.Expected.(int32), empty, asserts.EmptyCnt.Message)
}

func AsyncRequestCache(c *Cache, runTimes int, fields ...string) (success, empty int32) {

	runASync(func(runnedCounter *int32) {
		v, _, err := c.Get(fields...)
		if err == nil {
			atomic.AddInt32(&success, 1)
		}
		if v == "" {
			atomic.AddInt32(&empty, 1)
		}
		atomic.AddInt32(runnedCounter, 1)
	}, runTimes)

	return success, empty
}

func runASync(action func(*int32), times int) {
	var runTimes int32
	for i := 0; i < times; i++ {
		go action(&runTimes)
	}
	for {
		if runTimes == int32(times) {
			break
		}
	}
}
