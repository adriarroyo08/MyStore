package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"regexp"
	"strings"
)

// contentSecurityPolicy es la misma para todas las respuestas. El panel
// (web/index.html) es la única página HTML: su <script> y su <style> en línea
// se autorizan por hash, calculado al arrancar a partir del fichero embebido,
// así que editar el panel no obliga a tocar la política. Las respuestas JSON,
// APK e imágenes no ejecutan nada y la política no les afecta.
var contentSecurityPolicy = strings.Join([]string{
	"default-src 'none'",
	"script-src " + inlineHashes(adminPage, "script"),
	"style-src " + inlineHashes(adminPage, "style"),
	"img-src 'self'",
	"connect-src 'self'",
	"base-uri 'none'",
	"form-action 'none'",
	"frame-ancestors 'none'",
}, "; ")

// inlineHashes devuelve las fuentes 'sha256-…' de cada <tag> sin atributos de page.
// El navegador calcula el hash después de normalizar los saltos de línea a LF
// (preprocesado del parser HTML), así que aquí se hace lo mismo: si el fichero
// llega con CRLF (checkout en Windows), el hash sigue coincidiendo.
func inlineHashes(page []byte, tag string) string {
	re := regexp.MustCompile(`(?s)<` + tag + `>(.*?)</` + tag + `>`)
	var srcs []string
	for _, m := range re.FindAllSubmatch(page, -1) {
		body := bytes.ReplaceAll(m[1], []byte("\r\n"), []byte("\n"))
		body = bytes.ReplaceAll(body, []byte("\r"), []byte("\n"))
		sum := sha256.Sum256(body)
		srcs = append(srcs, "'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'")
	}
	if len(srcs) == 0 {
		return "'none'"
	}
	return strings.Join(srcs, " ")
}

// securityHeaders añade las cabeceras de seguridad a todas las respuestas,
// incluidos los errores 404 y 405 del mux. HSTS solo lo aplica el navegador
// cuando la respuesta llega por HTTPS (detrás del proxy), así que en local por
// HTTP no molesta. Sin includeSubDomains ni preload: este servidor solo
// responde por su propio nombre y no debe imponer HTTPS a otros.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Strict-Transport-Security", "max-age=31536000")
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
