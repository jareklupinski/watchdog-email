package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
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
		log.Panicf("Failed Popping Redis Entry: %v", err)
	}
	if len(redisStrings) < 2 {
		log.Println("No Entries In Database!")
		return false
	}

	emailAddress := redisStrings[0]
	alarm, err := strconv.ParseInt(redisStrings[1], 10, 64)
	if err != nil {
		log.Printf("Failed Parsing Redis Time: %v\n", err)
		alarm = 0
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
			log.Panicf("Failed to re-add Watchdog.Email timer for %s: %v", emailAddress, err)
		}
		return false
	}
}

func runForever(quit <-chan os.Signal, ready chan<- bool) {
	redisContext := util.NewRedisController()

	ticker := time.NewTicker(10 * time.Minute)
	multiballTicker := time.NewTicker(10 * time.Millisecond)

	log.Println("Watchdog.Email Worker Running")

	multiball := checkWatchdog(redisContext)
	for {
		select {
		case <-ticker.C:
			multiball = checkWatchdog(redisContext)
		case <-multiballTicker.C:
			if multiball {
				multiball = checkWatchdog(redisContext)
			}
		case <-quit:
			ticker.Stop()
			redisContext.CloseRedisController()
			ready <- true
		}
	}
}

func main() {
	log.Println("Watchdog.Email Worker Starting")

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	ready := make(chan bool)

	go runForever(quit, ready)

	<-ready

	log.Println("Watchdog.Email Worker Exiting")
	os.Exit(0)
}
