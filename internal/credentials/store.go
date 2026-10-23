// Package credentials stores opt-in connection passwords in an authenticated,
// encrypted file. A master password is never persisted. Store is not concurrent;
// the UI serializes operations and keeps password derivation off its event loop.
package credentials

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

var (
	ErrLocked      = errors.New("password store is locked")
	ErrInvalid     = errors.New("invalid or unsupported password store")
	ErrUnlock      = errors.New("incorrect master password or damaged password store")
	ErrChanged     = errors.New("password store changed; lock and reopen it before retrying")
	ErrBusy        = errors.New("password store is being updated by another instance")
	ErrWeak        = errors.New("master password must contain at least 8 characters and at most 1024 bytes")
	ErrPermissions = errors.New("password store must be a private regular file (permissions 0600)")
)

const maxFile = 2 * 1024 * 1024
const memoryKiB, iterations, parallelism = 64 * 1024, 3, 2

type envelope struct {
	Version     int    `json:"version"`
	KDF         string `json:"kdf"`
	Memory      uint32 `json:"memory_kib"`
	Iterations  uint32 `json:"iterations"`
	Parallelism uint8  `json:"parallelism"`
	Salt        []byte `json:"salt"`
	Ciphertext  []byte `json:"ciphertext"`
}

type record struct {
	Binding  string `json:"binding"`
	Password string `json:"password"`
}
type Store struct {
	path   string
	raw    []byte
	exists bool
	env    envelope
	key    []byte
}

// Path avoids overwriting either profiles or preferences even with custom names.
func Path(settingsPath, profilesPath string) string {
	if settingsPath == "" {
		return ""
	}
	path := filepath.Join(filepath.Dir(settingsPath), "credentials.enc")
	for strings.EqualFold(path, filepath.Clean(settingsPath)) || strings.EqualFold(path, filepath.Clean(profilesPath)) {
		path = filepath.Join(filepath.Dir(path), "secret-"+filepath.Base(path))
	}
	return path
}

