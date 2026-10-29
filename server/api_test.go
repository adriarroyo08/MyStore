package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestServesAdminPage(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	res := do(t, "GET", srv.URL+"/", "", nil, "")
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/html") || !bytes.Contains(body, []byte("MyStore")) {
		t.Errorf("status %d, type %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
	if res := do(t, "GET", srv.URL+"/nope", "", nil, ""); res.StatusCode != 404 {
		t.Errorf("unknown path: %d", res.StatusCode)
	}
}

func pngBytes(w, h int) []byte {
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, w, h)))
	return b.Bytes()
}

func postShot(t *testing.T, srv *httptest.Server, token string, data []byte) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("image", "shot.png")
	fw.Write(data)
	w.Close()
	return do(t, "POST", srv.URL+"/api/apps/com.example.demo-phone/screenshots", token, &buf, w.FormDataContentType())
}

func shotURLs(t *testing.T, res *http.Response) []any {
	t.Helper()
	var a map[string]any
	if err := json.NewDecoder(res.Body).Decode(&a); err != nil {
		t.Fatal(err)
	}
	return a["screenshots"].([]any)
}

func TestScreenshots(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	upload(t, srv, testToken, "phone-v1.apk")
	if apps := listApps(t, srv, ""); len(apps[0]["screenshots"].([]any)) != 0 {
		t.Errorf("screenshots = %v", apps[0]["screenshots"])
	}
	if res := postShot(t, srv, "", pngBytes(8, 8)); res.StatusCode != 401 {
		t.Errorf("no token: %d", res.StatusCode)
	}
	if res := postShot(t, srv, testToken, pngBytes(8, 8)); res.StatusCode != 201 {
		t.Fatalf("post: %d", res.StatusCode)
	}
	res := postShot(t, srv, testToken, pngBytes(16, 16))
	urls := shotURLs(t, res)
	if res.StatusCode != 201 || len(urls) != 2 || urls[1] != "/api/apps/com.example.demo-phone/screenshots/2" {
		t.Fatalf("urls = %v", urls)
	}
	get := do(t, "GET", srv.URL+urls[0].(string), "", nil, "")
	if body, _ := io.ReadAll(get.Body); get.StatusCode != 200 || get.Header.Get("Content-Type") != "image/png" || !bytes.Equal(body, pngBytes(8, 8)) {
		t.Errorf("get: %d %q", get.StatusCode, get.Header.Get("Content-Type"))
	}
	base := srv.URL + "/api/apps/com.example.demo-phone/screenshots/"
	if res := do(t, "DELETE", base+"1", "", nil, ""); res.StatusCode != 401 {
		t.Errorf("delete no token: %d", res.StatusCode)
	}
	res = do(t, "DELETE", base+"1", testToken, nil, "")
	if urls := shotURLs(t, res); res.StatusCode != 200 || len(urls) != 1 {
		t.Fatalf("delete: %d %v", res.StatusCode, urls)
	}
	// la que era la 2 ahora es la 1
	get = do(t, "GET", base+"1", "", nil, "")
	if body, _ := io.ReadAll(get.Body); !bytes.Equal(body, pngBytes(16, 16)) {
		t.Error("renumbering lost the second image")
	}
	for _, n := range []string{"0", "2", "-1", "x", "01", "1.png", "..%2Fmeta.json", "999999999999999999999"} {
		if res := do(t, "GET", base+n, "", nil, ""); res.StatusCode != 404 {
			t.Errorf("GET %s: %d", n, res.StatusCode)
		}
		if res := do(t, "DELETE", base+n, testToken, nil, ""); res.StatusCode != 404 {
			t.Errorf("DELETE %s: %d", n, res.StatusCode)
		}
	}
}

