package state

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// a cache entry maps a cache key to the layer that was produced
// eg -> key: sha256:abc, layerDigest: sha256:xyz, createdBy: "RUN npm install"
type CacheEntry struct {
	Key       string `json:"key"`
	Layer     Layer  `json:"layer"`
}

// getCacheDir returns the path to the cache dir
// eg -> ~/.docksmith/cache/
func getCacheDir() string {
	return filepath.Join(getBaseDir(), "cache")
}

// ComputeCacheKey hashes together the previous layer digest + instruction type + args
// this means if anything in the chain changes, the key changes too
// eg -> sha256(sha256:prevdigest + "RUN" + "npm install")
func ComputeCacheKey(prevDigest string, instType string, args []string) string {
	h := sha256.New()
	h.Write([]byte(prevDigest))
	h.Write([]byte(instType))
	for _, arg := range args {
		h.Write([]byte(arg))
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}

// GetCacheEntry looks up a cache key and returns the stored layer if it exists
// returns nil if there is no cache entry for that key
func GetCacheEntry(key string) *CacheEntry {
	path := filepath.Join(getCacheDir(), key+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		// file doesnt exist -> cache miss
		return nil
	}
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// corrupted cache entry -> treat as miss
		return nil
	}
	return &entry
}

// SetCacheEntry writes a cache entry to disk
// called after a successful instruction execution
func SetCacheEntry(key string, layer Layer) error {
	entry := CacheEntry{
		Key:   key,
		Layer: layer,
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(getCacheDir(), key+".json")
	return os.WriteFile(path, data, 0644)
}
