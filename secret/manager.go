// Package secret loads credentials without hard-coding where they live.
//
// A key such as BOT_TOKEN is looked up in this order, first hit wins:
//
//  1. the environment variable BOT_TOKEN
//  2. the file named by the environment variable BOT_TOKEN_FILE
//  3. a Docker/Kubernetes secret file: /run/secrets/bot_token
//  4. the encrypted store (secrets.enc), if SECRET_MASTER_KEY is available
//  5. the .env file
//
// Empty values count as "not set" so a blank line in a compose file does
// not shadow a real secret further down the list.
package secret

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// MasterKeyName is the key holding the passphrase of the encrypted store.
// It is resolved through steps 1-3 and 5 above (never from the store).
const MasterKeyName = "SECRET_MASTER_KEY"

// Defaults used when Options leaves a field empty.
const (
	DefaultDotEnv     = ".env"
	DefaultSecretsDir = "/run/secrets"
	DefaultStorePath  = "secrets.enc"
)

// Options configures a Manager. The zero value uses the defaults.
type Options struct {
	// DotEnv is the path of the .env file. A missing file is not an error.
	DotEnv string
	// SecretsDir is where Docker/Kubernetes mount secret files.
	SecretsDir string
	// StorePath is the encrypted store. It defaults to the SECRET_STORE
	// environment variable, then "secrets.enc". A missing file is fine.
	StorePath string

	// lookupEnv replaces os.LookupEnv in tests.
	lookupEnv func(string) (string, bool)
}

// Manager resolves secrets from the configured sources.
type Manager struct {
	lookupEnv  func(string) (string, bool)
	dotenv     map[string]string
	secretsDir string
	storePath  string
	store      *Store
	// storeSkipped is true when a store file exists but could not be
	// opened because no master key was found.
	storeSkipped bool
}

// New builds a Manager. It fails only when a source exists but is
// broken: an unreadable .env, or a store that the master key cannot open.
func New(opts Options) (*Manager, error) {
	m := &Manager{
		lookupEnv:  opts.lookupEnv,
		secretsDir: opts.SecretsDir,
		storePath:  opts.StorePath,
	}
	if m.lookupEnv == nil {
		m.lookupEnv = os.LookupEnv
	}
	if m.secretsDir == "" {
		m.secretsDir = DefaultSecretsDir
	}
	if opts.DotEnv == "" {
		opts.DotEnv = DefaultDotEnv
	}

	dotenv, err := loadDotEnv(opts.DotEnv)
	if err != nil {
		return nil, err
	}
	m.dotenv = dotenv

	if m.storePath == "" {
		m.storePath, _ = m.lookup("SECRET_STORE")
	}
	if m.storePath == "" {
		m.storePath = DefaultStorePath
	}
	if _, err := os.Stat(m.storePath); err == nil {
		master, _ := m.lookup(MasterKeyName)
		if master == "" {
			m.storeSkipped = true
		} else {
			store, err := OpenStore(m.storePath, master)
			if err != nil {
				return nil, err
			}
			m.store = store
		}
	}
	return m, nil
}

// lookup resolves key and reports which source supplied it.
func (m *Manager) lookup(key string) (value, source string) {
	if v, ok := m.lookupEnv(key); ok && v != "" {
		return v, "env"
	}
	if path, ok := m.lookupEnv(key + "_FILE"); ok && path != "" {
		if v := readTrimmed(path); v != "" {
			return v, "file " + path
		}
	}
	path := filepath.Join(m.secretsDir, strings.ToLower(key))
	if v := readTrimmed(path); v != "" {
		return v, "file " + path
	}
	if m.store != nil {
		if v, ok := m.store.Get(key); ok && v != "" {
			return v, "store " + m.storePath
		}
	}
	if v := m.dotenv[key]; v != "" {
		return v, ".env"
	}
	return "", ""
}

func readTrimmed(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// Get returns the secret and whether it was found.
func (m *Manager) Get(key string) (string, bool) {
	v, _ := m.lookup(key)
	return v, v != ""
}

// GetOr returns the secret, or fallback when it is not set.
func (m *Manager) GetOr(key, fallback string) string {
	if v, ok := m.Get(key); ok {
		return v
	}
	return fallback
}

// Source says where key was found ("env", ".env", "file <path>",
// "store <path>") or "" when it is not set. Safe to log.
func (m *Manager) Source(key string) string {
	_, source := m.lookup(key)
	return source
}

// StorePath is the encrypted store file this Manager uses.
func (m *Manager) StorePath() string { return m.storePath }

// Require returns the secret or an error naming every place that was
// checked, so a missing credential is quick to fix.
func (m *Manager) Require(key string) (string, error) {
	if v, ok := m.Get(key); ok {
		return v, nil
	}
	msg := fmt.Sprintf("secret %s is not set (checked env %s, env %s_FILE, %s, .env)",
		key, key, key, filepath.ToSlash(filepath.Join(m.secretsDir, strings.ToLower(key))))
	if m.store != nil {
		msg += "; also not in " + m.storePath
	}
	if m.storeSkipped {
		msg += fmt.Sprintf("; %s was skipped because %s is not set", m.storePath, MasterKeyName)
	}
	return "", errors.New(msg)
}

// Int64 returns a required numeric secret such as API_ID.
func (m *Manager) Int64(key string) (int64, error) {
	v, err := m.Require(key)
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("secret %s must be a whole number", key)
	}
	return n, nil
}

// Int64List returns an optional list of numbers separated by commas or
// spaces, such as ADMIN_ID=111,222. An unset key gives an empty list.
func (m *Manager) Int64List(key string) ([]int64, error) {
	v, ok := m.Get(key)
	if !ok {
		return nil, nil
	}
	fields := strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ';' || r == ' ' })
	out := make([]int64, 0, len(fields))
	for _, f := range fields {
		n, err := strconv.ParseInt(f, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("secret %s: %q is not a whole number", key, f)
		}
		out = append(out, n)
	}
	return out, nil
}

// Mask hides a secret for logs, keeping just enough to tell two apart.
func Mask(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}
