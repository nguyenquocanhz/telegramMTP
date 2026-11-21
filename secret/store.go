package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

const (
	storeVersion    = 1
	storeIterations = 600_000
	storeSaltSize   = 16
	storeKeySize    = 32 // AES-256
)

// storeAAD ties a ciphertext to this file format.
var storeAAD = []byte("telegramMTP/secret/v1")

// ErrWrongKey means the master key did not decrypt the store (or the
// file was tampered with; AES-GCM cannot tell the two apart).
var ErrWrongKey = errors.New("secret: wrong master key or corrupted store")

// storeFile is the on-disk JSON layout. Only Ciphertext is secret.
type storeFile struct {
	Version    int    `json:"version"`
	KDF        string `json:"kdf"`
	Iterations int    `json:"iterations"`
	Salt       []byte `json:"salt"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

// Store is a small key/value file encrypted with AES-256-GCM. The key is
// derived from a passphrase with PBKDF2-HMAC-SHA256, so the file is safe
// to keep next to the binary or in a backup as long as the passphrase is
// kept somewhere else.
type Store struct {
	path       string
	passphrase string
	values     map[string]string
}

// OpenStore decrypts the store at path. A missing file gives an empty
// store that is created on the first Save.
func OpenStore(path, passphrase string) (*Store, error) {
	if passphrase == "" {
		return nil, errors.New("secret: master key is empty")
	}
	s := &Store{path: path, passphrase: passphrase, values: make(map[string]string)}

	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("secret: read store: %w", err)
	}

	var f storeFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("secret: %s is not a secret store: %w", path, err)
	}
	if f.Version != storeVersion || f.KDF != "pbkdf2-sha256" || f.Iterations <= 0 {
		return nil, fmt.Errorf("secret: %s has an unsupported format (version %d, kdf %q)", path, f.Version, f.KDF)
	}
	aead, err := newAEAD(passphrase, f.Salt, f.Iterations)
	if err != nil {
		return nil, err
	}
	if len(f.Nonce) != aead.NonceSize() {
		return nil, ErrWrongKey
	}
	plain, err := aead.Open(nil, f.Nonce, f.Ciphertext, storeAAD)
	if err != nil {
		return nil, ErrWrongKey
	}
	if err := json.Unmarshal(plain, &s.values); err != nil {
		return nil, fmt.Errorf("secret: store content: %w", err)
	}
	return s, nil
}

func newAEAD(passphrase string, salt []byte, iterations int) (cipher.AEAD, error) {
	key, err := pbkdf2.Key(sha256.New, passphrase, salt, iterations, storeKeySize)
	if err != nil {
		return nil, fmt.Errorf("secret: derive key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("secret: cipher: %w", err)
	}
	return cipher.NewGCM(block)
}

// Get returns a stored value.
func (s *Store) Get(key string) (string, bool) {
	v, ok := s.values[key]
	return v, ok
}

// Set stores a value in memory; call Save to persist it.
func (s *Store) Set(key, value string) { s.values[key] = value }

// Delete removes a key and reports whether it existed.
func (s *Store) Delete(key string) bool {
	_, ok := s.values[key]
	delete(s.values, key)
	return ok
}

// Keys lists the stored keys in alphabetical order.
func (s *Store) Keys() []string {
	keys := make([]string, 0, len(s.values))
	for k := range s.values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Save encrypts and writes the store with a fresh salt and nonce. The
// file is replaced atomically so a crash cannot leave it half written.
func (s *Store) Save() error {
	plain, err := json.Marshal(s.values)
	if err != nil {
		return err
	}
	salt := make([]byte, storeSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	aead, err := newAEAD(s.passphrase, salt, storeIterations)
	if err != nil {
		return err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(storeFile{
		Version:    storeVersion,
		KDF:        "pbkdf2-sha256",
		Iterations: storeIterations,
		Salt:       salt,
		Nonce:      nonce,
		Ciphertext: aead.Seal(nil, nonce, plain, storeAAD),
	}, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".secrets-*.tmp")
	if err != nil {
		return fmt.Errorf("secret: write store: %w", err)
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		tmp.Close()
		return fmt.Errorf("secret: write store: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("secret: write store: %w", err)
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return fmt.Errorf("secret: write store: %w", err)
	}
	if err := os.Rename(tmp.Name(), s.path); err != nil {
		return fmt.Errorf("secret: write store: %w", err)
	}
	return nil
}