func Open(path string) (*Store, error) {
	s := &Store{path: path}
	raw, err := read(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	s.raw, s.exists = raw, true
	if err = json.Unmarshal(raw, &s.env); err != nil || !valid(s.env) {
		return s, ErrInvalid
	}
	return s, nil
}

func valid(e envelope) bool {
	return e.Version == 1 && e.KDF == "argon2id" && e.Memory == memoryKiB && e.Iterations == iterations && e.Parallelism == parallelism && len(e.Salt) == 16 && len(e.Ciphertext) >= 28
}

func read(path string) ([]byte, error) {
	if path == "" {
		return nil, ErrInvalid
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || (runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0) {
		return nil, ErrPermissions
	}
	if info.Size() > maxFile {
		return nil, ErrInvalid
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxFile+1))
	if len(data) > maxFile {
		return nil, ErrInvalid
	}
	return data, err
}

// ValidateCurrent invalidates cached key material if another process replaced
// the encrypted store. No master password or key is exported to the UI.
func (s *Store) ValidateCurrent() error {
	if err := s.checkCurrent(); err != nil {
		s.Lock()
		return err
	}
	return nil
}
func (s *Store) Exists() bool   { return s != nil && s.exists }
func (s *Store) Unlocked() bool { return s != nil && len(s.key) == 32 }
func (s *Store) Lock() {
	if s != nil {
		clear(s.key)
		s.key = nil
	}
}
func wipe(data []byte) { clear(data) }
func derive(master []byte, salt []byte) []byte {
	return argon2.IDKey(master, salt, iterations, memoryKiB, parallelism, 32)
}
func CheckMaster(master []byte) error {
	if len(master) > 1024 || !utf8.Valid(master) || utf8.RuneCount(master) < 8 || len(bytes.TrimSpace(master)) == 0 {
		return ErrWeak
	}
	return nil
}

func aead(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCMWithRandomNonce(block)
}
func (s *Store) decrypt(key []byte) (map[string]record, error) {
	if !valid(s.env) {
		return nil, ErrInvalid
	}
	g, err := aead(key)
	if err != nil {
		return nil, err
	}
	plain, err := g.Open(nil, nil, s.env.Ciphertext, []byte("LazyOra credentials v1"))
	if err != nil {
		return nil, ErrUnlock
	}
	defer wipe(plain)
	var records map[string]record
	if json.Unmarshal(plain, &records) != nil || records == nil || len(records) > 1000 {
		return nil, ErrInvalid
	}
	return records, nil
}

// Unlock validates by authenticated decryption, without a separate password hash.
func (s *Store) Unlock(master []byte) error {
	s.Lock()
	if err := s.checkCurrent(); err != nil {
		return err
	}
	if !s.exists || !valid(s.env) {
		return ErrInvalid
	}
	if len(master) > 1024 {
		return ErrUnlock
	}
	key := derive(master, s.env.Salt)
	if _, err := s.decrypt(key); err != nil {
		wipe(key)
		return err
	}
	s.key = key
	return nil
}

func (s *Store) Create(master []byte) error {
	if s.exists {
		return ErrChanged
	}
	if err := CheckMaster(master); err != nil {
		return err
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	key := derive(master, salt)
	defer wipe(key)
	return s.persist(key, salt, map[string]record{})
}
func (s *Store) Get(id, binding string) (string, bool, error) {
	if !s.Unlocked() {
		return "", false, ErrLocked
	}
	if err := s.checkCurrent(); err != nil {
		return "", false, err
	}
	records, err := s.decrypt(s.key)
	if err != nil {
		return "", false, err
	}
	r, ok := records[id]
	if !ok || r.Binding != binding {
		return "", false, nil
	}
	return r.Password, true, nil
}
func (s *Store) Put(id, binding, password string) error {
	if !s.Unlocked() {
		return ErrLocked
	}
	if id == "" || binding == "" || password == "" || len(password) > 65536 {
		return ErrInvalid
	}
	records, err := s.decrypt(s.key)
	if err != nil {
		return err
	}
	records[id] = record{binding, password}
	if len(records) > 1000 {
		return ErrInvalid
	}
	return s.persist(s.key, s.env.Salt, records)
}
func (s *Store) Delete(id string) error {
	if !s.Unlocked() {
		return ErrLocked
	}
	records, err := s.decrypt(s.key)
	if err != nil {
		return err
	}
	delete(records, id)
	return s.persist(s.key, s.env.Salt, records)
}
func (s *Store) ChangeMaster(master []byte) error {
	if !s.Unlocked() {
		return ErrLocked
	}
	if err := CheckMaster(master); err != nil {
		return err
	}
	records, err := s.decrypt(s.key)
	if err != nil {
		return err
	}
	salt := make([]byte, 16)
	if _, err = rand.Read(salt); err != nil {
		return err
	}
	key := derive(master, salt)
	defer wipe(key)
	return s.persist(key, salt, records)
}

// writeLock serializes writers; comparison with the loaded snapshot also
// prevents a second, already-unlocked process from losing another user's update.
func (s *Store) checkCurrent() error {
	current, err := read(s.path)
	if errors.Is(err, os.ErrNotExist) && !s.exists {
		return nil
	}
	if err != nil {
		return ErrChanged
	}
	if !s.exists || !bytes.Equal(current, s.raw) {
		return ErrChanged
	}
	return nil
}

func (s *Store) writeLock() (func(), error) {
	if s.path == "" {
		return nil, ErrInvalid
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(s.path+".lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if errors.Is(err, os.ErrExist) {
		return nil, ErrBusy
	}
	if err != nil {
		return nil, err
	}
	f.Close()
	release := func() { os.Remove(s.path + ".lock") }
	current, err := read(s.path)
	if !(errors.Is(err, os.ErrNotExist) && !s.exists) && (err != nil || !s.exists || !bytes.Equal(current, s.raw)) {
		release()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return nil, ErrChanged
	}
	return release, nil
}
func (s *Store) persist(key, salt []byte, records map[string]record) error {
	plain, err := json.Marshal(records)
	if err != nil {
		return err
	}
	defer wipe(plain)
	g, err := aead(key)
	if err != nil {
		return err
	}
	e := envelope{Version: 1, KDF: "argon2id", Memory: memoryKiB, Iterations: iterations, Parallelism: parallelism, Salt: salt, Ciphertext: g.Seal(nil, nil, plain, []byte("LazyOra credentials v1"))}
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if len(data) > maxFile {
		return ErrInvalid
	}
	release, err := s.writeLock()
	if err != nil {
		return err
	}
	defer release()
	file, err := os.CreateTemp(filepath.Dir(s.path), ".credentials-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(file.Name(), s.path); err != nil {
		return err
	}
	nextKey := bytes.Clone(key)
	s.Lock()
	s.key = nextKey
	s.env = e
	s.raw = data
	s.exists = true
	return nil
}

// Reset deliberately needs no master password, but must be explicitly confirmed
// by the caller. It deletes only saved passwords, preserving connection profiles.
func (s *Store) Reset() error {
	release, err := s.writeLock()
	if err != nil {
		return err
	}
	defer release()
	if err = os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	s.Lock()
	s.raw = nil
	s.env = envelope{}
	s.exists = false
	return nil
}