func TestScreenshotErrors(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	upload(t, srv, testToken, "phone-v1.apk")
	for name, data := range map[string][]byte{"text": []byte("hola"), "huge": pngBytes(4097, 1), "empty": nil} {
		if res := postShot(t, srv, testToken, data); res.StatusCode != 400 || errorMessage(t, res) == "" {
			t.Errorf("%s: %d", name, res.StatusCode)
		}
	}
	if res := postShot(t, srv, testToken, append(pngBytes(8, 8), make([]byte, 5<<20)...)); res.StatusCode != 413 {
		t.Errorf("too big: %d", res.StatusCode)
	}
	for i := 0; i < 8; i++ {
		if res := postShot(t, srv, testToken, pngBytes(8, 8)); res.StatusCode != 201 {
			t.Fatalf("shot %d: %d", i, res.StatusCode)
		}
	}
	if res := postShot(t, srv, testToken, pngBytes(8, 8)); res.StatusCode != 409 {
		t.Errorf("ninth: %d", res.StatusCode)
	}
	res := do(t, "POST", srv.URL+"/api/apps/com.example.nope-phone/screenshots", testToken, strings.NewReader("x"), "text/plain")
	if res.StatusCode != 400 && res.StatusCode != 404 {
		t.Errorf("unknown app: %d", res.StatusCode)
	}
}

func TestScreenshotsSurviveUpgradeAndDelete(t *testing.T) {
	srv, s := newTestServer(t, 1<<20)
	upload(t, srv, testToken, "phone-v1.apk")
	postShot(t, srv, testToken, pngBytes(8, 8))
	if res := upload(t, srv, testToken, "phone-v2.apk"); res.StatusCode != 201 {
		t.Fatalf("v2: %d", res.StatusCode)
	}
	apps := listApps(t, srv, "")
	if len(apps[0]["screenshots"].([]any)) != 1 {
		t.Fatalf("screenshots = %v", apps[0]["screenshots"])
	}
	get := do(t, "GET", srv.URL+"/api/apps/com.example.demo-phone/screenshots/1", "", nil, "")
	if body, _ := io.ReadAll(get.Body); get.StatusCode != 200 || !bytes.Equal(body, pngBytes(8, 8)) {
		t.Errorf("after upgrade: %d", get.StatusCode)
	}
	do(t, "DELETE", srv.URL+"/api/apps/com.example.demo-phone", testToken, nil, "")
	if _, err := os.Stat(filepath.Join(s.apps, "com.example.demo-phone")); err == nil {
		t.Error("app dir survived delete")
	}
}

func putIcon(t *testing.T, srv *httptest.Server, token, id string, data []byte) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("image", "icon.png")
	fw.Write(data)
	w.Close()
	return do(t, "PUT", srv.URL+"/api/apps/"+id+"/icon", token, &buf, w.FormDataContentType())
}

func TestCustomIcon(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	const id = "com.example.noicon-phone"
	upload(t, srv, testToken, "noicon-v1.apk")
	if res := putIcon(t, srv, "", id, pngBytes(8, 8)); res.StatusCode != 401 {
		t.Errorf("sin token: %d", res.StatusCode)
	}
	res := putIcon(t, srv, testToken, id, pngBytes(300, 200))
	var a map[string]any
	json.NewDecoder(res.Body).Decode(&a)
	if res.StatusCode != 200 || a["hasIcon"] != true {
		t.Fatalf("put: %d %v", res.StatusCode, a)
	}
	get := do(t, "GET", srv.URL+"/api/apps/"+id+"/icon", "", nil, "")
	body, _ := io.ReadAll(get.Body)
	cfg, err := png.DecodeConfig(bytes.NewReader(body))
	if err != nil || cfg.Width != iconSize || cfg.Height != iconSize {
		t.Fatalf("icono servido: %v %+v", err, cfg)
	}
	if bytes.Equal(body, genericIcon) {
		t.Error("sirve el icono genérico")
	}
	// una APK nueva sin icono conserva el personalizado
	if res := upload(t, srv, testToken, "noicon-v2.apk"); res.StatusCode != 201 {
		t.Fatalf("v2: %d", res.StatusCode)
	}
	if apps := listApps(t, srv, ""); apps[0]["hasIcon"] != true {
		t.Error("hasIcon perdido tras subir v2")
	}
	get = do(t, "GET", srv.URL+"/api/apps/"+id+"/icon", "", nil, "")
	if again, _ := io.ReadAll(get.Body); !bytes.Equal(again, body) {
		t.Error("icono distinto tras subir v2")
	}
}

