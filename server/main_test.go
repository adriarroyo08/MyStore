package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
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

// Al cancelar el contexto (SIGTERM) la petición en curso termina bien y serve vuelve sin error.
func TestServeShutsDownGracefully(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		w.Write([]byte("ok"))
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serve(ctx, newServer("", h), ln) }()

	got := make(chan string, 1)
	go func() {
		res, err := http.Get("http://" + ln.Addr().String())
		if err != nil {
			got <- err.Error()
			return
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		got <- string(b)
	}()
	<-entered
	cancel()
	time.Sleep(50 * time.Millisecond)
	select {
	case err := <-done:
		t.Fatalf("serve volvió con una petición en curso: %v", err)
	default:
	}
	close(release)
	if b := <-got; b != "ok" {
		t.Errorf("respuesta = %q", b)
	}
	if err := <-done; err != nil {
		t.Errorf("serve = %v", err)
	}
}

// Una petición que no acaba no bloquea el apagado más allá de shutdownTimeout.
func TestServeShutdownTimeout(t *testing.T) {
	old := shutdownTimeout
	shutdownTimeout = 100 * time.Millisecond
	t.Cleanup(func() { shutdownTimeout = old })
	entered := make(chan struct{})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-r.Context().Done() // se libera cuando el servidor cierra la conexión
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serve(ctx, newServer("", h), ln) }()
	go http.Get("http://" + ln.Addr().String())
	<-entered
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("serve = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve no volvió tras el plazo de apagado")
	}
}
