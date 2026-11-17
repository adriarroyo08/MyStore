package main

import (
	"strings"
	"testing"
)

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
