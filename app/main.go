// A tiny demo app with two roles:
//
//	server  – answers /orders with a random mix of successes and errors
//	client  – calls the server in a loop to generate traffic
package main

import (
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"time"
)

func main() {
	switch os.Getenv("ROLE") {
	case "server":
		runServer()
	case "client":
		runClient(os.Getenv("TARGET"))
	default:
		log.Fatal("set ROLE to server or client")
	}
}

func runServer() {
	http.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		switch n := rand.IntN(100); {
		case n < 70:
			w.Write([]byte("ok\n"))
		case n < 85:
			http.Error(w, "order not found", http.StatusNotFound)
		case n < 95:
			http.Error(w, "database exploded", http.StatusInternalServerError)
		default:
			http.Error(w, "try again later", http.StatusServiceUnavailable)
		}
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func runClient(target string) {
	for range time.Tick(200 * time.Millisecond) {
		resp, err := http.Get(target)
		if err != nil {
			log.Println(err)
			continue
		}
		io.Copy(io.Discard, resp.Body) // drain so the connection is reused
		resp.Body.Close()
	}
}
