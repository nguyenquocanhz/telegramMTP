package secret

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// utf8BOM is the byte order mark some Windows editors put at the start
// of a file. It is built from bytes so no literal BOM sits in this source.
var utf8BOM = string([]byte{0xEF, 0xBB, 0xBF})

// loadDotEnv reads KEY=VALUE lines. It understands blank lines, # comments,
// an optional "export " prefix and single or double quoted values. A
// missing file yields an empty map.
func loadDotEnv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("secret: open %s: %w", path, err)
	}
	defer f.Close()

	out := make(map[string]string)
	sc := bufio.NewScanner(f)
	for line := 1; sc.Scan(); line++ {
		text := strings.TrimSpace(strings.TrimPrefix(sc.Text(), utf8BOM))
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		text = strings.TrimPrefix(text, "export ")
		key, value, ok := strings.Cut(text, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("secret: %s line %d: expected KEY=VALUE", path, line)
		}
		out[key] = unquote(strings.TrimSpace(value))
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("secret: read %s: %w", path, err)
	}
	return out, nil
}

func unquote(v string) string {
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		return v[1 : len(v)-1]
	}
	// Unquoted values may carry a trailing comment: KEY=value # note
	if i := strings.Index(v, " #"); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	return v
}
