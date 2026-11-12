package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// Todas las respuestas, también los errores y las rutas que no existen,
// llevan las cuatro cabeceras de seguridad.
func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	cases := []struct{ method, path string }{
		{"GET", "/"},
		{"GET", "/api/apps"},
		{"GET", "/api/apps/no-existe/icon"},
		{"GET", "/api/apps/no-existe/apk"},
		{"GET", "/api/stats"}, // 401 sin token
		{"POST", "/api/apps"}, // 401 sin token
		{"GET", "/no-existe"}, // 404 del mux
		{"PUT", "/api/apps"},  // 405 del mux
	}
	for _, c := range cases {
		req, _ := http.NewRequest(c.method, srv.URL+c.path, nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		for _, h := range []string{"Strict-Transport-Security", "Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy"} {
			if res.Header.Get(h) == "" {
				t.Errorf("%s %s (%d): falta %s", c.method, c.path, res.StatusCode, h)
			}
		}
		if got := res.Header.Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("%s %s: X-Content-Type-Options = %q", c.method, c.path, got)
		}
		if got := res.Header.Get("Strict-Transport-Security"); !strings.HasPrefix(got, "max-age=") {
			t.Errorf("%s %s: Strict-Transport-Security = %q", c.method, c.path, got)
		}
	}
}

// La CSP no admite 'unsafe-inline' y autoriza por hash exactamente el
// <script> y el <style> en línea del panel: si alguien los edita, el hash
// se recalcula solo y este test comprueba que coincide con lo que haría el navegador.
func TestCSPAllowsAdminPageInlineCode(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	res, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	csp := res.Header.Get("Content-Security-Policy")
	if strings.Contains(csp, "unsafe-inline") || strings.Contains(csp, "unsafe-eval") {
		t.Fatalf("CSP demasiado permisiva: %s", csp)
	}
	for _, d := range []string{"default-src 'none'", "connect-src 'self'", "img-src 'self'", "frame-ancestors 'none'", "base-uri 'none'", "form-action 'none'"} {
		if !strings.Contains(csp, d) {
			t.Errorf("falta %q en la CSP: %s", d, csp)
		}
	}
	directive := func(name string) string {
		for _, part := range strings.Split(csp, ";") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, name+" ") {
				return part
			}
		}
		return ""
	}
	for tag, dir := range map[string]string{"script": "script-src", "style": "style-src"} {
		blocks := regexp.MustCompile(`(?s)<`+tag+`>(.*?)</`+tag+`>`).FindAllSubmatch(adminPage, -1)
		if len(blocks) == 0 {
			t.Fatalf("el panel no tiene <%s> en línea; revisa el test", tag)
		}
		for _, b := range blocks {
			sum := sha256.Sum256(bytes.ReplaceAll(b[1], []byte("\r\n"), []byte("\n")))
			want := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
			if !strings.Contains(directive(dir), want) {
				t.Errorf("%s no autoriza el <%s> del panel (%s): %s", dir, tag, want, csp)
			}
		}
	}
}

// El navegador normaliza CRLF y CR sueltos a LF antes de calcular el hash:
// un panel con CRLF debe dar el mismo hash que con LF. El valor esperado es
// el SHA-256 en base64 de "a\nb" (printf 'a\nb' | sha256sum).
func TestInlineHashesNormalizeNewlines(t *testing.T) {
	const want = "'sha256-fhj3NzEbLcOy8mndeDlrA1HxT7Zu+oefdoyyMYGIPHg='"
	for _, page := range []string{"<script>a\nb</script>", "<script>a\r\nb</script>", "<script>a\rb</script>"} {
		if got := inlineHashes([]byte(page), "script"); got != want {
			t.Errorf("%q: %s, quiero %s", page, got, want)
		}
	}
	if got := inlineHashes([]byte("<p>sin script</p>"), "script"); got != "'none'" {
		t.Errorf("sin bloques: %s, quiero 'none'", got)
	}
}
