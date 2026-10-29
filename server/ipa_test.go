package main

import (
	"archive/zip"
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// writeIPA crea un .ipa en un temporal; files mapea ruta dentro del zip → contenido.
func writeIPA(t *testing.T, files map[string][]byte) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range files {
		w, _ := zw.Create(name)
		w.Write(data)
	}
	zw.Close()
	p := filepath.Join(t.TempDir(), "app.ipa")
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func xmlIPA(t *testing.T, id, short, build string, extra map[string][]byte) string {
	files := map[string][]byte{"Payload/Demo.app/Info.plist": []byte(xmlPlist(id, short, build)), "Payload/Demo.app/Demo": []byte("bin")}
	for k, v := range extra {
		files[k] = v
	}
	return writeIPA(t, files)
}

func TestParseIPAXML(t *testing.T) {
	p := xmlIPA(t, "com.example.ios", "1.2", "5", nil)
	m, icon, err := ParseIPA(p)
	if err != nil {
		t.Fatal(err)
	}
	if m.Package != "com.example.ios" || m.Type != "ios" || m.Label != "Demo X" || m.VersionName != "1.2" || m.VersionCode != 5 || m.MinOS != "16.0" || m.CertSHA256 != "" {
		t.Errorf("got %+v", m)
	}
	st, _ := os.Stat(p)
	if m.Size != st.Size() || icon != nil {
		t.Errorf("size %d icon %d", m.Size, len(icon))
	}
}

func TestParseIPADottedBuild(t *testing.T) {
	m, _, err := ParseIPA(xmlIPA(t, "com.example.ios", "1.0", "2.5.7", nil))
	if err != nil || m.VersionCode != 2_005_007 {
		t.Errorf("%+v %v", m, err)
	}
	if _, _, err := ParseIPA(xmlIPA(t, "com.example.ios", "1.0", "abc", nil)); err == nil {
		t.Error("build no numérico debería fallar")
	}
}

func TestParseIPABinaryPlistAndIcon(t *testing.T) {
	bp, _ := os.ReadFile("testdata/info.bplist")
	p := writeIPA(t, map[string][]byte{
		"Payload/Demo.app/Info.plist":           bp,
		"Payload/Demo.app/AppIcon60x60@2x.png":  pngBytes(40, 40),
		"Payload/Demo.app/AppIcon60x60@3x.png":  pngBytes(80, 80),
		"Payload/Demo.app/Other.png":            pngBytes(200, 200),
	})
	m, icon, err := ParseIPA(p)
	if err != nil {
		t.Fatal(err)
	}
	if m.Package != "com.example.ios" || m.Label != "Demo iOS ñ" || m.VersionName != "1.2" || m.VersionCode != 1_002_003 {
		t.Errorf("got %+v", m)
	}
	img, err := png.Decode(bytes.NewReader(icon))
	if err != nil || img.Bounds().Dx() != 80 {
		t.Errorf("icono: %v %v", err, icon != nil)
	}
}

func TestParseIPACrushedIcon(t *testing.T) {
	crushed := append([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x04CgBI"), make([]byte, 40)...)
	p := xmlIPA(t, "com.example.ios", "1", "1", map[string][]byte{"Payload/Demo.app/AppIcon60x60@2x.png": crushed})
	if _, icon, err := ParseIPA(p); err != nil || icon != nil {
		t.Errorf("icon %d err %v", len(icon), err)
	}
}

func TestParseIPAInvalid(t *testing.T) {
	for name, p := range map[string]string{
		"apk":     "testdata/phone-v1.apk",
		"no-ipa":  "testdata/not-an-apk.apk",
		"noplist": writeIPA(t, map[string][]byte{"Payload/Demo.app/Demo": []byte("x")}),
		"noid":    writeIPA(t, map[string][]byte{"Payload/Demo.app/Info.plist": []byte(`<plist><dict><key>A</key><string>b</string></dict></plist>`)}),
	} {
		if _, _, err := ParseIPA(p); err == nil {
			t.Errorf("%s: debería fallar", name)
		}
	}
	if isIPA("testdata/phone-v1.apk") || !isIPA(xmlIPA(t, "a.b", "1", "1", nil)) {
		t.Error("isIPA")
	}
}
