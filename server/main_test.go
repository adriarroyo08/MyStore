package main

import (
	"net/http"
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
