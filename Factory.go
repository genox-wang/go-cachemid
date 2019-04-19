package gocachemid

import (
	"crypto/md5"
	"errors"
	"fmt"
	"time"
)

const (
	// LockSuffix 分布式锁后缀
	LockSuffix = "_lock"
	// LockExpire 锁过期时间
	LockExpire = time.Second * 10
	// DefaultCacheExpire 默认缓存过期时间
	DefaultCacheExpire = time.Second * 10
	// DefaultCache2ExpirePadding 默认二级缓存间隔
	DefaultCache2ExpirePadding = -1
)

// FuncReadData 自定义获取数据的方法
// 当不缓存不存在会把，会到把参数传到这个方法去获取数据
// result 获取的数据
// err 获取数据报错
// 是否要缓存数据
type FuncReadData func(...string) (result string, err error, toCache bool)

// Cache 自定义缓存类，支持2级缓存
// 定时释放冷数据
// 防雪崩
// 数据源数据更新操作完成，调用Del释放缓存，来优化缓存数据一致性
type Cache struct {
	CacheClient      ClientBase
	KeyPrefix        string
	ExpireTime       time.Duration // 过期时间
	Cache2Enabled    bool          // 是否开启二级缓存
	Cache2ExpireTime time.Duration // 二级缓存过期时间
	FuncReadData     FuncReadData
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
func (c *Cache) Get(fields ...string) (string, bool, error) {

	cache1, err := c.CacheClient.Get(c.GetCacheLayerKey(1, fields...))
	if err == nil {
		return cache1, true, nil
	}

	if c.Lock(c.GetLockKey(fields), LockExpire) {
		var (
			newVal  string
			err     error
			toCache bool
		)
		if c.FuncReadData != nil {
			newVal, err, toCache = c.FuncReadData(fields...)
			if err != nil {
				cache2, err := c.CacheClient.Get(c.GetCacheLayerKey(2, fields...))
				if err == nil {
					return cache2, false, err
				}
				return "", false, err
			}
		}
		if toCache { // 需要缓存
			c.CacheClient.Set(c.GetCacheLayerKey(1, fields...), newVal, c.ExpireTime)
			if c.Cache2Enabled {
				c.CacheClient.Set(c.GetCacheLayerKey(2, fields...), newVal, -1)
			}
		}
		c.UnLock(c.GetLockKey(fields))
		return newVal, false, nil
	}

	if c.Cache2Enabled {
		cache2, err := c.CacheClient.Get(c.GetCacheLayerKey(2, fields...))
		if err == nil {
			return cache2, true, nil
		}
		return "", false, errors.New("data not exist")
	}
	return "", false, errors.New("data not exist")
}

// GetCacheLayerKey 按层维度格式化的键
func (c *Cache) GetCacheLayerKey(layer int, fs ...string) string {
	key := c.KeyPrefix

	suffix := ""
	for _, f := range fs {
		suffix += "_" + f
	}

	suffix = _md5(suffix)

	return fmt.Sprintf("%s:%s:%d", key, suffix, layer)
}

// GetLockKey 按维度格式化锁的键
func (c *Cache) GetLockKey(fs []string) string {
	key := c.KeyPrefix

	suffix := ""
	for _, f := range fs {
		suffix += "_" + f
	}

	suffix = _md5(suffix)

	return fmt.Sprintf("%s:%s:%d", key, suffix, LockExpire)
}

// // SHA256  SHA256加密
// func SHA256(s string) string {
// 	h := sha256.New()
// 	h.Write([]byte(s))
// 	return fmt.Sprintf("%x", h.Sum(nil))
// }

func _md5(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}
