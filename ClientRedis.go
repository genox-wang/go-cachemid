package gocachemid

import (
	"time"

	"github.com/go-redis/redis"
)

// ClientRedis Redis 缓存客户端
type ClientRedis struct {
	client *redis.Client
}

// Set Redis 设置缓存数据
func (c *ClientRedis) Set(key string, val string, expireTime time.Duration) error {
	return c.client.Set(key, val, expireTime).Err()
}

// Add Redis 新增缓存数据，存在则返回 false
func (c *ClientRedis) Add(key string, val string, expireTime time.Duration) bool {
	return c.client.SetNX(key, val, expireTime).Val()
}

// Get Redis 读取缓存数据
func (c *ClientRedis) Get(key string) (string, error) {
	r := c.client.Get(key)
	return r.Val(), r.Err()
}

// Del Redis 删除缓存数据
func (c *ClientRedis) Del(key string) error {
	return c.client.Del(key).Err()
}

// Connect 连接 Redis 服务器
func (c *ClientRedis) Connect() error {

	c.client = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	_, err := c.client.Ping().Result()
	return err
}
