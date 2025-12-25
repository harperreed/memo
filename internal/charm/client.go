// ABOUTME: Charm KV client wrapper using transactional Do API
// ABOUTME: Short-lived connections to avoid lock contention with other MCP servers

package charm

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/kv"
	charmproto "github.com/charmbracelet/charm/proto"
)

const (
	// DBName is the name of the charm kv database for memo.
	DBName = "memo"
)

// Client holds configuration for KV operations.
// Unlike the previous implementation, it does NOT hold a persistent connection.
// Each operation opens the database, performs the operation, and closes it.
type Client struct {
	dbName         string
	autoSync       bool
	staleThreshold time.Duration
}

// Option configures a Client.
type Option func(*Client)

// WithDBName sets the database name.
func WithDBName(name string) Option {
	return func(c *Client) {
		c.dbName = name
	}
}

// WithAutoSync enables or disables auto-sync after writes.
func WithAutoSync(enabled bool) Option {
	return func(c *Client) {
		c.autoSync = enabled
	}
}

// NewClient creates a new client with the given options.
func NewClient(opts ...Option) (*Client, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	// Set charm host if configured
	if cfg.CharmHost != "" {
		if err := os.Setenv("CHARM_HOST", cfg.CharmHost); err != nil {
			return nil, err
		}
	}

	c := &Client{
		dbName:         DBName,
		autoSync:       cfg.AutoSync,
		staleThreshold: cfg.StaleThreshold,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// Get retrieves a value by key (read-only, no lock contention).
func (c *Client) Get(key []byte) ([]byte, error) {
	if err := c.SyncIfStale(); err != nil {
		return nil, err
	}
	var val []byte
	err := kv.DoReadOnly(c.dbName, func(k *kv.KV) error {
		var err error
		val, err = k.Get(key)
		return err
	})
	return val, err
}

// Set stores a value with the given key.
func (c *Client) Set(key, value []byte) error {
	return kv.Do(c.dbName, func(k *kv.KV) error {
		if err := k.Set(key, value); err != nil {
			return err
		}
		if c.autoSync {
			return k.Sync()
		}
		return nil
	})
}

// Delete removes a key.
func (c *Client) Delete(key []byte) error {
	return kv.Do(c.dbName, func(k *kv.KV) error {
		if err := k.Delete(key); err != nil {
			return err
		}
		if c.autoSync {
			return k.Sync()
		}
		return nil
	})
}

// Keys returns all keys in the database.
func (c *Client) Keys() ([][]byte, error) {
	if err := c.SyncIfStale(); err != nil {
		return nil, err
	}
	var keys [][]byte
	err := kv.DoReadOnly(c.dbName, func(k *kv.KV) error {
		var err error
		keys, err = k.Keys()
		return err
	})
	return keys, err
}

// DoReadOnly executes a function with read-only database access.
// Use this for batch read operations that need multiple Gets.
func (c *Client) DoReadOnly(fn func(k *kv.KV) error) error {
	if err := c.SyncIfStale(); err != nil {
		return err
	}
	return kv.DoReadOnly(c.dbName, fn)
}

// Do executes a function with write access to the database.
// Use this for batch write operations.
func (c *Client) Do(fn func(k *kv.KV) error) error {
	return kv.Do(c.dbName, func(k *kv.KV) error {
		if err := fn(k); err != nil {
			return err
		}
		if c.autoSync {
			return k.Sync()
		}
		return nil
	})
}

// Sync triggers a manual sync with the charm server.
func (c *Client) Sync() error {
	return kv.Do(c.dbName, func(k *kv.KV) error {
		return k.Sync()
	})
}

// LastSyncTime returns the timestamp of the last sync operation.
func (c *Client) LastSyncTime() time.Time {
	var lastSync time.Time
	_ = kv.DoReadOnly(c.dbName, func(k *kv.KV) error {
		lastSync = k.LastSyncTime()
		return nil
	})
	return lastSync
}

// IsStale checks if the data is stale based on the configured threshold.
func (c *Client) IsStale() bool {
	if c.staleThreshold == 0 {
		return false
	}
	var isStale bool
	_ = kv.DoReadOnly(c.dbName, func(k *kv.KV) error {
		isStale = k.IsStale(c.staleThreshold)
		return nil
	})
	return isStale
}

// SyncIfStale syncs with the charm server if data is stale.
func (c *Client) SyncIfStale() error {
	if !c.IsStale() {
		return nil
	}
	fmt.Fprintf(os.Stderr, "Data stale (last sync > %v ago), syncing...\n", c.staleThreshold)
	return c.Sync()
}

// Reset clears all data (nuclear option).
func (c *Client) Reset() error {
	return kv.Do(c.dbName, func(k *kv.KV) error {
		return k.Reset()
	})
}

// ID returns the charm user ID for this device.
func (c *Client) ID() (string, error) {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return "", err
	}
	return cc.ID()
}

// User returns the current charm user information.
func (c *Client) User() (*charmproto.User, error) {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return nil, err
	}
	return cc.Bio()
}

// Link initiates the charm linking process for this device.
func (c *Client) Link() error {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return err
	}
	_, err = cc.Bio()
	return err
}

// Unlink removes the charm account association from this device.
func (c *Client) Unlink() error {
	return c.Reset()
}

// Config returns the current configuration.
func (c *Client) Config() *Config {
	cfg, _ := LoadConfig()
	return cfg
}

// --- Legacy compatibility layer ---
// These functions maintain backwards compatibility with existing code.

var globalClient *Client

// InitClient initializes the global charm client.
// With the new architecture, this just creates a Client instance.
func InitClient() error {
	if globalClient != nil {
		return nil
	}
	var err error
	globalClient, err = NewClient()
	return err
}

// GetClient returns the global client, initializing if needed.
func GetClient() (*Client, error) {
	if err := InitClient(); err != nil {
		return nil, err
	}
	return globalClient, nil
}

// ResetClient resets the global client singleton.
func ResetClient() error {
	globalClient = nil
	return nil
}

// Close is a no-op for backwards compatibility.
// With Do API, connections are automatically closed after each operation.
func (c *Client) Close() error {
	return nil
}
