package main

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type appJSON struct {
	App
	IconURL string `json:"iconUrl"`
	APKURL  string `json:"apkUrl"`
}

func withURLs(a App) appJSON {
	return appJSON{App: a, IconURL: "/api/apps/" + a.ID + "/icon", APKURL: "/api/apps/" + a.ID + "/apk"}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// storeError traduce los errores del almacén a respuestas HTTP.
func storeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrInvalidAPK):
		writeError(w, http.StatusBadRequest, ErrInvalidAPK.Error())
	case errors.Is(err, ErrNotNewer), errors.Is(err, ErrCertMismatch):
		writeError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("error: %v", err)
		writeError(w, http.StatusInternalServerError, "error interno del servidor")
	}
}

var genericIcon = func() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	for i := range img.Pix {
		img.Pix[i] = []uint8{0x9a, 0xa0, 0xa6, 0xff}[i%4]
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	return b.Bytes()
}()

func NewHandler(s *Store, token string, maxUploadBytes int64) http.Handler {
	auth := func(next http.HandlerFunc) http.HandlerFunc {
		want := []byte("Bearer " + token)
		return func(w http.ResponseWriter, r *http.Request) {
			if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), want) != 1 {
				writeError(w, http.StatusUnauthorized, "token incorrecto o ausente")
				return
			}
			log.Printf("%s %s", r.Method, r.URL.Path)
			next(w, r)
		}
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/apps", func(w http.ResponseWriter, r *http.Request) {
		apps, err := s.List(r.URL.Query().Get("type"))
		if err != nil {
			storeError(w, err)
			return
		}
		out := make([]appJSON, len(apps))
		for i, a := range apps {
			out[i] = withURLs(a)
		}
		writeJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("GET /api/apps/{id}/icon", func(w http.ResponseWriter, r *http.Request) {
		path, ok, err := s.IconPath(r.PathValue("id"))
		if err != nil {
			storeError(w, err)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		if !ok {
			w.Write(genericIcon)
			return
		}
		http.ServeFile(w, r, path)
	})

	mux.HandleFunc("GET /api/apps/{id}/apk", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		a, err := s.Get(id)
		if err != nil {
			storeError(w, err)
			return
		}
		path, err := s.APKPath(id)
		if err != nil {
			storeError(w, err)
			return
		}
		f, err := os.Open(path)
		if err != nil {
			storeError(w, err)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "application/vnd.android.package-archive")
		w.Header().Set("Content-Disposition", `attachment; filename="`+a.Package+"-"+a.VersionName+`.apk"`)
		http.ServeContent(w, r, "", a.UpdatedAt, f)
	})

	mux.HandleFunc("POST /api/apps", auth(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
		mr, err := r.MultipartReader()
		if err != nil {
			writeError(w, http.StatusBadRequest, "se esperaba un formulario multipart con el campo apk")
			return
		}
		tmp, err := os.CreateTemp(s.apps, ".upload-*.apk")
		if err != nil {
			storeError(w, err)
			return
		}
		defer os.Remove(tmp.Name()) // no-op si Put ya lo movió
		defer tmp.Close()

		var notes, description string
		gotAPK := false
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err == nil {
				switch part.FormName() {
				case "apk":
					_, err = io.Copy(tmp, part)
					gotAPK = true
				case "notes", "description":
					var b []byte
					b, err = io.ReadAll(io.LimitReader(part, 64<<10))
					if part.FormName() == "notes" {
						notes = strings.TrimSpace(string(b))
					} else {
						description = strings.TrimSpace(string(b))
					}
				}
			}
			if err != nil {
				var tooBig *http.MaxBytesError
				if errors.As(err, &tooBig) {
					writeError(w, http.StatusRequestEntityTooLarge, "la APK supera el tamaño máximo permitido")
				} else {
					writeError(w, http.StatusBadRequest, "subida incompleta o mal formada")
				}
				return
			}
		}
		if !gotAPK {
			writeError(w, http.StatusBadRequest, "falta el campo apk")
			return
		}
		if err := tmp.Close(); err != nil {
			storeError(w, err)
			return
		}
		a, err := s.Put(tmp.Name(), notes, description)
		if err != nil {
			storeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, withURLs(a))
	}))

	mux.HandleFunc("PATCH /api/apps/{id}", auth(func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Description, Notes *string }
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "JSON no válido")
			return
		}
		a, err := s.Update(r.PathValue("id"), body.Description, body.Notes)
		if err != nil {
			storeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, withURLs(a))
	}))

	mux.HandleFunc("DELETE /api/apps/{id}", auth(func(w http.ResponseWriter, r *http.Request) {
		if err := s.Delete(r.PathValue("id")); err != nil {
			storeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	return mux
}
