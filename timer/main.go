package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	log.SetFlags(0)
	log.Println("Watchdog.Email Timer Starting")

	_, err := http.Get("http://watchdog.email/")
	if err != nil {
		log.Panic(err)
	}

	log.Println("Watchdog.Email Timer Exiting")
	os.Exit(0)
}
