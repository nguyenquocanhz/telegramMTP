package secret

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// newTestManager builds a Manager that sees only the given environment
// and files under a temp dir.
func newTestManager(t *testing.T, dir string, env map[string]string) *Manager {
	t.Helper()
	m, err := New(Options{
		DotEnv:     filepath.Join(dir, ".env"),
		SecretsDir: filepath.Join(dir, "run-secrets"),
		StorePath:  filepath.Join(dir, "secrets.enc"),
		lookupEnv:  func(k string) (string, bool) { v, ok := env[k]; return v, ok },
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestLookupOrder(t *testing.T) {
	dir := t.TempDir()
	secretsDir := filepath.Join(dir, "run-secrets")
	if err := os.Mkdir(secretsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	tokenFile := filepath.Join(dir, "token.txt")

	// Every source knows TOKEN; each step removes the strongest one.
	write(t, filepath.Join(dir, ".env"), "TOKEN=from-dotenv\n")
	store, err := OpenStore(filepath.Join(dir, "secrets.enc"), "master")
	if err != nil {
		t.Fatal(err)
	}
	store.Set("TOKEN", "from-store")
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(secretsDir, "token"), "from-docker-secret\n")
	write(t, tokenFile, "  from-file\n")
	env := map[string]string{"TOKEN": "from-env", "TOKEN_FILE": tokenFile, MasterKeyName: "master"}

	steps := []struct {
		want, source string
		remove       func()
	}{
		{"from-env", "env", func() { env["TOKEN"] = "" }}, // empty counts as unset
		{"from-file", "file " + tokenFile, func() { delete(env, "TOKEN_FILE") }},
		{"from-docker-secret", "file " + filepath.Join(secretsDir, "token"), func() { os.Remove(filepath.Join(secretsDir, "token")) }},
		{"from-store", "store " + filepath.Join(dir, "secrets.enc"), func() { os.Remove(filepath.Join(dir, "secrets.enc")) }},
		{"from-dotenv", ".env", func() { os.Remove(filepath.Join(dir, ".env")) }},
	}
	for _, step := range steps {
		m := newTestManager(t, dir, env)
		if got, ok := m.Get("TOKEN"); !ok || got != step.want {
			t.Errorf("Get = %q, %v; want %q", got, ok, step.want)
		}
		if got := m.Source("TOKEN"); got != step.source {
			t.Errorf("Source = %q, want %q", got, step.source)
		}
		step.remove()
	}

	m := newTestManager(t, dir, env)
	if _, ok := m.Get("TOKEN"); ok {
		t.Error("TOKEN still found after removing every source")
	}
	if _, err := m.Require("TOKEN"); err == nil || !strings.Contains(err.Error(), "TOKEN_FILE") {
		t.Errorf("Require error should list the places checked, got %v", err)
	}
}

func TestStoreSkippedWithoutMasterKey(t *testing.T) {
	dir := t.TempDir()
	store, _ := OpenStore(filepath.Join(dir, "secrets.enc"), "master")
	store.Set("TOKEN", "value")
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}

	m := newTestManager(t, dir, map[string]string{})
	_, err := m.Require("TOKEN")
	if err == nil || !strings.Contains(err.Error(), MasterKeyName) {
		t.Errorf("want a hint about %s, got %v", MasterKeyName, err)
	}

	_, err = New(Options{
		DotEnv:    filepath.Join(dir, ".env"),
		StorePath: filepath.Join(dir, "secrets.enc"),
		lookupEnv: func(k string) (string, bool) { return "wrong", k == MasterKeyName },
	})
	if !errors.Is(err, ErrWrongKey) {
		t.Errorf("wrong master key: err = %v, want ErrWrongKey", err)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.enc")
	s, err := OpenStore(path, "correct horse")
	if err != nil {
		t.Fatal(err)
	}
	s.Set("BOT_TOKEN", "123456:super-secret-token")
	s.Set("API_HASH", "0123456789abcdef")
	s.Set("GONE", "x")
	if !s.Delete("GONE") || s.Delete("GONE") {
		t.Error("Delete should report true once, then false")
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	raw, _ := os.ReadFile(path)
	if bytes.Contains(raw, []byte("super-secret-token")) || bytes.Contains(raw, []byte("BOT_TOKEN")) {
		t.Fatal("store file contains plaintext")
	}

	reopened, err := OpenStore(path, "correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := reopened.Get("BOT_TOKEN"); v != "123456:super-secret-token" {
		t.Errorf("BOT_TOKEN = %q", v)
	}
	if got := reopened.Keys(); !reflect.DeepEqual(got, []string{"API_HASH", "BOT_TOKEN"}) {
		t.Errorf("Keys = %v", got)
	}

	if _, err := OpenStore(path, "wrong"); !errors.Is(err, ErrWrongKey) {
		t.Errorf("wrong passphrase: err = %v, want ErrWrongKey", err)
	}

	// Flipping one ciphertext byte must be detected.
	tampered := bytes.Replace(raw, []byte(`"ciphertext": "`), []byte(`"ciphertext": "A`), 1)
	write(t, path, string(tampered))
	if _, err := OpenStore(path, "correct horse"); err == nil {
		t.Error("tampered store opened without error")
	}
}

func TestDotEnvParsing(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	write(t, path, strings.Join([]string{
		"# comment",
		"",
		"API_ID=12345",
		"export API_HASH = abcdef  # trailing comment",
		`BOT_TOKEN="123:AA BB#cc"`,
		"SINGLE='quoted value'",
		"EMPTY=",
	}, "\n"))

	got, err := loadDotEnv(path)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"API_ID":    "12345",
		"API_HASH":  "abcdef",
		"BOT_TOKEN": "123:AA BB#cc",
		"SINGLE":    "quoted value",
		"EMPTY":     "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("loadDotEnv = %v, want %v", got, want)
	}

	write(t, path, "not a pair\n")
	if _, err := loadDotEnv(path); err == nil {
		t.Error("malformed line: want an error")
	}
}

func TestNumbersAndMask(t *testing.T) {
	m := newTestManager(t, t.TempDir(), map[string]string{
		"API_ID":   "12345",
		"ADMIN_ID": "111, 222;333",
		"BAD":      "12x",
	})

	if n, err := m.Int64("API_ID"); err != nil || n != 12345 {
		t.Errorf("Int64 = %d, %v", n, err)
	}
	if _, err := m.Int64("BAD"); err == nil {
		t.Error("Int64(BAD): want an error")
	}
	if ids, err := m.Int64List("ADMIN_ID"); err != nil || !reflect.DeepEqual(ids, []int64{111, 222, 333}) {
		t.Errorf("Int64List = %v, %v", ids, err)
	}
	if ids, err := m.Int64List("MISSING"); err != nil || len(ids) != 0 {
		t.Errorf("Int64List(MISSING) = %v, %v", ids, err)
	}
	if _, err := m.Int64List("BAD"); err == nil {
		t.Error("Int64List(BAD): want an error")
	}

	if got := Mask("123456:ABCDEF"); got != "12****EF" || strings.Contains(got, "3456") {
		t.Errorf("Mask = %q", got)
	}
	if got := Mask("short"); got != "****" {
		t.Errorf("Mask(short) = %q", got)
	}
}
