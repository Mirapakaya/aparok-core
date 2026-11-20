package anystoreprovider

import (
	"crypto/sha256"
	"sync"
)

var (
	storageKeyLock sync.Mutex
	storageKey     []byte
)

// SetStorageKey sets the 32-byte encryption key used for all any-store databases.
// If the provided key is not 32 bytes, it is hashed with SHA-256 to derive a 32-byte key.
func SetStorageKey(key []byte) {
	if len(key) == 0 {
		return
	}
	storageKeyLock.Lock()
	defer storageKeyLock.Unlock()
	storageKey = deriveKey(key)
}

// GetStorageKey returns the currently configured storage key, or nil if not set.
func GetStorageKey() []byte {
	storageKeyLock.Lock()
	defer storageKeyLock.Unlock()
	return append([]byte(nil), storageKey...)
}

func deriveKey(key []byte) []byte {
	if len(key) == 32 {
		return append([]byte(nil), key...)
	}
	h := sha256.Sum256(key)
	return h[:]
}
