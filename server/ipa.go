package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image/png"
	"io"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

var infoPlistRe = regexp.MustCompile(`^Payload/[^/]+\.app/Info\.plist$`)

// isIPA dice si el zip contiene Payload/<X>.app/Info.plist.
func isIPA(p string) bool {
	zr, err := zip.OpenReader(p)
	if err != nil {
		return false
	}
	defer zr.Close()
	for _, f := range zr.File {
		if infoPlistRe.MatchString(f.Name) {
			return true
		}
	}
	return false
}

func readZipFile(f *zip.File, max int64) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, max))
}

// buildNumber convierte CFBundleVersion en un entero que crece con la versión:
// "7" → 7; "a.b.c" → a*1_000_000+b*1_000+c.
func buildNumber(s string) (int64, error) {
	parts := strings.Split(strings.TrimSpace(s), ".")
	if len(parts) > 3 {
		return 0, fmt.Errorf("CFBundleVersion %q no admitido", s)
	}
	var n int64
	for _, p := range parts {
		v, err := strconv.ParseInt(p, 10, 64)
		if err != nil || v < 0 {
			return 0, fmt.Errorf("CFBundleVersion %q no es numérico", s)
		}
		if len(parts) == 1 {
			return v, nil
		}
		n = n*1000 + v
	}
	for i := len(parts); i < 3; i++ {
		n *= 1000
	}
	return n, nil
}

// ParseIPA lee los metadatos y el icono (PNG, o nil) de un .ipa. No hay firma que comprobar:
// los perfiles de aprovisionamiento cambian el certificado en cada reexportación.
func ParseIPA(p string) (Meta, []byte, error) {
	invalid := func(err error) (Meta, []byte, error) {
		return Meta{}, nil, fmt.Errorf("%w: %v", ErrInvalidAPK, err)
	}
	st, err := os.Stat(p)
	if err != nil {
		return Meta{}, nil, err
	}
	zr, err := zip.OpenReader(p)
	if err != nil {
		return invalid(err)
	}
	defer zr.Close()
	var plistFile *zip.File
	for _, f := range zr.File {
		if infoPlistRe.MatchString(f.Name) {
			plistFile = f
			break
		}
	}
	if plistFile == nil {
		return invalid(fmt.Errorf("no hay Info.plist"))
	}
	data, err := readZipFile(plistFile, 5<<20)
	if err != nil {
		return invalid(err)
	}
	v, err := parsePlist(data)
	info, _ := v.(map[string]any)
	if err != nil || info == nil {
		return invalid(fmt.Errorf("Info.plist ilegible"))
	}
	str := func(k string) string { s, _ := info[k].(string); return s }
	m := Meta{
		Package:     str("CFBundleIdentifier"),
		Type:        "ios",
		Label:       str("CFBundleDisplayName"),
		VersionName: str("CFBundleShortVersionString"),
		MinOS:       str("MinimumOSVersion"),
		Size:        st.Size(),
	}
	if m.Package == "" {
		return invalid(fmt.Errorf("falta CFBundleIdentifier"))
	}
	if m.Label == "" {
		m.Label = str("CFBundleName")
	}
	if m.Label == "" {
		m.Label = m.Package
	}
	build := str("CFBundleVersion")
	if n, ok := info["CFBundleVersion"].(int64); ok {
		build = strconv.FormatInt(n, 10)
	}
	if m.VersionCode, err = buildNumber(build); err != nil {
		return invalid(err)
	}
	if m.VersionName == "" {
		m.VersionName = build
	}
	return m, ipaIcon(zr, path.Dir(plistFile.Name), info), nil
}

// ipaIcon busca el PNG más grande cuyo nombre empiece por los de CFBundleIconFiles (o AppIcon).
// ponytail: Xcode suele "aplastar" los PNG al empaquetar (formato Apple CgBI, que image/png no
// lee); entonces se devuelve nil y se queda el icono genérico. Desaplastarlo (deflate raw, BGRA
// premultiplicado) si molesta; mientras, PUT /api/apps/{id}/icon.
func ipaIcon(zr *zip.ReadCloser, appDir string, info map[string]any) []byte {
	prefixes := []string{}
	for _, k := range []string{"CFBundleIcons", "CFBundleIcons~ipad"} {
		icons, _ := info[k].(map[string]any)
		primary, _ := icons["CFBundlePrimaryIcon"].(map[string]any)
		files, _ := primary["CFBundleIconFiles"].([]any)
		for _, f := range files {
			if s, ok := f.(string); ok {
				prefixes = append(prefixes, s)
			}
		}
	}
	prefixes = append(prefixes, "AppIcon")
	var best *zip.File
	for _, f := range zr.File {
		if path.Dir(f.Name) != appDir || !strings.HasSuffix(strings.ToLower(f.Name), ".png") {
			continue
		}
		base := strings.TrimSuffix(path.Base(f.Name), path.Ext(f.Name))
		for _, pre := range prefixes {
			if strings.HasPrefix(base, strings.TrimSuffix(pre, ".png")) {
				if best == nil || f.UncompressedSize64 > best.UncompressedSize64 {
					best = f
				}
				break
			}
		}
	}
	if best == nil {
		return nil
	}
	data, err := readZipFile(best, 5<<20)
	if err != nil {
		return nil
	}
	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
		return nil
	}
	return data
}
