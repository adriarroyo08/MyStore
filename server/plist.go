package main

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"math"
	"strconv"
	"strings"
	"unicode/utf16"
)

var errPlist = errors.New("plist no válido")

// parsePlist lee un plist XML o binario (bplist00) y devuelve map[string]any, []any,
// string, int64, float64 o bool. Fechas y datos se devuelven como nil.
func parsePlist(data []byte) (any, error) {
	if bytes.HasPrefix(data, []byte("bplist00")) {
		return parseBplist(data)
	}
	d := xml.NewDecoder(bytes.NewReader(data))
	d.Strict = false
	d.Entity = xml.HTMLEntity
	for {
		t, err := d.Token()
		if err != nil {
			return nil, errPlist
		}
		if se, ok := t.(xml.StartElement); ok && se.Name.Local == "plist" {
			return xmlValue(d, 0)
		}
	}
}

// xmlValue lee el siguiente valor (saltando espacios) dentro del elemento actual.
func xmlValue(d *xml.Decoder, depth int) (any, error) {
	if depth > 64 {
		return nil, errPlist
	}
	for {
		t, err := d.Token()
		if err != nil {
			return nil, errPlist
		}
		if _, ok := t.(xml.EndElement); ok {
			return nil, errEnd // cierre del array que contiene la llamada
		}
		se, ok := t.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "dict":
			m := map[string]any{}
			for {
				k, end, err := xmlKey(d)
				if err != nil {
					return nil, err
				}
				if end {
					return m, nil
				}
				if m[k], err = xmlValue(d, depth+1); err != nil {
					return nil, err
				}
			}
		case "array":
			l := []any{}
			for {
				v, err := xmlValue(d, depth+1)
				if err == errEnd {
					return l, nil
				} else if err != nil {
					return nil, err
				}
				l = append(l, v)
			}
		case "true", "false":
			d.Skip()
			return se.Name.Local == "true", nil
		case "string", "integer", "real", "date", "data", "key":
			var s string
			if d.DecodeElement(&s, &se) != nil {
				return nil, errPlist
			}
			switch se.Name.Local {
			case "integer":
				n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
				return n, err
			case "real":
				f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
				return f, err
			case "date", "data":
				return nil, nil
			}
			return s, nil
		default:
			return nil, errPlist
		}
	}
}

var errEnd = errors.New("fin")

// xmlKey devuelve la siguiente <key> de un dict, o end=true al cerrarlo.
func xmlKey(d *xml.Decoder) (string, bool, error) {
	for {
		t, err := d.Token()
		if err != nil {
			return "", false, errPlist
		}
		switch t := t.(type) {
		case xml.EndElement:
			return "", true, nil
		case xml.StartElement:
			var s string
			if t.Name.Local != "key" || d.DecodeElement(&s, &t) != nil {
				return "", false, errPlist
			}
			return s, false, nil
		}
	}
}

func parseBplist(b []byte) (v any, err error) {
	defer func() {
		if recover() != nil {
			v, err = nil, errPlist
		}
	}()
	if len(b) < 8+32 {
		return nil, errPlist
	}
	tr := b[len(b)-32:]
	offSize, refSize := int(tr[6]), int(tr[7])
	n := binary.BigEndian.Uint64(tr[8:16])
	top := binary.BigEndian.Uint64(tr[16:24])
	table := binary.BigEndian.Uint64(tr[24:32])
	if offSize < 1 || offSize > 8 || refSize < 1 || refSize > 8 || n == 0 || top >= n || table > uint64(len(b)) || n > uint64(len(b)) {
		return nil, errPlist
	}
	uint := func(p []byte) uint64 {
		var x uint64
		for _, c := range p {
			x = x<<8 | uint64(c)
		}
		return x
	}
	var obj func(i uint64, depth int) any
	obj = func(i uint64, depth int) any {
		if depth > 64 || i >= n {
			panic(errPlist)
		}
		pos := uint(b[table+i*uint64(offSize):][:offSize])
		m := b[pos]
		kind, low := m>>4, uint64(m&0x0f)
		pos++
		count := func() uint64 { // longitud en el nibble bajo, o en un entero siguiente
			if low != 0x0f {
				return low
			}
			sz := 1 << (b[pos] & 0x0f)
			c := uint(b[pos+1:][:sz])
			pos += 1 + uint64(sz)
			return c
		}
		switch kind {
		case 0x0:
			return m == 0x09
		case 0x1:
			sz := 1 << low
			return int64(uint(b[pos:][:sz])) // 8 bytes con signo; los de 1-4 son sin signo
		case 0x2:
			if low == 3 {
				return math.Float64frombits(uint(b[pos:][:8]))
			}
			return float64(math.Float32frombits(uint32(uint(b[pos:][:4]))))
		case 0x5:
			c := count()
			return string(b[pos:][:c])
		case 0x6:
			c := count()
			u := make([]uint16, c)
			for j := range u {
				u[j] = binary.BigEndian.Uint16(b[pos+uint64(j)*2:])
			}
			return string(utf16.Decode(u))
		case 0xa:
			c := count()
			l := make([]any, c)
			for j := range l {
				l[j] = obj(uint(b[pos+uint64(j)*uint64(refSize):][:refSize]), depth+1)
			}
			return l
		case 0xd:
			c := count()
			mp := make(map[string]any, c)
			for j := uint64(0); j < c; j++ {
				k, _ := obj(uint(b[pos+j*uint64(refSize):][:refSize]), depth+1).(string)
				mp[k] = obj(uint(b[pos+(c+j)*uint64(refSize):][:refSize]), depth+1)
			}
			return mp
		}
		return nil // fechas, datos, UIDs: no hacen falta
	}
	return obj(top, 0), nil
}

