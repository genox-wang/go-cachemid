package gocachemid

import "time"

// ClientBase 缓存客户端接口定义
type ClientBase interface {
	Set(key string, val string, expireTime time.Duration) error
	Add(key string, val string, expireTime time.Duration) bool
	Get(key string) (string, error)
	Del(key string) error
	DelWithPrefix(keyPrefix string) error
}
