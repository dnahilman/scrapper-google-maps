package domain

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
)

// StringArray maps Postgres text[] to []string for GORM.
type StringArray []string

func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	escaped := make([]string, len(s))
	for i, v := range s {
		v = strings.ReplaceAll(v, `\`, `\\`)
		v = strings.ReplaceAll(v, `"`, `\"`)
		escaped[i] = `"` + v + `"`
	}
	return "{" + strings.Join(escaped, ",") + "}", nil
}

func (s *StringArray) Scan(src any) error {
	if src == nil {
		*s = nil
		return nil
	}
	var raw string
	switch v := src.(type) {
	case []byte:
		raw = string(v)
	case string:
		raw = v
	default:
		return fmt.Errorf("StringArray: unsupported scan source %T", src)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		*s = []string{}
		return nil
	}
	if !strings.HasPrefix(raw, "{") || !strings.HasSuffix(raw, "}") {
		return errors.New("StringArray: invalid array literal")
	}
	body := raw[1 : len(raw)-1]
	out := []string{}
	var buf strings.Builder
	inQuote := false
	escaped := false
	for i := 0; i < len(body); i++ {
		c := body[i]
		switch {
		case escaped:
			buf.WriteByte(c)
			escaped = false
		case c == '\\':
			escaped = true
		case c == '"':
			inQuote = !inQuote
		case c == ',' && !inQuote:
			out = append(out, buf.String())
			buf.Reset()
		default:
			buf.WriteByte(c)
		}
	}
	out = append(out, buf.String())
	*s = out
	return nil
}
