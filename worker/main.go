package main

import (
	"log"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
	_ "github.com/heroku/x/hmetrics/onload"

	"watchdog-email/util"
)

func checkWatchdog(r *util.RedisController) bool {
	rds := r.GetRedisConnection()
	now := time.Now().Unix()

	redisStrings, err := redis.Strings(rds.Do("ZPOPMIN", "email"))
	if err != nil {
		log.Panic(err)
	}
	if len(redisStrings) < 2 {
		log.Panic("No Entries In Database!")
	}

	emailAddress := redisStrings[0]
	alarm, err := strconv.ParseInt(redisStrings[1], 10, 64)
	if err != nil {
		log.Panic(err)
	}

	if alarm < now {
		if emailAddress == "" {
			log.Println("Empty Email Address")
		} else {
			util.SendEmail(emailAddress)
		}
		return true
	} else {
		rows, err := redis.Int(rds.Do("ZADD", "email", "CH", alarm, emailAddress))
		if err != nil || rows < 1 {
			log.Panicf("Failed to set Watchdog.Email timer for %s: %s", emailAddress, err)
		}
		return false
	}
}

func main() {
	log.Println("Watchdog.Email Worker Starting")
	redisContext := util.NewRedisController()
	log.Println("Watchdog.Email Worker Running")
	for checkWatchdog(redisContext) {
	}
	redisContext.CloseRedisController()
	log.Println("Watchdog.Email Worker Exiting")
}
