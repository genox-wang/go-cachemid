package gocachemid

import (
	"gosync"
	"testing"
	"time"

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
	return &ClientGoCache{}
}

func TestLock(t *testing.T) {
	c := NewCache(SelectedCache(), "", nil, 0, true)

	success := gosync.NewCounter(0)
	fail := gosync.NewCounter(0)

	testKey := "key"

	runnerNum := 5000

	runASync(func(runnedCounter *gosync.Counter) {
		if c.Lock(testKey, time.Second*10) {
			success.Incr(1)
		} else {
			fail.Incr(1)
		}
		runnedCounter.Incr(1)
	}, runnerNum)

	assert.Equal(t, success.Val(), int64(1), "同一键名只能被锁一次")

	c.UnLock(testKey)

	assert.Equal(t, c.Lock(testKey, time.Second*10), true, "解锁后应该可以再上锁")
}

func TestCache(t *testing.T) {
	crossCounter := gosync.NewCounter(0)

	funcReadData := func(fs ...string) string {
		time.Sleep(time.Second * 1)
		crossCounter.Incr(1)
		return fs[0]
	}

	expireTime := time.Second * 2
	keyPrefix := "cacheTest_"

	testCache := NewCache(SelectedCache(), keyPrefix, funcReadData, expireTime, true)

	runnerNum := 50

	// 还没有生成缓存
	doTestCache(t, testCache, runnerNum, &crossCounter, CacheTestAsserts{
		CrossCnt:   AssertModel{Expected: 1, Message: "并发只允许一次穿透"},
		SuccessCnt: AssertModel{Expected: 1, Message: "二级缓存生成前其他请求都应该失败"},
		EmptyCnt:   AssertModel{Expected: runnerNum - 1, Message: "二级缓存生成前其他请求都应该返回空"},
	}, "1")

	// 生成缓存且没过期
	doTestCache(t, testCache, runnerNum, &crossCounter, CacheTestAsserts{
		CrossCnt:   AssertModel{Expected: 0, Message: "缓存在不应该穿透缓存"},
		SuccessCnt: AssertModel{Expected: runnerNum, Message: "缓存存在应该全部打在缓存上"},
		EmptyCnt:   AssertModel{Expected: 0, Message: "缓存存在应该不存在空返回"},
	}, "1")

	// 等待 缓存过期
	time.Sleep(expireTime)

	// 缓存过期 二级缓存没过期，且使用二级缓存
	doTestCache(t, testCache, runnerNum, &crossCounter, CacheTestAsserts{
		CrossCnt:   AssertModel{Expected: 1, Message: "只允许一次缓存穿透"},
		SuccessCnt: AssertModel{Expected: runnerNum, Message: "开启二级缓存应该是0失败"},
		EmptyCnt:   AssertModel{Expected: 0, Message: "开启二级缓存应该是0空返回"},
	}, "1")

	// 关闭 2级缓存
	testCache.SetCache2Enabled(false)
	// 等待 缓存过期
	time.Sleep(expireTime)

	// 缓存过期 二级缓存没过期，且不使用二级缓存
	doTestCache(t, testCache, runnerNum, &crossCounter, CacheTestAsserts{
		CrossCnt:   AssertModel{Expected: 1, Message: "只允许一次缓存穿透"},
		SuccessCnt: AssertModel{Expected: 1, Message: "不开启二级缓存只有穿透的能成功"},
		EmptyCnt:   AssertModel{Expected: runnerNum - 1, Message: "不开启二级缓存应该剩余的全返回空"},
	}, "1")
}

func TestCacheNoExistData(t *testing.T) {
	crossCounter := gosync.NewCounter(0)

	funcReadData := func(fs ...string) string {
		crossCounter.Incr(1)
		// 模拟读取不到数据
		return ""
	}

	expireTime := time.Second * 4
	keyPrefix := "cacheNoExistTest_"

	testCache := NewCache(SelectedCache(), keyPrefix, funcReadData, expireTime, true)

	runnerNum := 50

	testCache.Get("1")

	doTestCache(t, testCache, runnerNum, &crossCounter, CacheTestAsserts{
		CrossCnt:   AssertModel{Expected: 0, Message: "缓存存在不应该有穿透"},
		SuccessCnt: AssertModel{Expected: runnerNum, Message: "走缓存应该都成功"},
		EmptyCnt:   AssertModel{Expected: runnerNum, Message: "空数据全返回空"},
	}, "1")
}

func doTestCache(t *testing.T, c *Cache, runnerNum int, crossCounter *gosync.Counter, asserts CacheTestAsserts, fields ...string) {
	crossCounter.ToZero()
	success, empty := AsyncRequestCache(c, runnerNum, fields...)

	assert.Equal(t, int64(asserts.CrossCnt.Expected.(int)), crossCounter.Val(), asserts.CrossCnt.Message)
	assert.Equal(t, int64(asserts.SuccessCnt.Expected.(int)), success, asserts.SuccessCnt.Message)
	assert.Equal(t, int64(asserts.EmptyCnt.Expected.(int)), empty, asserts.EmptyCnt.Message)
}

func AsyncRequestCache(c *Cache, runTimes int, fields ...string) (success, empty int64) {
	successCounter := gosync.NewCounter(0)
	emptyCounter := gosync.NewCounter(0)

	runASync(func(runnedCounter *gosync.Counter) {
		v, err := c.Get(fields...)
		if err == nil {
			successCounter.Incr(1)
		}
		if v == "" {
			emptyCounter.Incr(1)
		}
		runnedCounter.Incr(1)
	}, runTimes)

	success = successCounter.Val()
	empty = emptyCounter.Val()
	return success, empty
}

func runASync(action func(*gosync.Counter), times int) {
	runnedCounter := gosync.NewCounter(0)
	for i := 0; i < times; i++ {
		go action(&runnedCounter)
	}
	for {
		if runnedCounter.Val() == int64(times) {
			break
		}
	}
}
