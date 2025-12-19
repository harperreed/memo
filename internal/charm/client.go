// ABOUTME: Charm KV client wrapper for memo data storage
// ABOUTME: Provides thread-safe initialization and automatic sync

package charm

import (
	"fmt"
	"os"
	"sync"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/kv"
	charmproto "github.com/charmbracelet/charm/proto"
)

const (
	// DBName is the name of the charm kv database for memo.
	DBName = "memo"
)

var (
	globalClient *Client
	clientOnce   sync.Once
	clientErr    error
)

// Client wraps charm KV with sync support.
type Client struct {
	kv     *kv.KV
	config *Config
}

// InitClient initializes the global charm client (thread-safe, idempotent).
func InitClient() error {
	clientOnce.Do(func() {
		cfg, err := LoadConfig()
		if err != nil {
			clientErr = err
			return
		}

		// Set charm host before opening KV
		if cfg.CharmHost != "" {
			if err := os.Setenv("CHARM_HOST", cfg.CharmHost); err != nil {
				clientErr = err
				return
			}
		}

		db, err := kv.OpenWithDefaultsFallback(DBName)
		if err != nil {
			clientErr = err
			return
		}

		globalClient = &Client{
			kv:     db,
			config: cfg,
		}

		// Sync on startup to pull remote data (skip in read-only mode)
		if cfg.AutoSync && !db.IsReadOnly() {
			_ = db.Sync() // Best effort
		}
	})
	return clientErr
}

// GetClient returns the global client, initializing if needed.
func GetClient() (*Client, error) {
	if err := InitClient(); err != nil {
		return nil, err
	}
	return globalClient, nil
}

// Close releases client resources.
func (c *Client) Close() error {
	if c.kv != nil {
		return c.kv.Close()
	}
	return nil
}

// KV returns the underlying charm kv store.
func (c *Client) KV() *kv.KV {
	return c.kv
}

// IsReadOnly returns true if the database is open in read-only mode.
// This happens when another process (like an MCP server) holds the lock.
func (c *Client) IsReadOnly() bool {
	return c.kv.IsReadOnly()
}

// Sync triggers a manual sync with the charm server.
func (c *Client) Sync() error {
	if c.kv.IsReadOnly() {
		return nil // Skip sync in read-only mode
	}
	return c.kv.Sync()
}

// syncIfEnabled syncs if auto_sync is enabled.
func (c *Client) syncIfEnabled() {
	if c.config.AutoSync && !c.kv.IsReadOnly() {
		_ = c.kv.Sync() // Best effort
	}
}

// ID returns the charm user ID for this device.
func (c *Client) ID() (string, error) {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return "", err
	}
	return cc.ID()
}

// Config returns the current configuration.
func (c *Client) Config() *Config {
	return c.config
}

// Set stores a value with the given key and syncs.
func (c *Client) Set(key, value []byte) error {
	if c.kv.IsReadOnly() {
		return fmt.Errorf("cannot write: database is locked by another process (MCP server?)")
	}
	if err := c.kv.Set(key, value); err != nil {
		return err
	}
	c.syncIfEnabled()
	return nil
}

// Get retrieves a value by key.
func (c *Client) Get(key []byte) ([]byte, error) {
	return c.kv.Get(key)
}

// Delete removes a key and syncs.
func (c *Client) Delete(key []byte) error {
	if c.kv.IsReadOnly() {
		return fmt.Errorf("cannot write: database is locked by another process (MCP server?)")
	}
	if err := c.kv.Delete(key); err != nil {
		return err
	}
	c.syncIfEnabled()
	return nil
}

// Reset clears all data (nuclear option).
func (c *Client) Reset() error {
	if c.kv.IsReadOnly() {
		return fmt.Errorf("cannot write: database is locked by another process (MCP server?)")
	}
	return c.kv.Reset()
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
	// The charm client handles the linking flow automatically
	// when we try to access user info
	_, err = cc.Bio()
	return err
}

// Unlink removes the charm account association from this device
// This clears the local KV data; the SSH key remains but data is detached.
func (c *Client) Unlink() error {
	if c.kv.IsReadOnly() {
		return fmt.Errorf("cannot write: database is locked by another process (MCP server?)")
	}
	// Reset the KV store to clear all data
	if err := c.kv.Reset(); err != nil {
		return err
	}
	// Close the current connection
	return c.Close()
}

// ResetClient resets the global client singleton, allowing reinitialization.
func ResetClient() error {
	if globalClient != nil {
		if err := globalClient.Close(); err != nil {
			return err
		}
		globalClient = nil
	}
	// Reset the once so next GetClient will reinitialize
	clientOnce = sync.Once{}
	clientErr = nil
	return nil
}
