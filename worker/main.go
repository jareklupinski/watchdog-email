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
			log.Panicf("Failed to set Watchdog.Email timer for %s: %v", emailAddress, err)
		}
		return false
	}
}

func runForever(quit <-chan os.Signal, redisContext *util.RedisController) {
	ticker := time.NewTicker(10 * time.Minute)
	for {
		select {
		case <-ticker.C:
			i := 0
			for checkWatchdog(redisContext) {
				i++
				select {
				case <-quit:
					log.Println("Watchdog Routine Interrupted")
					break
				default:
					continue
				}
			}
			log.Printf("Watchdog.Email Worker Sent %d emails\n", i)
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func main() {
	log.Println("Watchdog.Email Worker Starting")

	redisContext := util.NewRedisController()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go runForever(quit, redisContext)

	<-quit

	redisContext.CloseRedisController()

	log.Println("Watchdog.Email Worker Exiting")
	os.Exit(0)
}
