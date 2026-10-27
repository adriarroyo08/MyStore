package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound     = errors.New("app no encontrada")
	ErrNotNewer     = errors.New("ya hay una versión igual o más nueva de esta app")
	ErrCertMismatch = errors.New("la APK está firmada con otra clave; borra la app si quieres cambiar de firma")
)

type App struct {
	Meta
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Notes       string    `json:"notes"`
	HasIcon     bool      `json:"hasIcon"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Store guarda cada app en apps/<id>/{app.apk, icon.png, meta.json}.
// ponytail: un solo mutex para todas las escrituras; basta para una tienda personal.
type Store struct {
	apps string
	mu   sync.Mutex
}

var idRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_.]*-(phone|watch)$`)

func NewStore(dir string) (*Store, error) {
	apps := filepath.Join(dir, "apps")
	if err := os.MkdirAll(apps, 0o755); err != nil {
		return nil, err
	}
	// Restos de escrituras interrumpidas. Un .old-<id> sin <id> es un corte a mitad de
	// replaceDir: se restaura en vez de borrarse.
	entries, err := os.ReadDir(apps)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, ".") {
			continue
		}
		if id, ok := strings.CutPrefix(name, ".old-"); ok {
			if _, err := os.Stat(filepath.Join(apps, id)); errors.Is(err, os.ErrNotExist) {
				if os.Rename(filepath.Join(apps, name), filepath.Join(apps, id)) == nil {
					continue
				}
			}
		}
		os.RemoveAll(filepath.Join(apps, name))
	}
	return &Store{apps: apps}, nil
}

func (s *Store) dir(id string) (string, error) {
	if !idRe.MatchString(id) {
		return "", ErrNotFound
	}
	return filepath.Join(s.apps, id), nil
}

func (s *Store) Get(id string) (App, error) {
	d, err := s.dir(id)
	if err != nil {
		return App{}, err
	}
	data, err := os.ReadFile(filepath.Join(d, "meta.json"))
	if errors.Is(err, os.ErrNotExist) {
		return App{}, ErrNotFound
	} else if err != nil {
		return App{}, err
	}
	var a App
	return a, json.Unmarshal(data, &a)
}

// List devuelve las apps de un tipo ("phone", "watch" o "" para todas), ordenadas por nombre.
func (s *Store) List(typ string) ([]App, error) {
	entries, err := os.ReadDir(s.apps)
	if err != nil {
		return nil, err
	}
	apps := []App{}
	for _, e := range entries {
		a, err := s.Get(e.Name())
		if err != nil {
			continue // directorio temporal o ajeno
		}
		if typ == "" || a.Type == typ {
			apps = append(apps, a)
		}
	}
	sort.Slice(apps, func(i, j int) bool { return strings.ToLower(apps[i].Label) < strings.ToLower(apps[j].Label) })
	return apps, nil
}

// Put valida la APK en tmpAPK y la mueve al almacén, sustituyendo la versión anterior.
// Si description viene vacía se conserva la anterior.
func (s *Store) Put(tmpAPK, notes, description string) (App, error) {
	meta, icon, err := ParseAPK(tmpAPK)
	if err != nil {
		return App{}, err
	}
	a := App{Meta: meta, ID: meta.Package + "-" + meta.Type, Notes: notes, Description: description, HasIcon: icon != nil}
	dest, err := s.dir(a.ID)
	if err != nil {
		return App{}, fmt.Errorf("%w: nombre de paquete no admitido", ErrInvalidAPK)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	old, err := s.Get(a.ID)
	switch {
	case err == nil:
		if meta.VersionCode <= old.VersionCode {
			return App{}, fmt.Errorf("%w (guardada: %s, código %d)", ErrNotNewer, old.VersionName, old.VersionCode)
		}
		if meta.CertSHA256 != old.CertSHA256 {
			return App{}, ErrCertMismatch
		}
		if a.Description == "" {
			a.Description = old.Description
		}
	case !errors.Is(err, ErrNotFound):
		return App{}, err
	}
	a.UpdatedAt = time.Now().UTC()

	tmp, err := os.MkdirTemp(s.apps, ".tmp-")
	if err != nil {
		return App{}, err
	}
	defer os.RemoveAll(tmp)
	if err := moveFile(tmpAPK, filepath.Join(tmp, "app.apk")); err != nil {
		return App{}, err
	}
	if icon != nil {
		if err := os.WriteFile(filepath.Join(tmp, "icon.png"), icon, 0o644); err != nil {
			return App{}, err
		}
	}
	if err := writeMeta(tmp, a); err != nil {
		return App{}, err
	}
	return a, replaceDir(tmp, dest)
}

func (s *Store) Update(id string, description, notes *string) (App, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, err := s.Get(id)
	if err != nil {
		return App{}, err
	}
	if description != nil {
		a.Description = *description
	}
	if notes != nil {
		a.Notes = *notes
	}
	d, _ := s.dir(id)
	return a, writeMeta(d, a)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.Get(id); err != nil {
		return err
	}
	d, _ := s.dir(id)
	return os.RemoveAll(d)
}

func (s *Store) APKPath(id string) (string, error) {
	if _, err := s.Get(id); err != nil {
		return "", err
	}
	d, _ := s.dir(id)
	return filepath.Join(d, "app.apk"), nil
}

// IconPath devuelve la ruta del icono y si la app tiene uno.
func (s *Store) IconPath(id string) (string, bool, error) {
	a, err := s.Get(id)
	if err != nil {
		return "", false, err
	}
	d, _ := s.dir(id)
	return filepath.Join(d, "icon.png"), a.HasIcon, nil
}

// writeMeta escribe meta.json de forma atómica.
func writeMeta(dir string, a App) error {
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, "meta.json.tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, "meta.json"))
}

// replaceDir pone src en el lugar de dst. En Windows no se puede renombrar encima de un
// directorio, así que se aparta el antiguo y se restaura si algo falla.
func replaceDir(src, dst string) error {
	old := filepath.Join(filepath.Dir(dst), ".old-"+filepath.Base(dst))
	os.RemoveAll(old)
	hadOld := os.Rename(dst, old) == nil
	if err := os.Rename(src, dst); err != nil {
		if hadOld {
			os.Rename(old, dst)
		}
		return err
	}
	return os.RemoveAll(old)
}

// moveFile renombra y, si está en otro disco, copia.
func moveFile(src, dst string) error {
	if os.Rename(src, dst) == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	in.Close()
	return os.Remove(src)
}
