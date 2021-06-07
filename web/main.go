package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
	_ "github.com/heroku/x/hmetrics/onload"

	"watchdog-email/util"
)

func startWatchdog(pool *redis.Pool) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		conn := pool.Get()
		defer func() {
			err := conn.Close()
			if err != nil {
				log.Panic(err)
			}
		}()
		email := c.Query("email")
		timeout := c.Query("timeout")
		if email == "" {
			rows, err := redis.Int(conn.Do("ZCOUNT", "email", "-inf", "+inf"))
			if err != nil {
				log.Println(err)
				c.String(http.StatusServiceUnavailable, "Failed Loading Timers")
				return
			}
			nextTimeout := "0"
			if rows > 0 {
				strings, err := redis.Strings(conn.Do("ZRANGEBYSCORE", "email", "-inf", "+inf", "WITHSCORES", "LIMIT", "0", "1"))
				if err != nil {
					log.Println(err)
					c.String(http.StatusServiceUnavailable, "Failed Loading Timers")
					return
				}
				alarm, err := strconv.ParseInt(strings[1], 10, 64)
				if err != nil {
					log.Println(err)
					c.String(http.StatusServiceUnavailable, "Failed Loading Timers")
					return
				}
				nextTimeout = time.Unix(alarm, 0).String()
			}

			c.HTML(http.StatusOK, "index.html", gin.H{
				"numWatchdogs": rows,
				"nextTimeout":  nextTimeout,
			})
			return
		}
		if !util.EmailIsValid(email) {
			c.String(http.StatusBadRequest, "Cannot set Watchdog.Email timer for %s", email)
			return
		}
		timeoutValue := int64(90000) // (60 seconds / minute) * (60 minutes / hour) * (25 hours / timeout)
		if timeout != "" {
			timeoutValue, err := strconv.ParseInt(timeout, 10, 64)
			if err != nil || timeoutValue < 600 || timeoutValue > 90000 {
				log.Println(err)
				c.String(http.StatusBadRequest, "Invalid timeout value! Minimum 600 seconds (5 minutes), Maximum 90000 seconds (25 hours).")
				return
			}
		}
		now := time.Now().Unix()
		alarm := now + timeoutValue
		rows, err := redis.Int(conn.Do("ZADD", "email", "CH", alarm, email))
		if err != nil || rows < 1 {
			log.Printf("Failed to set Watchdog.Email timer for %s: %s", email, err)
			c.String(http.StatusInternalServerError, "Failed to set Watchdog.Email timer for %s", email)
			return
		}
		c.String(http.StatusOK, "Watchdog.Email has been set at %d to send an email to %s at %d", now, email, alarm)
	}
	return fn
}

func sendHead(c *gin.Context) {
	c.String(http.StatusOK, "")
	return
}

func runForever(quit <-chan os.Signal, ready chan<- bool) {
	addr := os.Getenv("REDIS_URL")
	pool := util.NewPool(addr)

	port := os.Getenv("PORT")
	if port == "" {
		log.Panic("$PORT must be set")
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.LoadHTMLGlob("templates/*.html")
	router.StaticFile("/robots.txt", "static/robots.txt")
	router.StaticFile("/favicon.ico", "static/favicon.ico")
	router.Static("/static", "static")
	router.HEAD("/", sendHead)
	router.GET("/", startWatchdog(pool))

	serverAddress := fmt.Sprintf(":%s", port)
	srv := &http.Server{
		Addr:    serverAddress,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Panicf("Server Error: %v", err)
		}
	}()

	log.Println("Watchdog.Email Server Running")

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Panicf("Server forced to shutdown: %v", err)
	}

	err := pool.Close()
	if err != nil {
		log.Panic(err)
	}

	ready <- true
}

func main() {
	log.SetFlags(0)
	log.Println("Watchdog.Email Server Starting")

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	ready := make(chan bool)

	go runForever(quit, ready)

	<-ready

	log.Println("Watchdog.Email Server Exiting")
	os.Exit(0)
}