func TestCustomIconErrors(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	upload(t, srv, testToken, "noicon-v1.apk")
	const id = "com.example.noicon-phone"
	for name, data := range map[string][]byte{"basura": []byte("nope"), "enorme": pngBytes(4097, 1)} {
		if res := putIcon(t, srv, testToken, id, data); res.StatusCode != 400 || errorMessage(t, res) == "" {
			t.Errorf("%s: %d", name, res.StatusCode)
		}
	}
	if res := putIcon(t, srv, testToken, id, append(pngBytes(8, 8), make([]byte, 2<<20)...)); res.StatusCode != 413 {
		t.Errorf("grande: %d", res.StatusCode)
	}
	if res := putIcon(t, srv, testToken, "com.example.nope-phone", pngBytes(8, 8)); res.StatusCode != 404 {
		t.Errorf("desconocida: %d", res.StatusCode)
	}
}

func TestApkIconWinsOverCustom(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	const id = "com.example.demo-phone"
	upload(t, srv, testToken, "phone-v1.apk")
	putIcon(t, srv, testToken, id, pngBytes(8, 8))
	upload(t, srv, testToken, "phone-v2.apk")
	apps := listApps(t, srv, "")
	if apps[0]["customIcon"] == true {
		t.Error("customIcon no se limpió")
	}
}

func TestVersionHistory(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	upload(t, srv, testToken, "phone-v1.apk")
	if res := upload(t, srv, testToken, "phone-v2.apk"); res.StatusCode != 201 {
		t.Fatalf("v2: %d", res.StatusCode)
	}
	apps := listApps(t, srv, "")
	vs, _ := apps[0]["versions"].([]any)
	if len(vs) != 1 {
		t.Fatalf("versions = %v", apps[0]["versions"])
	}
	v := vs[0].(map[string]any)
	if v["versionCode"] != float64(1) || v["apkUrl"] != "/api/apps/com.example.demo-phone/versions/1/apk" || v["notes"] != "notas" {
		t.Errorf("version = %v", v)
	}
	res := do(t, "GET", srv.URL+"/api/apps/com.example.demo-phone/versions/1/apk", "", nil, "")
	got, _ := io.ReadAll(res.Body)
	want, _ := os.ReadFile("testdata/phone-v1.apk")
	if res.StatusCode != 200 || !bytes.Equal(got, want) {
		t.Errorf("old apk: status %d, %d bytes", res.StatusCode, len(got))
	}
	for _, p := range []string{"versions/2/apk", "versions/9/apk", "versions/x/apk"} {
		if res := do(t, "GET", srv.URL+"/api/apps/com.example.demo-phone/"+p, "", nil, ""); res.StatusCode != 404 {
			t.Errorf("%s: %d", p, res.StatusCode)
		}
	}
}

func TestVersionHistoryIsCapped(t *testing.T) {
	defer func(n int) { maxVersions = n }(maxVersions)
	maxVersions = 0
	srv, s := newTestServer(t, 1<<20)
	upload(t, srv, testToken, "phone-v1.apk")
	upload(t, srv, testToken, "phone-v2.apk")
	if a, _ := s.Get("com.example.demo-phone"); len(a.Versions) != 0 {
		t.Errorf("versions = %v", a.Versions)
	}
	if _, err := os.Stat(filepath.Join(s.apps, "com.example.demo-phone", "versions", "1.apk")); err == nil {
		t.Error("old apk kept beyond the cap")
	}
}

