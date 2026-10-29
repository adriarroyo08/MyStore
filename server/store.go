package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/image/draw"
)

var (
	ErrNotFound     = errors.New("app no encontrada")
	ErrNotNewer     = errors.New("ya hay una versión igual o más nueva de esta app")
	ErrCertMismatch = errors.New("la APK está firmada con otra clave; borra la app si quieres cambiar de firma")
	ErrBadImage     = errors.New("la captura no es una imagen PNG, JPEG o WebP válida de hasta 4096 px por lado")
	ErrBadIcon      = errors.New("el icono no es una imagen PNG, JPEG o WebP válida de hasta 4096 px por lado")
	ErrTooManyShots = errors.New("ya hay 8 capturas; quita alguna antes de añadir otra")
)

const maxShots = 8

// Versiones anteriores que se guardan por app (las APKs pueden pesar más de 1 GB).
var maxVersions = 3

// Version es una versión anterior de una app; su APK está en versions/<versionCode>.apk.
type Version struct {
	VersionCode int64     `json:"versionCode"`
	VersionName string    `json:"versionName"`
	Notes       string    `json:"notes"`
	Size        int64     `json:"size"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type App struct {
	Meta
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Notes       string    `json:"notes"`
	HasIcon     bool      `json:"hasIcon"`
	CustomIcon  bool      `json:"customIcon"` // icon.png lo subió el admin, no viene de la APK
	Shots       []string  `json:"shots"` // nombres de fichero en shots/, en orden de subida
	Versions    []Version `json:"versions"` // anteriores, de la más nueva a la más vieja
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Store guarda cada app en apps/<id>/{app.apk, icon.png, meta.json}.
// ponytail: un solo mutex para todas las escrituras; basta para una tienda personal.
type Store struct {
	root    string // raíz de datos; apps/ y devices.json cuelgan de aquí
	apps    string
	mu      sync.Mutex
	devMu   sync.Mutex
	devices map[string]time.Time
}

// Device es un cliente que ha consultado el catálogo (cabecera X-MyStore-Device).
type Device struct {
	Device   string    `json:"device"`
	LastSeen time.Time `json:"lastSeen"`
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
	s := &Store{root: dir, apps: apps, devices: map[string]time.Time{}}
	if data, err := os.ReadFile(filepath.Join(dir, "devices.json")); err == nil {
		json.Unmarshal(data, &s.devices) // si está corrupto se empieza de cero
	}
	return s, nil
}

// SeenDevice anota el dispositivo; escribe a disco como mucho una vez por minuto y dispositivo.
func (s *Store) SeenDevice(device string) {
	device = strings.TrimSpace(device)
	if r := []rune(device); len(r) > 100 {
		device = string(r[:100])
	}
	if device == "" {
		return
	}
	s.devMu.Lock()
	defer s.devMu.Unlock()
	now := time.Now().UTC()
	if last, ok := s.devices[device]; ok && now.Sub(last) < time.Minute {
		return
	}
	s.devices[device] = now
	if data, err := json.Marshal(s.devices); err == nil {
		tmp := filepath.Join(s.root, "devices.json.tmp")
		if os.WriteFile(tmp, data, 0o644) == nil {
			os.Rename(tmp, filepath.Join(s.root, "devices.json"))
		}
	}
}

// Devices devuelve los dispositivos vistos, el más reciente primero.
func (s *Store) Devices() []Device {
	s.devMu.Lock()
	defer s.devMu.Unlock()
	out := make([]Device, 0, len(s.devices))
	for d, t := range s.devices {
		out = append(out, Device{d, t})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastSeen.After(out[j].LastSeen) })
	return out
}

// DiskBytes suma el tamaño de todo el árbol apps/.
// ponytail: los enlaces duros de versions/ cuentan dos veces; sobreestima, no importa para un panel.
func (s *Store) DiskBytes() int64 {
	var n int64
	filepath.WalkDir(s.apps, func(_ string, e os.DirEntry, err error) error {
		if err == nil && !e.IsDir() {
			if fi, err := e.Info(); err == nil {
				n += fi.Size()
			}
		}
		return nil
	})
	return n
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
		a.Shots = old.Shots
		a.Category = old.Category
		prev := Version{old.VersionCode, old.VersionName, old.Notes, old.Size, old.UpdatedAt}
		a.Versions = append([]Version{prev}, old.Versions...)
		a.Versions = a.Versions[:min(len(a.Versions), maxVersions)]
		if icon == nil && old.CustomIcon { // la APK nueva no trae icono: se conserva el personalizado
			a.HasIcon, a.CustomIcon = true, true
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
	if a.CustomIcon {
		if err := copyFile(filepath.Join(dest, "icon.png"), filepath.Join(tmp, "icon.png")); err != nil {
			return App{}, err
		}
	}
	if len(a.Shots) > 0 { // las capturas sobreviven a la nueva versión
		if err := os.Mkdir(filepath.Join(tmp, "shots"), 0o755); err != nil {
			return App{}, err
		}
		for _, f := range a.Shots {
			if err := copyFile(filepath.Join(dest, "shots", f), filepath.Join(tmp, "shots", f)); err != nil {
				return App{}, err
			}
		}
	}
	if len(a.Versions) > 0 { // enlaces duros: no se copian GB y el directorio viejo sigue intacto si algo falla
		if err := os.Mkdir(filepath.Join(tmp, "versions"), 0o755); err != nil {
			return App{}, err
		}
		for i, v := range a.Versions {
			src := filepath.Join(dest, "versions", fmt.Sprintf("%d.apk", v.VersionCode))
			if i == 0 {
				src = filepath.Join(dest, "app.apk")
			}
			if err := linkFile(src, filepath.Join(tmp, "versions", fmt.Sprintf("%d.apk", v.VersionCode))); err != nil {
				return App{}, err
			}
		}
	}
	if err := writeMeta(tmp, a); err != nil {
		return App{}, err
	}
	return a, replaceDir(tmp, dest)
}

func (s *Store) Update(id string, description, notes, category *string) (App, error) {
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
	if category != nil {
		a.Category = strings.TrimSpace(*category)
		if r := []rune(a.Category); len(r) > 40 {
			a.Category = strings.TrimSpace(string(r[:40]))
		}
	}
	d, _ := s.dir(id)
	return a, writeMeta(d, a)
}

// DeleteVersion quita una versión anterior del historial y borra su APK.
func (s *Store) DeleteVersion(id string, code int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, err := s.Get(id)
	if err != nil {
		return err
	}
	for i, v := range a.Versions {
		if v.VersionCode == code {
			a.Versions = append(a.Versions[:i:i], a.Versions[i+1:]...)
			d, _ := s.dir(id)
			if err := writeMeta(d, a); err != nil {
				return err
			}
			os.Remove(filepath.Join(d, "versions", fmt.Sprintf("%d.apk", code)))
			return nil
		}
	}
	return ErrNotFound
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

// VersionAPKPath devuelve la APK de una versión anterior que siga guardada.
func (s *Store) VersionAPKPath(id string, code int64) (Version, string, error) {
	a, err := s.Get(id)
	if err != nil {
		return Version{}, "", err
	}
	for _, v := range a.Versions {
		if v.VersionCode == code {
			d, _ := s.dir(id)
			return v, filepath.Join(d, "versions", fmt.Sprintf("%d.apk", code)), nil
		}
	}
	return Version{}, "", ErrNotFound
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

// SetIcon sustituye el icono de la app por data (PNG/JPEG/WebP), recortado a cuadrado y
// escalado a iconSize como los iconos de APK.
func (s *Store) SetIcon(id string, data []byte) (App, error) {
	img, err := decodeBitmap(data)
	if err != nil || img.Bounds().Empty() {
		return App{}, ErrBadIcon
	}
	b := img.Bounds()
	side := min(b.Dx(), b.Dy())
	src := image.Rect(b.Min.X+(b.Dx()-side)/2, b.Min.Y+(b.Dy()-side)/2, 0, 0)
	src.Max = src.Min.Add(image.Pt(side, side))
	dst := image.NewRGBA(image.Rect(0, 0, iconSize, iconSize))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, src, draw.Src, nil)
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return App{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a, err := s.Get(id)
	if err != nil {
		return App{}, err
	}
	d, _ := s.dir(id)
	tmp := filepath.Join(d, "icon.png.tmp")
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return App{}, err
	}
	if err := os.Rename(tmp, filepath.Join(d, "icon.png")); err != nil {
		return App{}, err
	}
	a.HasIcon, a.CustomIcon = true, true
	return a, writeMeta(d, a)
}

// AddShot valida data como imagen y la añade al final de las capturas de la app.
func (s *Store) AddShot(id string, data []byte) (App, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	ext := map[string]string{"png": ".png", "jpeg": ".jpg", "webp": ".webp"}[format]
	if err != nil || ext == "" || cfg.Width > 4096 || cfg.Height > 4096 || cfg.Width < 1 || cfg.Height < 1 {
		return App{}, ErrBadImage
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a, err := s.Get(id)
	if err != nil {
		return App{}, err
	}
	if len(a.Shots) >= maxShots {
		return App{}, ErrTooManyShots
	}
	d, _ := s.dir(id)
	if err := os.MkdirAll(filepath.Join(d, "shots"), 0o755); err != nil {
		return App{}, err
	}
	name := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	tmp := filepath.Join(d, "shots", name+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return App{}, err
	}
	if err := os.Rename(tmp, filepath.Join(d, "shots", name)); err != nil {
		return App{}, err
	}
	a.Shots = append(a.Shots, name)
	return a, writeMeta(d, a)
}

// RemoveShot quita la captura n (1..N); las siguientes se desplazan.
func (s *Store) RemoveShot(id string, n int) (App, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, err := s.Get(id)
	if err != nil {
		return App{}, err
	}
	if n < 1 || n > len(a.Shots) {
		return App{}, ErrNotFound
	}
	d, _ := s.dir(id)
	name := a.Shots[n-1]
	a.Shots = append(a.Shots[:n-1:n-1], a.Shots[n:]...)
	if err := writeMeta(d, a); err != nil {
		return App{}, err
	}
	os.Remove(filepath.Join(d, "shots", name))
	return a, nil
}

// ShotPath devuelve la ruta de la captura n (1..N). Solo se usan nombres de meta.json.
func (s *Store) ShotPath(id string, n int) (string, error) {
	a, err := s.Get(id)
	if err != nil {
		return "", err
	}
	if n < 1 || n > len(a.Shots) {
		return "", ErrNotFound
	}
	d, _ := s.dir(id)
	return filepath.Join(d, "shots", a.Shots[n-1]), nil
}

func copyFile(src, dst string) error {
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
	return out.Close()
}

// linkFile crea un enlace duro y, si el sistema no lo admite, copia.
func linkFile(src, dst string) error {
	if os.Link(src, dst) == nil {
		return nil
	}
	return copyFile(src, dst)
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
