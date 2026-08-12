package serialization

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"

	c2q "github.com/Inoriol/comquad/compose2quadlet/internal/types"
)

func Marshal(unit c2q.QuadletUnit) string {
	var b strings.Builder
	writeUnit(&b, unit)
	return b.String()
}

func Write(w io.Writer, units []c2q.QuadletUnit) error {
	for i, u := range units {
		if i > 0 {
			if _, err := io.WriteString(w, "\n"); err != nil {
				return err
			}
		}
		if err := writeUnit(w, u); err != nil {
			return err
		}
	}
	return nil
}

func WriteUnits(dir string, units []c2q.QuadletUnit) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for _, u := range units {
		fileName := u.Name + "." + string(u.Type)
		path := filepath.Join(dir, fileName)
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		if err := writeUnit(f, u); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}

func writeUnit(w io.Writer, unit c2q.QuadletUnit) error {
	first := true
	for _, sec := range unit.Sections {
		if len(sec.Directives) == 0 {
			continue
		}
		if !first {
			if _, err := io.WriteString(w, "\n"); err != nil {
				return err
			}
		}
		first = false
		if _, err := io.WriteString(w, "["+sec.Name+"]\n"); err != nil {
			return err
		}
		for _, d := range sec.Directives {
			if len(d.Values) == 0 {
				if _, err := io.WriteString(w, d.Key+"=\n"); err != nil {
					return err
				}
				continue
			}
			for _, v := range d.Values {
				if _, err := io.WriteString(w, d.Key+"="+v+"\n"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func Unmarshal(data string, unitType c2q.UnitType, name string) c2q.QuadletUnit {
	u := c2q.QuadletUnit{Type: unitType, Name: name}
	scanner := bufio.NewScanner(strings.NewReader(data))
	var cur *c2q.Section

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			u.Sections = append(u.Sections, c2q.Section{Name: line[1 : len(line)-1]})
			cur = &u.Sections[len(u.Sections)-1]
			continue
		}
		if cur == nil {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := line[:eq]
		value := line[eq+1:]

		if len(cur.Directives) > 0 && cur.Directives[len(cur.Directives)-1].Key == key {
			cur.Directives[len(cur.Directives)-1].Values = append(
				cur.Directives[len(cur.Directives)-1].Values, value,
			)
		} else {
			var vals []string
			if value != "" {
				vals = []string{value}
			}
			cur.Directives = append(cur.Directives, c2q.Directive{Key: key, Values: vals})
		}
	}
	return u
}
