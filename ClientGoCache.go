package gocachemid

import (
	"errors"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

// ClientGoCache GoCache 缓存客户端
type ClientGoCache struct {
	client *cache.Cache
}

// Set GoCache 设置缓存数据
func (c *ClientGoCache) Set(key string, val string, expireTime time.Duration) error {
	c.client.Set(key, val, expireTime)
	return nil
}

// Add GoCache 新增缓存数据，存在则返回 false
func (c *ClientGoCache) Add(key string, val string, expireTime time.Duration) bool {
	return c.client.Add(key, val, expireTime) == nil
}

// Get GoCache 读取缓存数据
func (c *ClientGoCache) Get(key string) (string, error) {
	r, success := c.client.Get(key)
	if success {
		return r.(string), nil
	}
	return "", errors.New("not exist")
}

// Del GoCache 删除缓存数据
func (c *ClientGoCache) Del(key string) error {
	c.client.Delete(key)
	return nil
}

// DelWithPrefix GoCache 删除前缀为 keyPrefix 的缓存
func (c *ClientGoCache) DelWithPrefix(keyPrefix string) error {
	items := c.client.Items()
	for k := range items {
		if strings.HasPrefix(k, keyPrefix) {
			c.client.Delete(k)
		}
	}
	return nil
}

// Connect 连接 GoCache 服务器
// func (c *ClientGoCache) Connect() error {

// 	c.client = cache.New(DefaultCacheExpire, time.Minute*10)

// 	return nil
// }
