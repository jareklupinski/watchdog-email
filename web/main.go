package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
	_ "github.com/heroku/x/hmetrics/onload"

	"watchdog-email/util"
)

func startWatchdog(pool *redis.Pool) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		email := c.Query("email")
		if email == "" {
			c.HTML(http.StatusOK, "index.html", nil)
			return
		}
		if !util.EmailIsValid(email) {
			c.String(http.StatusBadRequest, "Cannot set Watchdog.Email timer for %s", email)
			return
		}
		conn := pool.Get()
		defer func() {
			err := conn.Close()
			if err != nil {
				log.Panic(err)
			}
		}()
		now := time.Now().Unix()
		alarm := now + 90000 // (60 seconds / minute) * (60 minutes / hour) * (25 hours / timeout)
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
