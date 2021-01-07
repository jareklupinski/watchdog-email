package util

import (
	"time"

	"github.com/gomodule/redigo/redis"
)

func NewPool(addr string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     50,
		IdleTimeout: 240 * time.Second,
		MaxActive:   50,
		Wait:        true,
		Dial:        func() (redis.Conn, error) { return redis.DialURL(addr) },
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}
