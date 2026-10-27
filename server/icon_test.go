package main

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"
	"math"
	"testing"
)

func decodeIcon(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("icon no es PNG: %v", err)
	}
	if b := img.Bounds(); b.Dx() != iconSize || b.Dy() != iconSize {
		t.Fatalf("tamaño %v, quiero %dx%d", b, iconSize, iconSize)
	}
	return img
}

func near(t *testing.T, img image.Image, x, y int, r, g, b uint8) {
	t.Helper()
	pr, pg, pb, _ := img.At(x, y).RGBA()
	d := func(got uint32, want uint8) bool { return math.Abs(float64(got>>8)-float64(want)) <= 3 }
	if !d(pr, r) || !d(pg, g) || !d(pb, b) {
		t.Errorf("píxel (%d,%d) = %d,%d,%d; quiero ≈ %d,%d,%d", x, y, pr>>8, pg>>8, pb>>8, r, g, b)
	}
}

func TestAdaptiveIcon(t *testing.T) {
	_, icon, err := ParseAPK("testdata/adaptive-v1.apk")
	if err != nil {
		t.Fatal(err)
	}
	img := decodeIcon(t, icon)
	near(t, img, 2, 2, 0x3D, 0xDC, 0x84)
	near(t, img, 96, 96, 255, 255, 255)
}

func TestBadVectorIcon(t *testing.T) {
	_, icon, err := ParseAPK("testdata/badvector-v1.apk")
	if err != nil {
		t.Fatal(err)
	}
	if icon != nil {
		near(t, decodeIcon(t, icon), 2, 2, 0x3D, 0xDC, 0x84)
	}
}

func TestPathParser(t *testing.T) {
	segs, err := parsePath("M10,10 l5,0 a5,5 0 0 1 0,10 z")
	if err != nil || len(segs) < 4 {
		t.Errorf("segs=%d err=%v", len(segs), err)
	}
	if _, err := parsePath("M10"); err == nil {
		t.Error("M10 debería fallar")
	}
}

func TestDecodeBitmapRejectsHugeImages(t *testing.T) {
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	data := b.Bytes()
	// IHDR empieza en el byte 8: longitud(4) "IHDR"(4) ancho(4) alto(4)...; se declara 60000×60000.
	binary.BigEndian.PutUint32(data[16:], 60000)
	binary.BigEndian.PutUint32(data[20:], 60000)
	binary.BigEndian.PutUint32(data[29:], crc32.ChecksumIEEE(data[12:29]))
	if _, err := decodeBitmap(data); err == nil {
		t.Error("huge image accepted")
	}
	b.Reset()
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 48, 48)))
	if _, err := decodeBitmap(b.Bytes()); err != nil {
		t.Errorf("small image rejected: %v", err)
	}
}
