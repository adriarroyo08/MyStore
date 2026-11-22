package main

import (
	"fmt"
	"log"
	"net"
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
	// Lo usa el HEALTHCHECK del Dockerfile: la imagen distroless no tiene curl.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := healthcheck(healthURL(env("MYSTORE_ADDR", ":8080"))); err != nil {
			log.Fatal(err)
		}
		return
	}
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

// healthURL convierte la dirección de escucha en la URL de /healthz vista desde el propio
// contenedor: si escucha en todas las interfaces se pregunta a 127.0.0.1.
func healthURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		host, port = "", "8080"
	}
	if ip := net.ParseIP(host); host == "" || (ip != nil && ip.IsUnspecified()) {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + "/healthz"
}

func healthcheck(url string) error {
	c := http.Client{Timeout: 3 * time.Second}
	res, err := c.Get(url)
	if err != nil {
		return err
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("healthz respondió %d", res.StatusCode)
	}
	return nil
}
