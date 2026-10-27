package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	token := os.Getenv("MYSTORE_TOKEN")
	if token == "" {
		log.Fatal("falta la variable MYSTORE_TOKEN")
	}
	maxMB, err := strconv.ParseInt(env("MYSTORE_MAX_UPLOAD_MB", "300"), 10, 64)
	if err != nil || maxMB <= 0 {
		log.Fatal("MYSTORE_MAX_UPLOAD_MB no es un número válido")
	}
	store, err := NewStore(env("MYSTORE_DATA", "/data"))
	if err != nil {
		log.Fatal(err)
	}
	addr := env("MYSTORE_ADDR", ":8080")
	log.Printf("MyStore escuchando en %s", addr)
	log.Fatal(newServer(addr, NewHandler(store, token, maxMB<<20)).ListenAndServe())
}

// Sin WriteTimeout: cortaría las descargas largas de APKs.
func newServer(addr string, h http.Handler) *http.Server {
	return &http.Server{Addr: addr, Handler: h, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 2 * time.Minute}
}