func TestCategory(t *testing.T) {
	srv, _ := newTestServer(t, 1<<20)
	upload(t, srv, testToken, "phone-v1.apk")
	app := srv.URL + "/api/apps/com.example.demo-phone"
	if res := do(t, "PATCH", app, "", strings.NewReader(`{"category":"Juegos"}`), "application/json"); res.StatusCode != 401 {
		t.Errorf("patch no token: %d", res.StatusCode)
	}
	if res := do(t, "PATCH", app, testToken, strings.NewReader(`{"category":"  Juegos "}`), "application/json"); res.StatusCode != 200 {
		t.Fatalf("patch: %d", res.StatusCode)
	}
	if c := listApps(t, srv, "")[0]["category"]; c != "Juegos" {
		t.Errorf("category = %v", c)
	}
	upload(t, srv, testToken, "phone-v2.apk") // se conserva en nuevas subidas
	if c := listApps(t, srv, "")[0]["category"]; c != "Juegos" {
		t.Errorf("after upload category = %v", c)
	}
	do(t, "PATCH", app, testToken, strings.NewReader(`{"category":"`+strings.Repeat("ñ", 60)+`"}`), "application/json")
	if c := listApps(t, srv, "")[0]["category"].(string); len([]rune(c)) != 40 {
		t.Errorf("category not capped: %d", len([]rune(c)))
	}
	do(t, "PATCH", app, testToken, strings.NewReader(`{"category":""}`), "application/json")
	if c := listApps(t, srv, "")[0]["category"]; c != "" {
		t.Errorf("category not cleared: %v", c)
	}
}

func TestDeleteVersion(t *testing.T) {
	srv, s := newTestServer(t, 1<<20)
	upload(t, srv, testToken, "phone-v1.apk")
	upload(t, srv, testToken, "phone-v2.apk")
	u := srv.URL + "/api/apps/com.example.demo-phone/versions/"
	if res := do(t, "DELETE", u+"1", "", nil, ""); res.StatusCode != 401 {
		t.Errorf("no token: %d", res.StatusCode)
	}
	for _, c := range []string{"9", "x", "2"} {
		if res := do(t, "DELETE", u+c, testToken, nil, ""); res.StatusCode != 404 {
			t.Errorf("%s: %d", c, res.StatusCode)
		}
	}
	if res := do(t, "DELETE", u+"1", testToken, nil, ""); res.StatusCode != 204 {
		t.Fatalf("delete: %d", res.StatusCode)
	}
	if a, _ := s.Get("com.example.demo-phone"); len(a.Versions) != 0 {
		t.Errorf("versions = %v", a.Versions)
	}
	if _, err := os.Stat(filepath.Join(s.apps, "com.example.demo-phone", "versions", "1.apk")); err == nil {
		t.Error("apk kept")
	}
	if res := do(t, "DELETE", u+"1", testToken, nil, ""); res.StatusCode != 404 {
		t.Errorf("twice: %d", res.StatusCode)
	}
}

func TestDevicesAndStats(t *testing.T) {
	srv, s := newTestServer(t, 1<<20)
	upload(t, srv, testToken, "phone-v1.apk")
	get := func(device string) {
		req, _ := http.NewRequest("GET", srv.URL+"/api/apps", nil)
		if device != "" {
			req.Header.Set("X-MyStore-Device", device)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
	}
	get("")
	get("POCO X6 · phone · 1.4")
	get("POCO X6 · phone · 1.4")
	get(strings.Repeat("d", 300))
	if res := do(t, "GET", srv.URL+"/api/stats", "", nil, ""); res.StatusCode != 401 {
		t.Errorf("stats no token: %d", res.StatusCode)
	}
	var st struct {
		Apps      int
		DiskBytes int64
		Devices   []struct {
			Device   string
			LastSeen string
		}
	}
	res := do(t, "GET", srv.URL+"/api/stats", testToken, nil, "")
	if err := json.NewDecoder(res.Body).Decode(&st); err != nil {
		t.Fatal(err)
	}
	if st.Apps != 1 || st.DiskBytes <= 0 || len(st.Devices) != 2 {
		t.Fatalf("stats = %+v", st)
	}
	if len(st.Devices[0].Device) != 100 || st.Devices[1].Device != "POCO X6 · phone · 1.4" {
		t.Errorf("devices = %+v", st.Devices) // más nuevo primero, con tope de 100
	}
	if _, err := time.Parse(time.RFC3339, st.Devices[0].LastSeen); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(s.apps), "devices.json")); err != nil {
		t.Error("devices.json no escrito")
	}
	s2, _ := NewStore(filepath.Dir(s.apps)) // persiste entre reinicios
	if len(s2.Devices()) != 2 {
		t.Errorf("reloaded = %v", s2.Devices())
	}
}
