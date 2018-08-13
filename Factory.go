package gocachemid

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// LockSuffix 分布式锁后缀
	LockSuffix = "_lock"
	// LockExpire 锁过期时间
	LockExpire = time.Second * 10
	// DefaultCacheExpire 默认缓存过期时间
	DefaultCacheExpire = time.Second * 10
	// DefaultCache2ExpirePadding 默认二级缓存间隔
	DefaultCache2ExpirePadding = time.Second * 10
)

// Cache 自定义缓存类，支持2级缓存
// 定时释放冷数据
// 防雪崩
// 数据源数据更新操作完成，调用Del释放缓存，来优化缓存数据一致性
type Cache struct {
	CacheClient   ClientBase
	KeyPrefix     string
	ExpireTime    time.Duration
	Cache2Enabled bool
	FuncReadData  func(...string) (string, error)
}

// Lock 尝试获取指定键名的分布式锁
// key 键名
// expireTime 过期时间, 0表示不过期
// 加锁成功返回 true
func (c *Cache) Lock(key string, expireTime time.Duration) bool {
	return c.CacheClient.Add(key, "1", expireTime)
}

// UnLock 对指定键名解锁
func (c *Cache) UnLock(key string) {
	c.CacheClient.Del(key)
}

// Del 可选维度释放缓存 一般在数据源数据更新操作后调用释放旧数据
func (c *Cache) Del(fields ...string) error {
	return c.CacheClient.Del(c.GetCacheLayerKey(1, fields...))
}

// DelWithPrefix 根据前缀删除缓存
func (c *Cache) DelWithPrefix(keyPrefix string) error {
	return c.CacheClient.DelWithPrefix(keyPrefix)
}

// SetCache2Enabled 开启或关闭2级缓存
func (c *Cache) SetCache2Enabled(enabled bool) {
	c.Cache2Enabled = enabled
}

// Get 可选维度获取缓存数据
func (c *Cache) Get(fields ...string) (string, error) {

	cache1, err := c.CacheClient.Get(c.GetCacheLayerKey(1, fields...))
	if err == nil {
		return cache1, nil
	}

	if c.Lock(c.GetLockKey(fields), LockExpire) {
		var (
			newVal string
			err    error
		)
		if c.FuncReadData != nil {
			newVal, err = c.FuncReadData(fields...)
			if err != nil {
				cache2, err := c.CacheClient.Get(c.GetCacheLayerKey(2, fields...))
				if err == nil {
					return cache2, err
				}
				return "", err
			}
		}
		c.CacheClient.Set(c.GetCacheLayerKey(1, fields...), newVal, c.ExpireTime)
		if c.Cache2Enabled {
			// c.CacheClient.Set(c.GetCacheLayerKey(2, fields...), newVal, c.ExpireTime+DefaultCache2ExpirePadding)
			c.CacheClient.Set(c.GetCacheLayerKey(2, fields...), newVal, -1)
		}
		c.UnLock(c.GetLockKey(fields))
		return newVal, nil
	}

	if c.Cache2Enabled {
		cache2, err := c.CacheClient.Get(c.GetCacheLayerKey(2, fields...))
		if err == nil {
			return cache2, nil
		}
		return "", errors.New("data not exist")
	}
	return "", errors.New("data not exist")
}

// GetCacheLayerKey 按层维度格式化的键
func (c *Cache) GetCacheLayerKey(layer int, fs ...string) string {
	key := c.KeyPrefix

	suffix := ""
	for _, f := range fs {
		suffix += "_" + f
	}
	suffix += fmt.Sprintf("_%d", layer)

	suffix = SHA256(suffix)

	return key + suffix
}

// GetLockKey 按维度格式化锁的键
func (c *Cache) GetLockKey(fs []string) string {
	key := c.KeyPrefix

	suffix := ""
	for _, f := range fs {
		suffix += "_" + f
	}
	suffix += LockSuffix

	suffix = SHA256(suffix)

	return key + suffix
}

// NewCache 创建缓存实例
// keyPrefix 键前缀
// funcReadData 按多维度读取数据的回调
// expireTime 缓存过期时间 0 表示默认过期时间
// cache2Enabled 是否开启2级缓存
func NewCache(client ClientBase, keyPrefix string, funcReadData func(...string) (string, error), expireTime time.Duration, cache2Enabled bool) *Cache {
	if client == nil {
		logrus.Panic("cache client can not be nil")
	}

	if funcReadData == nil {
		logrus.Warn("funcReadData is nil")
	}

	if expireTime == 0 {
		expireTime = DefaultCacheExpire
	}

	c := &Cache{
		KeyPrefix:     keyPrefix,
		FuncReadData:  funcReadData,
		ExpireTime:    expireTime,
		Cache2Enabled: cache2Enabled,
	}

	c.CacheClient = client
	c.CacheClient.Connect()

	return c
}

// SHA256  SHA256加密
func SHA256(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}
