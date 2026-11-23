package main

import (
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// luminance y contrast siguen la fórmula WCAG 2.x.
func luminance(hex string) float64 {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	var c [3]float64
	for i := range c {
		v, _ := strconv.ParseUint(hex[i*2:i*2+2], 16, 8)
		f := float64(v) / 255
		if f <= 0.03928 {
			c[i] = f / 12.92
		} else {
			c[i] = math.Pow((f+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*c[0] + 0.7152*c[1] + 0.0722*c[2]
}

func contrast(a, b string) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// themeTokens devuelve las variables de cada bloque :root (claro y oscuro).
func themeTokens(t *testing.T) (light, dark map[string]string) {
	t.Helper()
	blocks := regexp.MustCompile(`:root \{([^}]*)\}`).FindAllStringSubmatch(string(adminPage), -1)
	if len(blocks) != 2 {
		t.Fatalf("esperaba 2 bloques :root (claro y oscuro), hay %d", len(blocks))
	}
	parse := func(body string) map[string]string {
		m := map[string]string{}
		for _, kv := range regexp.MustCompile(`--([a-z-]+):\s*(#[0-9a-fA-F]{3,6})`).FindAllStringSubmatch(body, -1) {
			m[kv[1]] = kv[2]
		}
		return m
	}
	return parse(blocks[0][1]), parse(blocks[1][1])
}

func TestAdminPageContrast(t *testing.T) {
	light, dark := themeTokens(t)
	pairs := []struct {
		fg, bg string
		min    float64
	}{
		{"fg", "bg", 4.5}, {"fg", "card", 4.5},
		{"muted", "bg", 5}, {"muted", "card", 5}, // .tag de 11 px
		{"accent", "bg", 4.5}, {"accent", "card", 4.5},
		{"danger", "bg", 4.5}, {"danger", "card", 4.5},
		{"ok", "bg", 4.5}, {"ok", "card", 4.5},
		{"on-accent", "accent", 4.5}, {"on-danger", "danger", 4.5},
	}
	for name, theme := range map[string]map[string]string{"claro": light, "oscuro": dark} {
		for _, p := range pairs {
			fg, bg := theme[p.fg], theme[p.bg]
			if fg == "" || bg == "" {
				t.Errorf("%s: falta el token %s o %s", name, p.fg, p.bg)
				continue
			}
			if c := contrast(fg, bg); c < p.min {
				t.Errorf("%s: %s %s sobre %s %s da %.2f:1 (mínimo %.1f)", name, p.fg, fg, p.bg, bg, c, p.min)
			}
		}
	}
}

func TestAdminPageNoHardcodedWhiteOnColor(t *testing.T) {
	if regexp.MustCompile(`background:var\(--(accent|danger)\)[^}]*color:#fff`).MatchString(string(adminPage)) {
		t.Error("un botón sobre acento o peligro usa #fff fijo; usa --on-accent / --on-danger")
	}
}

func TestAdminPageBrandAccent(t *testing.T) {
	light, _ := themeTokens(t)
	if !strings.EqualFold(light["accent"], "#1b5e9e") { // BrandBlue de android/mobile Theme.kt
		t.Errorf("el acento claro debe ser el azul de marca de la app, es %s", light["accent"])
	}
}

func TestAdminPageMetadataAndImages(t *testing.T) {
	page := string(adminPage)
	for _, want := range []string{`<meta name="description"`, `<meta name="theme-color"`, `rel="icon"`} {
		if !strings.Contains(page, want) {
			t.Errorf("falta %s", want)
		}
	}
	if strings.Contains(page, `alt: ""`) {
		t.Error("hay imágenes con alt vacío")
	}
	for _, m := range regexp.MustCompile(`el\("img", \{.*?\}\)`).FindAllString(page, -1) {
		if !strings.Contains(m, "width:") || !strings.Contains(m, "height:") {
			t.Errorf("imagen sin width/height: %s", m)
		}
	}
}

func TestFavicon(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	res := do(t, "GET", srv.URL+"/favicon.svg", "", nil, "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("favicon: %d %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
}

// El panel es HTML estático sin lógica testeable desde Go: estas pruebas fijan
// los contratos de legibilidad en móvil para que una edición no los pierda.
func TestAdminPageMobileContracts(t *testing.T) {
	page := string(adminPage)
	for name, want := range map[string]string{
		"columna bajo 480 px":       "@media (max-width:480px)",
		"botones de 44 px":          "min-height:44px",
		"fechas en castellano":      `toLocaleDateString("es-ES"`,
		"agrupa móvil y reloj":      "byAppThenType",
		"sin fecha según navegador": "fmtDate",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("falta %s (%q) en web/index.html", name, want)
		}
	}
	if strings.Contains(page, ".toLocaleString()") || strings.Contains(page, ".toLocaleDateString()") {
		t.Error("hay fechas sin locale fijo")
	}
}
