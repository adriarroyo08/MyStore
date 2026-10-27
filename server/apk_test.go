package main

import (
	"bytes"
	"errors"
	"os"
	"testing"
)

func TestParsePhone(t *testing.T) {
	m, icon, err := ParseAPK("testdata/phone-v1.apk")
	if err != nil {
		t.Fatal(err)
	}
	if m.Package != "com.example.demo" || m.Type != "phone" || m.Label != "Demo" {
		t.Errorf("got %+v", m)
	}
	if m.VersionCode != 1 || m.VersionName != "1.0" || m.MinSDK != 30 {
		t.Errorf("got %+v", m)
	}
	if len(m.CertSHA256) != 64 {
		t.Errorf("cert = %q", m.CertSHA256)
	}
	if !bytes.HasPrefix(icon, []byte("\x89PNG")) {
		t.Errorf("icon is not PNG: %d bytes", len(icon))
	}
	st, _ := os.Stat("testdata/phone-v1.apk")
	if m.Size != st.Size() {
		t.Errorf("size = %d, want %d", m.Size, st.Size())
	}
}

func TestParseWatch(t *testing.T) {
	m, _, err := ParseAPK("testdata/watch-v1.apk")
	if err != nil {
		t.Fatal(err)
	}
	if m.Type != "watch" {
		t.Errorf("type = %q", m.Type)
	}
}

func TestCertDiffers(t *testing.T) {
	v1, _, _ := ParseAPK("testdata/phone-v1.apk")
	v2, _, _ := ParseAPK("testdata/phone-v2.apk")
	other, _, _ := ParseAPK("testdata/phone-v2-otherkey.apk")
	if v1.CertSHA256 != v2.CertSHA256 {
		t.Error("same key should give same cert")
	}
	if v2.CertSHA256 == other.CertSHA256 {
		t.Error("different key should give different cert")
	}
}

func TestNoIcon(t *testing.T) {
	m, icon, err := ParseAPK("testdata/noicon-v1.apk")
	if err != nil {
		t.Fatal(err)
	}
	if icon != nil || m.Package != "com.example.noicon" {
		t.Errorf("icon=%d bytes, meta %+v", len(icon), m)
	}
}

func TestInvalid(t *testing.T) {
	_, _, err := ParseAPK("testdata/not-an-apk.apk")
	if !errors.Is(err, ErrInvalidAPK) {
		t.Errorf("err = %v", err)
	}
}
