package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testToken = "secret"

func newTestServer(t *testing.T, maxUpload int64) (*httptest.Server, *Store) {
	t.Helper()
	s := newTestStore(t)
	srv := httptest.NewServer(NewHandler(s, testToken, maxUpload))
	t.Cleanup(srv.Close)
	return srv, s
}

func do(t *testing.T, method, url, token string, body io.Reader, contentType string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(method, url, body)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func upload(t *testing.T, srv *httptest.Server, token, fixtureName string) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("apk", fixtureName)
	data, _ := os.ReadFile(filepath.Join("testdata", fixtureName))
	fw.Write(data)
	w.WriteField("notes", "notas")
	w.Close()
	return do(t, "POST", srv.URL+"/api/apps", token, &buf, w.FormDataContentType())
}

func listApps(t *testing.T, srv *httptest.Server, query string) []map[string]any {
	t.Helper()
	res := do(t, "GET", srv.URL+"/api/apps"+query, "", nil, "")
	var apps []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&apps); err != nil {
		t.Fatal(err)
	}
	return apps
}

func errorMessage(t *testing.T, res *http.Response) string {
	t.Helper()
	var body struct{ Error string }
	json.NewDecoder(res.Body).Decode(&body)
	return body.Error
}

func TestUploadRequiresToken(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	for _, tok := range []string{"", "wrong"} {
		if res := upload(t, srv, tok, "phone-v1.apk"); res.StatusCode != 401 {
			t.Errorf("token %q: status %d", tok, res.StatusCode)
		}
	}
	if apps := listApps(t, srv, ""); len(apps) != 0 {
		t.Errorf("apps = %v", apps)
	}
}

func TestUploadAndList(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	if res := upload(t, srv, testToken, "phone-v1.apk"); res.StatusCode != 201 {
		t.Fatalf("status %d: %s", res.StatusCode, errorMessage(t, res))
	}
	apps := listApps(t, srv, "?type=phone")
	if len(apps) != 1 || apps[0]["apkUrl"] != "/api/apps/com.example.demo-phone/apk" || apps[0]["notes"] != "notas" {
		t.Errorf("apps = %v", apps)
	}
	if apps := listApps(t, srv, "?type=watch"); len(apps) != 0 {
		t.Errorf("watch apps = %v", apps)
	}
}

func TestUploadErrors(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	if res := upload(t, srv, testToken, "not-an-apk.apk"); res.StatusCode != 400 || errorMessage(t, res) == "" {
		t.Errorf("invalid: status %d", res.StatusCode)
	}
	upload(t, srv, testToken, "phone-v1.apk")
	if res := upload(t, srv, testToken, "phone-v1.apk"); res.StatusCode != 409 || errorMessage(t, res) == "" {
		t.Errorf("not newer: status %d", res.StatusCode)
	}
	if res := upload(t, srv, testToken, "phone-v2-otherkey.apk"); res.StatusCode != 409 {
		t.Errorf("other key: status %d", res.StatusCode)
	}
	res := do(t, "POST", srv.URL+"/api/apps", testToken, strings.NewReader("x"), "text/plain")
	if res.StatusCode != 400 {
		t.Errorf("no multipart: status %d", res.StatusCode)
	}
}

func TestUploadTooBig(t *testing.T) {
	srv, _ := newTestServer(t, 1024)
	if res := upload(t, srv, testToken, "phone-v1.apk"); res.StatusCode != 413 {
		t.Errorf("status %d", res.StatusCode)
	}
}

func TestDownloadAPK(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	upload(t, srv, testToken, "phone-v1.apk")
	res := do(t, "GET", srv.URL+"/api/apps/com.example.demo-phone/apk", "", nil, "")
	got, _ := io.ReadAll(res.Body)
	want, _ := os.ReadFile("testdata/phone-v1.apk")
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "application/vnd.android.package-archive" || !bytes.Equal(got, want) {
		t.Errorf("status %d, type %q, %d bytes", res.StatusCode, res.Header.Get("Content-Type"), len(got))
	}
	if cd := res.Header.Get("Content-Disposition"); !strings.Contains(cd, "com.example.demo-1.0.apk") {
		t.Errorf("Content-Disposition %q", cd)
	}
}

func TestIcons(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	upload(t, srv, testToken, "noicon-v1.apk")
	upload(t, srv, testToken, "phone-v1.apk")
	for _, id := range []string{"com.example.noicon-phone", "com.example.demo-phone"} {
		res := do(t, "GET", srv.URL+"/api/apps/"+id+"/icon", "", nil, "")
		body, _ := io.ReadAll(res.Body)
		if res.StatusCode != 200 || !bytes.HasPrefix(body, pngMagic) {
			t.Errorf("%s: status %d", id, res.StatusCode)
		}
	}
}

func TestPatchAndDelete(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	upload(t, srv, testToken, "phone-v1.apk")
	app := srv.URL + "/api/apps/com.example.demo-phone"
	if res := do(t, "PATCH", app, "", strings.NewReader(`{"description":"x"}`), "application/json"); res.StatusCode != 401 {
		t.Errorf("patch no token: %d", res.StatusCode)
	}
	if res := do(t, "PATCH", app, testToken, strings.NewReader(`{"description":"hola"}`), "application/json"); res.StatusCode != 200 {
		t.Errorf("patch: %d", res.StatusCode)
	}
	if apps := listApps(t, srv, ""); apps[0]["description"] != "hola" || apps[0]["notes"] != "notas" {
		t.Errorf("apps = %v", apps)
	}
	if res := do(t, "DELETE", app, "", nil, ""); res.StatusCode != 401 {
		t.Errorf("delete no token: %d", res.StatusCode)
	}
	if res := do(t, "DELETE", app, testToken, nil, ""); res.StatusCode != 204 {
		t.Errorf("delete: %d", res.StatusCode)
	}
	if res := do(t, "GET", app+"/apk", "", nil, ""); res.StatusCode != 404 {
		t.Errorf("after delete: %d", res.StatusCode)
	}
}

func TestTraversal(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	for _, p := range []string{"/api/apps/..%2F..%2Fetc-phone/apk", "/api/apps/..%5C..%5Cetc-phone/icon"} {
		if res := do(t, "GET", srv.URL+p, "", nil, ""); res.StatusCode != 404 {
			t.Errorf("%s: %d", p, res.StatusCode)
		}
	}
}
