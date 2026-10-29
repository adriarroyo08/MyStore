package main

import (
	"os"
	"testing"
)

func TestParsePlistBinary(t *testing.T) {
	data, _ := os.ReadFile("testdata/info.bplist")
	v, err := parsePlist(data)
	if err != nil {
		t.Fatal(err)
	}
	d := v.(map[string]any)
	if d["CFBundleIdentifier"] != "com.example.ios" || d["CFBundleDisplayName"] != "Demo iOS ñ" {
		t.Errorf("strings: %v", d)
	}
	if d["Big"] != int64(70000) || d["Neg"] != int64(-5) || d["Flag"] != true {
		t.Errorf("scalars: %v", d)
	}
	l := d["List"].([]any)
	if len(l) != 2 || l[1] != "b" {
		t.Errorf("list: %v", l)
	}
	files := d["CFBundleIcons"].(map[string]any)["CFBundlePrimaryIcon"].(map[string]any)["CFBundleIconFiles"].([]any)
	if files[0] != "AppIcon60x60" {
		t.Errorf("nested: %v", files)
	}
}

func TestParsePlistXML(t *testing.T) {
	v, err := parsePlist([]byte(xmlPlist("com.example.x", "2.0", "7")))
	if err != nil {
		t.Fatal(err)
	}
	d := v.(map[string]any)
	if d["CFBundleIdentifier"] != "com.example.x" || d["CFBundleVersion"] != "7" || d["N"] != int64(42) || d["B"] != true {
		t.Errorf("got %v", d)
	}
	if l := d["L"].([]any); len(l) != 2 || l[0] != "a&b" {
		t.Errorf("list: %v", l)
	}
}

func TestParsePlistGarbage(t *testing.T) {
	for _, s := range []string{"", "nope", "bplist00", "bplist00\x00\x01"} {
		if _, err := parsePlist([]byte(s)); err == nil {
			t.Errorf("%q should fail", s)
		}
	}
}

func xmlPlist(id, short, build string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>` + id + `</string>
<key>CFBundleShortVersionString</key><string>` + short + `</string>
<key>CFBundleVersion</key><string>` + build + `</string>
<key>CFBundleName</key><string>Demo X</string>
<key>MinimumOSVersion</key><string>16.0</string>
<key>N</key><integer>42</integer><key>B</key><true/><key>R</key><real>1.5</real>
<key>L</key><array><string>a&amp;b</string><string>c</string></array>
</dict></plist>`
}
