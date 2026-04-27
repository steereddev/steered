package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	dbName        = "steered.db"
	bucketConfig  = "config"
	bucketSecrets = "secrets"

	keyLLMProvider   = "llm_provider"
	keyLLMModel      = "llm_model"
	keyLLMEndpoint   = "llm_endpoint"
	keyAPIKey        = "api_key"
	keyProbeInterval = "probe_interval"

	// default TTL for sensitive data — 24 hours
	DefaultKeyTTL = 24 * time.Hour
)

// Config holds all steered configuration
type Config struct {
	LLMProvider   string `json:"llm_provider"`
	LLMModel      string `json:"llm_model"`
	LLMEndpoint   string `json:"llm_endpoint"`
	ProbeInterval int    `json:"probe_interval"`
}

// SecretEntry wraps a secret value with expiry
type SecretEntry struct {
	Value     string    `json:"value"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Manager handles all config persistence
type Manager struct {
	db     *bolt.DB
	dbPath string
}

// New creates a new config Manager
// stores db at ~/.steered/steered.db
func New() (*Manager, error) {
	dir, err := auraDir()
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dir, dbName)

	db, err := bolt.Open(dbPath, 0600, &bolt.Options{
		Timeout: 2 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open config db: %w", err)
	}

	// create buckets if they don't exist
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(bucketConfig)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(bucketSecrets)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init buckets: %w", err)
	}

	m := &Manager{db: db, dbPath: dbPath}

	// sweep expired secrets on every start
	m.sweepExpiredSecrets()

	return m, nil
}

// Close closes the database
func (m *Manager) Close() {
	if m.db != nil {
		m.db.Close()
	}
}

// SaveConfig saves non-sensitive configuration
func (m *Manager) SaveConfig(cfg *Config) error {
	return m.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketConfig))

		set := func(key, value string) error {
			return b.Put([]byte(key), []byte(value))
		}

		if err := set(keyLLMProvider, cfg.LLMProvider); err != nil {
			return err
		}
		if err := set(keyLLMModel, cfg.LLMModel); err != nil {
			return err
		}
		if err := set(keyLLMEndpoint, cfg.LLMEndpoint); err != nil {
			return err
		}

		interval := fmt.Sprintf("%d", cfg.ProbeInterval)
		return set(keyProbeInterval, interval)
	})
}

// LoadConfig loads non-sensitive configuration
func (m *Manager) LoadConfig() (*Config, error) {
	cfg := &Config{
		LLMProvider:   "",
		LLMModel:      "",
		LLMEndpoint:   "http://localhost:11434",
		ProbeInterval: 30,
	}

	err := m.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketConfig))

		get := func(key string) string {
			v := b.Get([]byte(key))
			if v == nil {
				return ""
			}
			return string(v)
		}

		if v := get(keyLLMProvider); v != "" {
			cfg.LLMProvider = v
		}
		if v := get(keyLLMModel); v != "" {
			cfg.LLMModel = v
		}
		if v := get(keyLLMEndpoint); v != "" {
			cfg.LLMEndpoint = v
		}

		return nil
	})

	return cfg, err
}

// SaveAPIKey saves an API key with TTL expiry
func (m *Manager) SaveAPIKey(key string, ttl time.Duration) error {
	entry := SecretEntry{
		Value:     key,
		ExpiresAt: time.Now().Add(ttl),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal secret: %w", err)
	}

	return m.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketSecrets))
		return b.Put([]byte(keyAPIKey), data)
	})
}

// LoadAPIKey loads the API key if not expired
// returns empty string if expired or not found
func (m *Manager) LoadAPIKey() (string, error) {
	var entry SecretEntry

	err := m.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketSecrets))
		data := b.Get([]byte(keyAPIKey))
		if data == nil {
			return nil
		}
		return json.Unmarshal(data, &entry)
	})
	if err != nil {
		return "", err
	}

	// check expiry
	if entry.Value == "" {
		return "", nil
	}
	if time.Now().After(entry.ExpiresAt) {
		// expired — delete it
		m.deleteAPIKey()
		return "", nil
	}

	return entry.Value, nil
}

// DeleteAPIKey manually removes the API key
func (m *Manager) deleteAPIKey() {
	m.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketSecrets))
		return b.Delete([]byte(keyAPIKey))
	})
}

// ClearSecrets wipes all secrets immediately
func (m *Manager) ClearSecrets() error {
	return m.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(bucketSecrets))
	})
}

// ClearAll wipes entire config and secrets
func (m *Manager) ClearAll() error {
	m.db.Close()
	return os.Remove(m.dbPath)
}

// sweepExpiredSecrets removes all expired secrets on startup
func (m *Manager) sweepExpiredSecrets() {
	m.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketSecrets))
		if b == nil {
			return nil
		}

		var expiredKeys [][]byte

		b.ForEach(func(k, v []byte) error {
			var entry SecretEntry
			if err := json.Unmarshal(v, &entry); err != nil {
				return nil
			}
			if time.Now().After(entry.ExpiresAt) {
				expiredKeys = append(expiredKeys, k)
			}
			return nil
		})

		for _, k := range expiredKeys {
			b.Delete(k)
		}

		return nil
	})
}

// IsLLMConfigured returns true if a provider is configured
func (m *Manager) IsLLMConfigured() bool {
	cfg, err := m.LoadConfig()
	if err != nil {
		return false
	}
	return cfg.LLMProvider != ""
}

// auraDir returns ~/.steered creating it if needed
func auraDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home dir: %w", err)
	}

	dir := filepath.Join(home, ".steered")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create .steered dir: %w", err)
	}

	return dir, nil
}
