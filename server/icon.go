package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"math"
	"strconv"
	"strings"

	"github.com/avast/apkparser"
	"golang.org/x/image/draw"
	"golang.org/x/image/vector"
	_ "golang.org/x/image/webp"
)

const iconSize = 192

// loadResources lee resources.arsc; nil si no hay o no se entiende (las referencias quedarán sin resolver).
func loadResources(zr *apkparser.ZipReader) (res *apkparser.ResourceTable) {
	defer func() {
		if recover() != nil {
			res = nil
		}
	}()
	f := zr.File["resources.arsc"]
	if f == nil {
		return nil
	}
	data, err := f.ReadAll(64 << 20)
	if err != nil {
		return nil
	}
	res, _ = apkparser.ParseResourceTable(bytes.NewReader(data))
	return res
}

// renderIcon devuelve un PNG cuadrado de iconSize px para el recurso de icono indicado
// (ruta dentro de la APK), o nil si no sabe dibujarlo.
func renderIcon(zr *apkparser.ZipReader, res *apkparser.ResourceTable, path string) (out []byte) {
	defer func() {
		if recover() != nil {
			out = nil
		}
	}()
	dst := image.NewRGBA(image.Rect(0, 0, iconSize, iconSize))
	if !drawLayer(dst, zr, res, path, dst.Bounds(), 0) {
		return nil
	}
	var buf bytes.Buffer
	if png.Encode(&buf, dst) != nil {
		return nil
	}
	return buf.Bytes()
}

// readXML pasa el XML binario de la APK a texto, resolviendo referencias con res.
func readXML(zr *apkparser.ZipReader, res *apkparser.ResourceTable, name string) ([]byte, error) {
	f := zr.File[name]
	if f == nil {
		return nil, fmt.Errorf("no existe %s", name)
	}
	data, err := f.ReadAll(5 << 20)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	if err := apkparser.ParseXml(bytes.NewReader(data), enc, res); err != nil {
		return nil, err
	}
	enc.Flush()
	return buf.Bytes(), nil
}

func attrMap(se xml.StartElement) map[string]string {
	m := make(map[string]string, len(se.Attr))
	for _, a := range se.Attr {
		m[a.Name.Local] = a.Value
	}
	return m
}

// drawLayer pinta ref (color "#...", PNG/WebP/JPEG o XML vector/adaptive-icon) en dst, de modo que
// los límites del drawable ocupen rect. Devuelve false si no pintó nada.
func drawLayer(dst *image.RGBA, zr *apkparser.ZipReader, res *apkparser.ResourceTable, ref string, rect image.Rectangle, depth int) bool {
	if strings.HasPrefix(ref, "#") {
		c, ok := parseColor(ref)
		if ok {
			draw.Draw(dst, dst.Bounds(), &image.Uniform{c}, image.Point{}, draw.Over)
		}
		return ok
	}
	f := zr.File[ref]
	if f == nil {
		return false
	}
	if !strings.HasSuffix(ref, ".xml") {
		data, err := f.ReadAll(5 << 20)
		if err != nil {
			return false
		}
		img, err := decodeBitmap(data)
		if err != nil {
			return false
		}
		draw.CatmullRom.Scale(dst, rect, img, img.Bounds(), draw.Over, nil)
		return true
	}
	doc, err := readXML(zr, res, ref)
	if err != nil {
		return false
	}
	switch rootElement(doc) {
	case "vector":
		return drawVector(dst, doc, rect) == nil
	case "adaptive-icon":
		return depth == 0 && drawAdaptive(dst, zr, res, doc)
	}
	// ponytail: solo vector y adaptive-icon; layer-list, inset, bitmap XML, etc. → sin icono.
	return false
}

func rootElement(doc []byte) string {
	d := xml.NewDecoder(bytes.NewReader(doc))
	for {
		tok, err := d.Token()
		if err != nil {
			return ""
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se.Name.Local
		}
	}
}

// drawAdaptive: lienzo de 108 dp, se muestra el 72 % central (zona visible de la máscara) a iconSize px.
// ponytail: sin <monochrome> ni drawables inline en background/foreground; si falla una capa se pinta la otra.
// Los drawables con varias densidades usan la primera que da apkparser (la más pequeña).
func drawAdaptive(dst *image.RGBA, zr *apkparser.ZipReader, res *apkparser.ResourceTable, doc []byte) bool {
	k := float64(iconSize) / 72
	lo, hi := int(math.Round(-18*k)), int(math.Round(90*k))
	rect := image.Rect(lo, lo, hi, hi)
	layers := map[string]string{}
	d := xml.NewDecoder(bytes.NewReader(doc))
	for {
		tok, err := d.Token()
		if err != nil {
			break
		}
		if se, ok := tok.(xml.StartElement); ok && (se.Name.Local == "background" || se.Name.Local == "foreground") {
			layers[se.Name.Local] = attrMap(se)["drawable"]
		}
	}
	drew := false
	for _, name := range []string{"background", "foreground"} {
		if ref := layers[name]; ref != "" && drawLayer(dst, zr, res, ref, rect, 1) {
			drew = true
		}
	}
	return drew
}

