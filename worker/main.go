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

func checkWatchdog(pool *redis.Pool) bool {
	conn := pool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Panic(err)
		}
	}()
	now := time.Now().Unix()

	redisStrings, err := redis.Strings(conn.Do("ZPOPMIN", "email"))
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
		rows, err := redis.Int(conn.Do("ZADD", "email", "CH", alarm, emailAddress))
		if err != nil || rows < 1 {
			log.Panicf("Failed to re-add Watchdog.Email timer for %s: %v", emailAddress, err)
		}
		return false
	}
}

func runForever(quit <-chan os.Signal, ready chan<- bool) {
	addr := os.Getenv("REDIS_URL")
	pool := util.NewPool(addr)

	ticker := time.NewTicker(1 * time.Minute)
	multiballTicker := time.NewTicker(10 * time.Millisecond)

	log.Println("Watchdog.Email Worker Running")

	multiball := checkWatchdog(pool)
	for {
		select {
		case <-ticker.C:
			multiball = checkWatchdog(pool)
		case <-multiballTicker.C:
			if multiball {
				multiball = checkWatchdog(pool)
			}
		case <-quit:
			ticker.Stop()
			multiballTicker.Stop()
			err := pool.Close()
			if err != nil {
				log.Panic(err)
			}
			ready <- true
		}
	}
}

func main() {
	log.SetFlags(0)
	log.Println("Watchdog.Email Worker Starting")

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	ready := make(chan bool)

	go runForever(quit, ready)

	<-ready

	log.Println("Watchdog.Email Worker Exiting")
	os.Exit(0)
}
