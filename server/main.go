package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
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
	log.Fatal(http.ListenAndServe(addr, NewHandler(store, token, maxMB<<20)))
}
