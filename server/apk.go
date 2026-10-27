package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"os"

	"github.com/avast/apkparser"
	"github.com/avast/apkverifier"
)

var ErrInvalidAPK = errors.New("el fichero no es una APK válida")

type Meta struct {
	Package     string `json:"package"`
	Type        string `json:"type"` // "phone" | "watch"
	Label       string `json:"label"`
	VersionCode int64  `json:"versionCode"`
	VersionName string `json:"versionName"`
	MinSDK      int    `json:"minSdk"`
	CertSHA256  string `json:"certSha256"`
	Size        int64  `json:"size"`
}

// Atributos sin namespace en las etiquetas: encoding/xml los casa con cualquier namespace.
type manifestXML struct {
	Package     string `xml:"package,attr"`
	VersionCode int64  `xml:"versionCode,attr"`
	VersionName string `xml:"versionName,attr"`
	UsesSDK     struct {
		Min int `xml:"minSdkVersion,attr"`
	} `xml:"uses-sdk"`
	Features []struct {
		Name string `xml:"name,attr"`
	} `xml:"uses-feature"`
	App struct {
		Label string `xml:"label,attr"`
		Icon  string `xml:"icon,attr"`
	} `xml:"application"`
}

var pngMagic = []byte("\x89PNG")

// ParseAPK lee los metadatos, el icono (PNG, o nil si no hay) y el certificado de firma de una APK.
func ParseAPK(path string) (Meta, []byte, error) {
	invalid := func(err error) (Meta, []byte, error) {
		return Meta{}, nil, fmt.Errorf("%w: %v", ErrInvalidAPK, err)
	}
	st, err := os.Stat(path)
	if err != nil {
		return Meta{}, nil, err
	}
	zr, err := apkparser.OpenZip(path)
	if err != nil {
		return invalid(err)
	}
	defer zr.Close()

	var buf bytes.Buffer
	if _, manifestErr := apkparser.ParseApkWithZip(zr, xml.NewEncoder(&buf)); manifestErr != nil {
		return invalid(manifestErr)
	}
	var mx manifestXML
	if err := xml.Unmarshal(buf.Bytes(), &mx); err != nil || mx.Package == "" {
		return invalid(fmt.Errorf("manifest ilegible: %v", err))
	}

	// Verify comprueba la firma (APK corrupta o manipulada → inválida), no solo la extrae.
	res, err := apkverifier.Verify(path, zr)
	if err != nil || len(res.SignerCerts) == 0 || len(res.SignerCerts[0]) == 0 {
		return invalid(fmt.Errorf("firma no válida: %v", err))
	}
	sum := sha256.Sum256(res.SignerCerts[0][0].Raw)

	m := Meta{
		Package:     mx.Package,
		Type:        "phone",
		Label:       mx.App.Label,
		VersionCode: mx.VersionCode,
		VersionName: mx.VersionName,
		MinSDK:      mx.UsesSDK.Min,
		CertSHA256:  hex.EncodeToString(sum[:]),
		Size:        st.Size(),
	}
	if m.Label == "" {
		m.Label = m.Package
	}
	for _, f := range mx.Features {
		if f.Name == "android.hardware.type.watch" {
			m.Type = "watch"
		}
	}

	// apkparser resuelve @mipmap/... a la ruta del fichero dentro de la APK.
	// Si no es PNG (adaptativo, vector, WebP) se rasteriza; si no se sabe, nil y la web usa el genérico.
	var icon []byte
	if f := zr.File[mx.App.Icon]; f != nil {
		if data, err := f.ReadAll(5 << 20); err == nil {
			if bytes.HasPrefix(data, pngMagic) {
				icon = data
			} else {
				icon = renderIcon(zr, loadResources(zr), mx.App.Icon)
			}
		}
	}
	return m, icon, nil
}
