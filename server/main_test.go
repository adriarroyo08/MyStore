package main

import (
	"net/http"
	"os"
	"testing"
)

func TestServerTimeouts(t *testing.T) {
	srv := newServer(":0", http.NotFoundHandler())
	if srv.ReadHeaderTimeout == 0 || srv.IdleTimeout == 0 {
		t.Errorf("timeouts not set: %+v", srv)
	}
	if srv.WriteTimeout != 0 {
		t.Error("WriteTimeout would cut long APK downloads")
	}
}

func TestHealthz(t *testing.T) {
	srv, s := newTestServer(t, 1<<20)
	if res := do(t, "GET", srv.URL+"/healthz", "", nil, ""); res.StatusCode != 200 {
		t.Fatalf("healthz = %d", res.StatusCode)
	}
	if err := healthcheck(srv.URL + "/healthz"); err != nil {
		t.Errorf("healthcheck con servidor sano: %v", err)
	}
	if len(s.Devices()) != 0 {
		t.Error("healthz no debe registrar dispositivos")
	}
	os.RemoveAll(s.apps) // datos inaccesibles
	if res := do(t, "GET", srv.URL+"/healthz", "", nil, ""); res.StatusCode != 503 {
		t.Errorf("healthz sin datos = %d", res.StatusCode)
	}
	if healthcheck(srv.URL+"/healthz") == nil {
		t.Error("healthcheck debería fallar con 503")
	}
	srv.Close()
	if healthcheck(srv.URL+"/healthz") == nil {
		t.Error("healthcheck debería fallar sin servidor")
	}
}

func TestHealthURL(t *testing.T) {
	for addr, want := range map[string]string{
		":8080":          "http://127.0.0.1:8080/healthz",
		"0.0.0.0:9000":   "http://127.0.0.1:9000/healthz",
		"[::]:9000":      "http://127.0.0.1:9000/healthz",
		"127.0.0.1:8081": "http://127.0.0.1:8081/healthz",
		"[::1]:8081":     "http://[::1]:8081/healthz",
	} {
		if got := healthURL(addr); got != want {
			t.Errorf("%q → %q, quiero %q", addr, got, want)
		}
	}
}
