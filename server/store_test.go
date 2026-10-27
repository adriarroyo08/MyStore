package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// fixture copia una APK de testdata a un temporal, porque Put mueve el fichero.
func fixture(t *testing.T, name string) string {
	t.Helper()
	src, err := os.Open(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	dst, err := os.CreateTemp(t.TempDir(), "*.apk")
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		t.Fatal(err)
	}
	return dst.Name()
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func mustPut(t *testing.T, s *Store, name, notes, desc string) App {
	t.Helper()
	a, err := s.Put(fixture(t, name), notes, desc)
	if err != nil {
		t.Fatalf("Put %s: %v", name, err)
	}
	return a
}

func TestPutAndList(t *testing.T) {
	s := newTestStore(t)
	a := mustPut(t, s, "phone-v1.apk", "", "")
	if a.ID != "com.example.demo-phone" || !a.HasIcon {
		t.Errorf("got %+v", a)
	}
	if l, _ := s.List("phone"); len(l) != 1 {
		t.Errorf("phone list = %d", len(l))
	}
	if l, _ := s.List("watch"); len(l) != 0 {
		t.Errorf("watch list = %d", len(l))
	}
}

func TestPhoneAndWatchCoexist(t *testing.T) {
	s := newTestStore(t)
	p := mustPut(t, s, "phone-v1.apk", "", "")
	w := mustPut(t, s, "watch-v1.apk", "", "")
	if l, _ := s.List(""); len(l) != 2 || p.ID == w.ID {
		t.Errorf("list = %d, ids %s %s", len(l), p.ID, w.ID)
	}
}

func TestRejectNotNewer(t *testing.T) {
	s := newTestStore(t)
	mustPut(t, s, "phone-v2.apk", "", "")
	if _, err := s.Put(fixture(t, "phone-v1.apk"), "", ""); !errors.Is(err, ErrNotNewer) {
		t.Errorf("err = %v", err)
	}
	if a, _ := s.Get("com.example.demo-phone"); a.VersionCode != 2 {
		t.Errorf("version = %d", a.VersionCode)
	}
}

func TestRejectOtherKey(t *testing.T) {
	s := newTestStore(t)
	mustPut(t, s, "phone-v1.apk", "", "")
	if _, err := s.Put(fixture(t, "phone-v2-otherkey.apk"), "", ""); !errors.Is(err, ErrCertMismatch) {
		t.Errorf("err = %v", err)
	}
	if a, _ := s.Get("com.example.demo-phone"); a.VersionCode != 1 {
		t.Errorf("version = %d", a.VersionCode)
	}
}

func TestUpgradeKeepsDescription(t *testing.T) {
	s := newTestStore(t)
	mustPut(t, s, "phone-v1.apk", "n1", "d")
	a := mustPut(t, s, "phone-v2.apk", "n2", "")
	if a.Description != "d" || a.Notes != "n2" {
		t.Errorf("got desc %q notes %q", a.Description, a.Notes)
	}
}

func TestUpdate(t *testing.T) {
	s := newTestStore(t)
	mustPut(t, s, "phone-v1.apk", "n", "d")
	desc := "nueva"
	a, err := s.Update("com.example.demo-phone", &desc, nil)
	if err != nil || a.Description != "nueva" || a.Notes != "n" {
		t.Errorf("got %+v, %v", a, err)
	}
	if a, _ := s.Get("com.example.demo-phone"); a.Description != "nueva" {
		t.Errorf("not persisted: %q", a.Description)
	}
}

func TestDeleteAllowsNewKey(t *testing.T) {
	s := newTestStore(t)
	mustPut(t, s, "phone-v1.apk", "", "")
	if err := s.Delete("com.example.demo-phone"); err != nil {
		t.Fatal(err)
	}
	mustPut(t, s, "phone-v2-otherkey.apk", "", "")
}

func TestBadIDs(t *testing.T) {
	s := newTestStore(t)
	mustPut(t, s, "phone-v1.apk", "", "")
	for _, id := range []string{"../x-phone", "a/b-phone", `a\b-phone`, "com.example.demo", "", "..-phone"} {
		if _, err := s.Get(id); !errors.Is(err, ErrNotFound) {
			t.Errorf("Get(%q) = %v", id, err)
		}
		if err := s.Delete(id); !errors.Is(err, ErrNotFound) {
			t.Errorf("Delete(%q) = %v", id, err)
		}
		if _, err := s.APKPath(id); !errors.Is(err, ErrNotFound) {
			t.Errorf("APKPath(%q) = %v", id, err)
		}
	}
}

func TestFailedPutKeepsOld(t *testing.T) {
	s := newTestStore(t)
	mustPut(t, s, "phone-v1.apk", "", "")
	if _, err := s.Put(fixture(t, "not-an-apk.apk"), "", ""); !errors.Is(err, ErrInvalidAPK) {
		t.Errorf("err = %v", err)
	}
	p, err := s.APKPath("com.example.demo-phone")
	if err != nil {
		t.Fatal(err)
	}
	if m, _, err := ParseAPK(p); err != nil || m.VersionCode != 1 {
		t.Errorf("stored apk broken: %+v %v", m, err)
	}
}

func TestConcurrentPuts(t *testing.T) {
	s := newTestStore(t)
	a, b := fixture(t, "phone-v2.apk"), fixture(t, "phone-v2.apk")
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, f := range []string{a, b} {
		wg.Add(1)
		go func() { defer wg.Done(); _, errs[i] = s.Put(f, "", "") }()
	}
	wg.Wait()
	ok, notNewer := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrNotNewer):
			notNewer++
		}
	}
	if ok != 1 || notNewer != 1 {
		t.Errorf("errs = %v", errs)
	}
	if got, err := s.Get("com.example.demo-phone"); err != nil || got.VersionCode != 2 {
		t.Errorf("got %+v %v", got, err)
	}
}

func TestIconPath(t *testing.T) {
	s := newTestStore(t)
	mustPut(t, s, "noicon-v1.apk", "", "")
	if _, ok, err := s.IconPath("com.example.noicon-phone"); err != nil || ok {
		t.Errorf("ok=%v err=%v", ok, err)
	}
}

func TestRecoversInterruptedReplace(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewStore(dir)
	mustPut(t, s, "phone-v1.apk", "", "")
	// Simula un corte entre los dos renombrados de replaceDir.
	apps := filepath.Join(dir, "apps")
	if err := os.Rename(filepath.Join(apps, "com.example.demo-phone"), filepath.Join(apps, ".old-com.example.demo-phone")); err != nil {
		t.Fatal(err)
	}
	s, _ = NewStore(dir)
	if a, err := s.Get("com.example.demo-phone"); err != nil || a.VersionCode != 1 {
		t.Errorf("app lost after restart: %+v %v", a, err)
	}
}
