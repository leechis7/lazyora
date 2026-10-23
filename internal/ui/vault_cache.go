package ui

import (
	"lazyora/internal/credentials"
	"path/filepath"
	"runtime"
	"strings"
)

func vaultCachePath(path string) string {
	if absolute, err := filepath.Abs(path); err == nil {
		path = absolute
	}
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}
	return path
}
func (m *workspaceSet) clearVaultCache() {
	for _, store := range m.vaults {
		store.Lock()
	}
	m.vaults = nil
}
func (m *workspaceSet) retainsVault(store *credentials.Store) bool {
	for _, cached := range m.vaults {
		if cached == store {
			return true
		}
	}
	return false
}
func (s *screen) invalidateVaultCache(path string) {
	if s.workspaces == nil {
		if s.vault != nil {
			s.vault.Lock()
		}
		s.vault = nil
		return
	}
	m := s.workspaces
	store := m.vaults[path]
	if store == nil {
		store = s.vault
	}
	if store != nil {
		store.Lock()
	}
	delete(m.vaults, path)
	for _, other := range m.screens {
		if other.vault == store {
			other.vault = nil
			if other.clearConnectionSecret != nil {
				other.clearConnectionSecret()
			}
		}
	}
}