// parseColor acepta #RGB, #ARGB, #RRGGBB, #AARRGGBB y el entero con signo que apkparser da a los colores literales.
func parseColor(s string) (color.NRGBA, bool) {
	var v uint64
	if h, ok := strings.CutPrefix(s, "#"); ok {
		if len(h) == 3 {
			h = "f" + h
		}
		if len(h) == 4 { // cada dígito se duplica
			h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2], h[3], h[3]})
		}
		n, err := strconv.ParseUint(h, 16, 32)
		if err != nil || (len(h) != 6 && len(h) != 8) {
			return color.NRGBA{}, false
		}
		if len(h) == 6 {
			n |= 0xff000000
		}
		v = n
	} else {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil || n < math.MinInt32 || n > math.MaxUint32 {
			return color.NRGBA{}, false
		}
		v = uint64(uint32(n))
	}
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: uint8(v >> 24)}, true
}

// affine: x' = a·x + c·y + e ; y' = b·x + d·y + f
type affine [6]float64

func (m affine) mul(n affine) affine { // aplica n y luego m
	return affine{
		m[0]*n[0] + m[2]*n[1], m[1]*n[0] + m[3]*n[1],
		m[0]*n[2] + m[2]*n[3], m[1]*n[2] + m[3]*n[3],
		m[0]*n[4] + m[2]*n[5] + m[4], m[1]*n[4] + m[3]*n[5] + m[5],
	}
}

func (m affine) apply(x, y float64) (float64, float64) {
	return m[0]*x + m[2]*y + m[4], m[1]*x + m[3]*y + m[5]
}

func translate(x, y float64) affine { return affine{1, 0, 0, 1, x, y} }

func floatAttr(m map[string]string, k string, def float64) float64 {
	if v, err := strconv.ParseFloat(m[k], 64); err == nil {
		return v
	}
	return def
}

