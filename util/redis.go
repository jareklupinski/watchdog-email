package util

import (
	"log"
	"os"

	"github.com/gomodule/redigo/redis"
)

type RedisController struct {
	Conn redis.Conn
}

func NewRedisController() *RedisController {
	redisURL := os.Getenv("REDIS_URL")
	redisConnection, err := redis.DialURL(redisURL)
	if err != nil {
		log.Panicf("Failed to connect to $REDIS_URL %s\n", redisURL)
	}
	redisContext := RedisController{
		Conn: redisConnection,
	}
	return &redisContext
}

func (r *RedisController) GetRedisConnection() redis.Conn {
	rds := r.Conn
	if rds == nil {
		log.Panic("Redis Connection attempted before Redis Initialized!")
	}
	return rds
}

func (r *RedisController) CloseRedisController() {
	err := r.Conn.Close()
	if err != nil {
		log.Panicf("Failed to close connection to redis %s\n", err)
	}
}
