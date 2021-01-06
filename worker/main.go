package main

import (
	"log"
	"time"

	"github.com/gomodule/redigo/redis"
	_ "github.com/heroku/x/hmetrics/onload"
	"watchdog-email/util"
)

func checkWatchdog(r *util.RedisController) {
	rds := r.GetRedisConnection()
	now := time.Now().Unix()

	emailAddresses, err := redis.Strings(rds.Do("ZRANGEBYSCORE", "email", 0, now))
	if err != nil {
		log.Panic(err)
	}

	for _, emailAddress := range emailAddresses {
		if emailAddress == "" {
			continue
		}
		util.SendEmail(emailAddress)
		rows, err := redis.Int(rds.Do("ZREM", "email", emailAddress))
		if err != nil || rows < 1 {
			log.Printf("Failed to remove email %s: %s\n", emailAddress, err)
		}
	}
}

func main() {
	log.Println("Watchdog.Email Worker Starting")
	redisContext := util.NewRedisController()
	log.Println("Watchdog.Email Worker Running")
	checkWatchdog(redisContext)
	redisContext.CloseRedisController()
	log.Println("Watchdog.Email Worker Exiting")
}