// drawVector rasteriza los <path> con relleno de un <vector>; el viewport ocupa rect.
// ponytail: sin trazo, gradientes, clip-path ni fillType evenOdd (el rasterizador es non-zero); se omiten.
func drawVector(dst *image.RGBA, doc []byte, rect image.Rectangle) error {
	d := xml.NewDecoder(bytes.NewReader(doc))
	var stack []affine
	rz := vector.NewRasterizer(dst.Bounds().Dx(), dst.Bounds().Dy())
	for {
		tok, err := d.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			a := attrMap(t)
			switch t.Name.Local {
			case "vector":
				vw, vh := floatAttr(a, "viewportWidth", 0), floatAttr(a, "viewportHeight", 0)
				if vw <= 0 || vh <= 0 {
					return fmt.Errorf("viewport inválido")
				}
				stack = append(stack, affine{float64(rect.Dx()) / vw, 0, 0, float64(rect.Dy()) / vh, float64(rect.Min.X), float64(rect.Min.Y)})
			case "group":
				if len(stack) == 0 {
					continue
				}
				px, py := floatAttr(a, "pivotX", 0), floatAttr(a, "pivotY", 0)
				sin, cos := math.Sincos(floatAttr(a, "rotation", 0) * math.Pi / 180)
				g := translate(floatAttr(a, "translateX", 0), floatAttr(a, "translateY", 0)).
					mul(translate(px, py)).
					mul(affine{cos, sin, -sin, cos, 0, 0}).
					mul(affine{floatAttr(a, "scaleX", 1), 0, 0, floatAttr(a, "scaleY", 1), 0, 0}).
					mul(translate(-px, -py))
				stack = append(stack, stack[len(stack)-1].mul(g))
			case "path":
				if len(stack) > 0 {
					fillPath(dst, rz, stack[len(stack)-1], a)
				}
			}
		case xml.EndElement:
			if (t.Name.Local == "group" || t.Name.Local == "vector") && len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if len(stack) == 0 {
		return fmt.Errorf("sin <vector>")
	}
	return nil
}

func fillPath(dst *image.RGBA, rz *vector.Rasterizer, m affine, a map[string]string) {
	c, ok := parseColor(a["fillColor"]) // referencia a gradiente o sin resolver → sin relleno
	if !ok {
		return
	}
	segs, err := parsePath(a["pathData"])
	if err != nil {
		return
	}
	c.A = uint8(math.Round(float64(c.A) * math.Max(0, math.Min(1, floatAttr(a, "fillAlpha", 1)))))
	rz.Reset(dst.Bounds().Dx(), dst.Bounds().Dy())
	pt := func(i int, s seg) (float32, float32) {
		x, y := m.apply(s.p[i], s.p[i+1])
		if !(math.Abs(x) < 1e6 && math.Abs(y) < 1e6) { // también cubre NaN
			panic("coordenada fuera de rango") // renderIcon lo convierte en nil
		}
		return float32(x), float32(y)
	}
	for _, s := range segs {
		switch s.op {
		case 'M':
			x, y := pt(0, s)
			rz.MoveTo(x, y)
		case 'L':
			x, y := pt(0, s)
			rz.LineTo(x, y)
		case 'Q':
			x1, y1 := pt(0, s)
			x, y := pt(2, s)
			rz.QuadTo(x1, y1, x, y)
		case 'C':
			x1, y1 := pt(0, s)
			x2, y2 := pt(2, s)
			x, y := pt(4, s)
			rz.CubeTo(x1, y1, x2, y2, x, y)
		case 'Z':
			rz.ClosePath()
		}
	}
	rz.Draw(dst, dst.Bounds(), image.NewUniform(c), image.Point{})
}

// seg es un segmento de ruta absoluto: op ∈ M L Q C Z (H V S T A se normalizan a estos).
type seg struct {
	op byte
	p  [6]float64
}

var pathArgs = map[byte]int{'M': 2, 'L': 2, 'T': 2, 'H': 1, 'V': 1, 'C': 6, 'S': 4, 'Q': 4, 'A': 7, 'Z': 0}

func parsePath(d string) ([]seg, error) {
	var out []seg
	var cx, cy, sx, sy, lx, ly float64 // actual, inicio del subtrayecto, último punto de control
	var prev byte
	i := 0
	skip := func() {
		for i < len(d) && strings.IndexByte(" ,\t\r\n", d[i]) >= 0 {
			i++
		}
	}
	isDigit := func(j int) bool { return j < len(d) && d[j] >= '0' && d[j] <= '9' }
	num := func() (float64, error) {
		skip()
		j := i
		if j < len(d) && (d[j] == '+' || d[j] == '-') {
			j++
		}
		digits := isDigit(j)
		for isDigit(j) {
			j++
		}
		if j < len(d) && d[j] == '.' {
			j++
			for isDigit(j) {
				j++
				digits = true
			}
		}
		if !digits {
			return 0, fmt.Errorf("número esperado en %d", i)
		}
		if j < len(d) && (d[j] == 'e' || d[j] == 'E') {
			k := j + 1
			if k < len(d) && (d[k] == '+' || d[k] == '-') {
				k++
			}
			if isDigit(k) {
				for isDigit(k) {
					k++
				}
				j = k
			}
		}
		v, err := strconv.ParseFloat(d[i:j], 64)
		i = j
		return v, err
	}
	flag := func() (float64, error) { // los flags de arco pueden ir pegados: "a5,5 0 011,1"
		skip()
		if i < len(d) && (d[i] == '0' || d[i] == '1') {
			i++
			return float64(d[i-1] - '0'), nil
		}
		return 0, fmt.Errorf("flag de arco esperado en %d", i)
	}
	for {
		skip()
		if i >= len(d) {
			return out, nil
		}
		c := d[i]
		up := c &^ 0x20 // mayúscula
		n, ok := pathArgs[up]
		if !ok {
			return nil, fmt.Errorf("comando %q desconocido", c)
		}
		rel := c != up
		i++
		for {
			var a [7]float64
			for k := 0; k < n; k++ {
				var err error
				if up == 'A' && (k == 3 || k == 4) {
					a[k], err = flag()
				} else {
					a[k], err = num()
				}
				if err != nil {
					return nil, err
				}
			}
			if rel { // relativo → absoluto
				switch up {
				case 'H':
					a[0] += cx
				case 'V':
					a[0] += cy
				case 'A':
					a[5] += cx
					a[6] += cy
				default:
					for k := 0; k < n; k += 2 {
						a[k] += cx
						a[k+1] += cy
					}
				}
			}
			switch up {
			case 'M':
				cx, cy, sx, sy = a[0], a[1], a[0], a[1]
				out = append(out, seg{'M', [6]float64{cx, cy}})
				up = 'L' // los pares siguientes son líneas
			case 'L':
				cx, cy = a[0], a[1]
				out = append(out, seg{'L', [6]float64{cx, cy}})
			case 'H':
				cx = a[0]
				out = append(out, seg{'L', [6]float64{cx, cy}})
			case 'V':
				cy = a[0]
				out = append(out, seg{'L', [6]float64{cx, cy}})
			case 'C':
				out = append(out, seg{'C', [6]float64{a[0], a[1], a[2], a[3], a[4], a[5]}})
				lx, ly, cx, cy = a[2], a[3], a[4], a[5]
			case 'S':
				x1, y1 := cx, cy
				if prev == 'C' || prev == 'S' {
					x1, y1 = 2*cx-lx, 2*cy-ly
				}
				out = append(out, seg{'C', [6]float64{x1, y1, a[0], a[1], a[2], a[3]}})
				lx, ly, cx, cy = a[0], a[1], a[2], a[3]
			case 'Q':
				out = append(out, seg{'Q', [6]float64{a[0], a[1], a[2], a[3]}})
				lx, ly, cx, cy = a[0], a[1], a[2], a[3]
			case 'T':
				x1, y1 := cx, cy
				if prev == 'Q' || prev == 'T' {
					x1, y1 = 2*cx-lx, 2*cy-ly
				}
				out = append(out, seg{'Q', [6]float64{x1, y1, a[0], a[1]}})
				lx, ly, cx, cy = x1, y1, a[0], a[1]
			case 'A':
				out = append(out, arcToCubics(cx, cy, a[0], a[1], a[2], a[3] != 0, a[4] != 0, a[5], a[6])...)
				cx, cy = a[5], a[6]
			case 'Z':
				out = append(out, seg{op: 'Z'})
				cx, cy = sx, sy
			}
			prev = up
			if up == 'Z' {
				break
			}
			skip()
			if !(i < len(d) && (isDigit(i) || d[i] == '.' || d[i] == '-' || d[i] == '+')) {
				break
			}
		}
	}
}

// arcToCubics convierte un arco SVG (parametrización por extremos) en curvas cúbicas de ≤90°.
func arcToCubics(x0, y0, rx, ry, rot float64, large, sweep bool, x, y float64) []seg {
	if x0 == x && y0 == y {
		return nil
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	if rx == 0 || ry == 0 {
		return []seg{{'L', [6]float64{x, y}}}
	}
	sinP, cosP := math.Sincos(rot * math.Pi / 180)
	dx, dy := (x0-x)/2, (y0-y)/2
	x1, y1 := cosP*dx+sinP*dy, -sinP*dx+cosP*dy
	if l := x1*x1/(rx*rx) + y1*y1/(ry*ry); l > 1 {
		s := math.Sqrt(l)
		rx, ry = rx*s, ry*s
	}
	num := rx*rx*ry*ry - rx*rx*y1*y1 - ry*ry*x1*x1
	coef := math.Sqrt(math.Max(0, num/(rx*rx*y1*y1+ry*ry*x1*x1)))
	if large == sweep {
		coef = -coef
	}
	cxp, cyp := coef*rx*y1/ry, -coef*ry*x1/rx
	cx, cy := cosP*cxp-sinP*cyp+(x0+x)/2, sinP*cxp+cosP*cyp+(y0+y)/2
	th1 := math.Atan2((y1-cyp)/ry, (x1-cxp)/rx)
	dth := math.Atan2((-y1-cyp)/ry, (-x1-cxp)/rx) - th1
	if sweep && dth < 0 {
		dth += 2 * math.Pi
	} else if !sweep && dth > 0 {
		dth -= 2 * math.Pi
	}
	n := int(math.Ceil(math.Abs(dth) / (math.Pi / 2)))
	if n == 0 {
		return nil
	}
	step := dth / float64(n)
	t := 4.0 / 3 * math.Tan(step/4)
	pos := func(a float64) (float64, float64) {
		s, c := math.Sincos(a)
		return cx + rx*c*cosP - ry*s*sinP, cy + rx*c*sinP + ry*s*cosP
	}
	der := func(a float64) (float64, float64) {
		s, c := math.Sincos(a)
		return -rx*s*cosP - ry*c*sinP, -rx*s*sinP + ry*c*cosP
	}
	out := make([]seg, 0, n)
	for k := 0; k < n; k++ {
		a, b := th1+float64(k)*step, th1+float64(k+1)*step
		pax, pay := pos(a)
		pbx, pby := pos(b)
		dax, day := der(a)
		dbx, dby := der(b)
		out = append(out, seg{'C', [6]float64{pax + t*dax, pay + t*day, pbx - t*dbx, pby - t*dby, pbx, pby}})
	}
	out[n-1].p[4], out[n-1].p[5] = x, y // sin deriva numérica en el extremo
	return out
}

// decodeBitmap decodifica PNG/WebP rechazando dimensiones absurdas: un fichero pequeño puede
// declarar 60000×60000 y hacer reservar gigas al decodificarlo.
func decodeBitmap(data []byte) (image.Image, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if cfg.Width > 4096 || cfg.Height > 4096 {
		return nil, fmt.Errorf("imagen demasiado grande: %dx%d", cfg.Width, cfg.Height)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}
