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
		log.Panic("No Entries In Database!")
	}

	emailAddress := redisStrings[0]
	alarm, err := strconv.ParseInt(redisStrings[1], 10, 64)
	if err != nil {
		log.Panicf("Failed Parsing Redis Time: %v", err)
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
			log.Panicf("Failed to set Watchdog.Email timer for %s: %v", emailAddress, err)
		}
		return false
	}
}

func runForever(quit <-chan os.Signal, ready chan<- bool) {
	redisContext := util.NewRedisController()

	ticker := time.NewTicker(10 * time.Minute)
	multiball := make(chan bool)

	log.Println("Watchdog.Email Worker Running")
	for {
		select {
		case <-ticker.C:
			fallthrough
		case <-multiball:
			if checkWatchdog(redisContext) {
				log.Println("Watchdog.Email Worker Sent an email")
				multiball <- true
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
