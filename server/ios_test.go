package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func uploadIPA(t *testing.T, srv *httptest.Server, path, notes string) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("apk", "app.ipa")
	data, _ := os.ReadFile(path)
	fw.Write(data)
	w.WriteField("notes", notes)
	w.Close()
	return do(t, "POST", srv.URL+"/api/apps", testToken, &buf, w.FormDataContentType())
}

func getWith(t *testing.T, url string, hdr map[string]string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func TestUploadIOS(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	res := uploadIPA(t, srv, xmlIPA(t, "com.example.ios", "1.2", "5", nil), "primera")
	if res.StatusCode != 201 {
		t.Fatalf("status %d: %s", res.StatusCode, errorMessage(t, res))
	}
	var a map[string]any
	json.NewDecoder(res.Body).Decode(&a)
	if a["id"] != "com.example.ios-ios" || a["type"] != "ios" || a["label"] != "Demo X" || a["versionName"] != "1.2" ||
		a["versionCode"] != float64(5) || a["minOS"] != "16.0" || a["notes"] != "primera" {
		t.Errorf("got %v", a)
	}
	if n := len(listApps(t, srv, "?type=ios")); n != 1 {
		t.Errorf("ios: %d", n)
	}
	if n := len(listApps(t, srv, "?type=phone")); n != 0 {
		t.Errorf("phone: %d", n)
	}
	dl := do(t, "GET", srv.URL+"/api/apps/com.example.ios-ios/apk", "", nil, "")
	if dl.Header.Get("Content-Type") != "application/octet-stream" || !strings.Contains(dl.Header.Get("Content-Disposition"), "-1.2.ipa") {
		t.Errorf("headers %v", dl.Header)
	}
}

func TestIOSVersions(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	uploadIPA(t, srv, xmlIPA(t, "com.example.ios", "1.0", "5", nil), "")
	if res := uploadIPA(t, srv, xmlIPA(t, "com.example.ios", "1.0", "5", nil), ""); res.StatusCode != 409 {
		t.Errorf("repetida: %d", res.StatusCode)
	}
	if res := uploadIPA(t, srv, xmlIPA(t, "com.example.ios", "0.9", "4", nil), ""); res.StatusCode != 409 {
		t.Errorf("anterior: %d", res.StatusCode)
	}
	if res := uploadIPA(t, srv, xmlIPA(t, "com.example.ios", "1.1", "6", nil), "arreglos"); res.StatusCode != 201 {
		t.Fatalf("nueva: %d %s", res.StatusCode, errorMessage(t, res))
	}
	vs := listApps(t, srv, "?type=ios")[0]["versions"].([]any)
	if len(vs) != 1 || vs[0].(map[string]any)["versionCode"] != float64(5) {
		t.Errorf("versions %v", vs)
	}
	old := do(t, "GET", srv.URL+"/api/apps/com.example.ios-ios/versions/5/apk", "", nil, "")
	if !strings.Contains(old.Header.Get("Content-Disposition"), "-1.0.ipa") {
		t.Errorf("old: %v", old.Header)
	}
}

func TestManifestPlist(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	uploadIPA(t, srv, xmlIPA(t, "com.example.ios", "1.2", "5", nil), "")
	res := getWith(t, srv.URL+"/api/apps/com.example.ios-ios/manifest.plist", map[string]string{"X-Forwarded-Proto": "https", "X-Forwarded-Host": "tienda.example.org"})
	if res.StatusCode != 200 || !strings.Contains(res.Header.Get("Content-Type"), "xml") {
		t.Fatalf("status %d %v", res.StatusCode, res.Header)
	}
	body, _ := io.ReadAll(res.Body)
	if _, err := parsePlist(body); err != nil {
		t.Fatalf("no es un plist: %v\n%s", err, body)
	}
	for _, want := range []string{
		"<string>https://tienda.example.org/api/apps/com.example.ios-ios/apk</string>",
		"<string>https://tienda.example.org/api/apps/com.example.ios-ios/icon</string>",
		"<key>bundle-identifier</key>", "<string>com.example.ios</string>",
		"<key>bundle-version</key>", "<string>1.2</string>",
		"<string>software-package</string>", "<string>display-image</string>", "<string>full-size-image</string>",
		"<key>title</key>", "<string>Demo X</string>",
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("falta %s\n%s", want, body)
		}
	}
	// sin cabeceras: usa el host de la petición
	res = getWith(t, srv.URL+"/api/apps/com.example.ios-ios/manifest.plist", nil)
	body, _ = io.ReadAll(res.Body)
	if !strings.Contains(string(body), "<string>"+srv.URL+"/api/apps/com.example.ios-ios/apk</string>") {
		t.Errorf("sin proxy:\n%s", body)
	}
}

func TestManifestPlistNotIOS(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	upload(t, srv, testToken, "phone-v1.apk")
	for _, id := range []string{"com.example.demo-phone", "nada.nada-ios"} {
		if res := do(t, "GET", srv.URL+"/api/apps/"+id+"/manifest.plist", "", nil, ""); res.StatusCode != 404 {
			t.Errorf("%s: %d", id, res.StatusCode)
		}
	}
}

func TestAltStoreSource(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	upload(t, srv, testToken, "phone-v1.apk")
	uploadIPA(t, srv, xmlIPA(t, "com.example.ios", "1.0", "5", nil), "uno")
	uploadIPA(t, srv, xmlIPA(t, "com.example.ios", "1.1", "6", nil), "dos")
	res := getWith(t, srv.URL+"/api/altstore.json", map[string]string{"X-Forwarded-Proto": "https", "X-Forwarded-Host": "tienda.automatito.win"})
	if res.StatusCode != 200 || !strings.Contains(res.Header.Get("Content-Type"), "json") {
		t.Fatalf("status %d %v", res.StatusCode, res.Header)
	}
	var src struct {
		Name, Identifier, SourceURL string
		Apps                        []struct {
			Name, BundleIdentifier, IconURL string
			Versions                        []struct {
				Version, BuildVersion, Date, LocalizedDescription, DownloadURL, MinOSVersion string
				Size                                                                          int64
			}
		}
	}
	if err := json.NewDecoder(res.Body).Decode(&src); err != nil {
		t.Fatal(err)
	}
	if src.Name != "MyStore" || src.Identifier != "win.automatito.tienda" || src.SourceURL != "https://tienda.automatito.win/api/altstore.json" {
		t.Errorf("cabecera %+v", src)
	}
	if len(src.Apps) != 1 || src.Apps[0].BundleIdentifier != "com.example.ios" || src.Apps[0].Name != "Demo X" ||
		src.Apps[0].IconURL != "https://tienda.automatito.win/api/apps/com.example.ios-ios/icon" {
		t.Fatalf("apps %+v", src.Apps)
	}
	vs := src.Apps[0].Versions
	if len(vs) != 2 || vs[0].Version != "1.1" || vs[1].Version != "1.0" || vs[0].BuildVersion != "6" ||
		vs[0].LocalizedDescription != "dos" || vs[0].MinOSVersion != "16.0" || vs[0].Size == 0 || vs[0].Date == "" ||
		vs[0].DownloadURL != "https://tienda.automatito.win/api/apps/com.example.ios-ios/apk" ||
		vs[1].DownloadURL != "https://tienda.automatito.win/api/apps/com.example.ios-ios/versions/5/apk" {
		t.Errorf("versions %+v", vs)
	}
}
