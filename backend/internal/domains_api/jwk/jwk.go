package jwk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/rs/zerolog/log"
)

type CognitoJWK struct {
	keySet     jwk.Set
	Region     string
	PoolID     string
	ClientID   string
	updateLock sync.RWMutex
	stopChan   chan struct{} // Channel to stop the refresh goroutine
}

func NewCognitoJWK(region, poolID, clientID string) (*CognitoJWK, error) {
	c := &CognitoJWK{
		Region:   region,
		PoolID:   poolID,
		ClientID: clientID,
		stopChan: make(chan struct{}),
	}

	// Initial load from file
	if err := c.loadFromFile(); err != nil {
		return nil, err
	}

	// Start the periodic refresh
	go c.startPeriodicRefresh()

	return c, nil
}

func (c *CognitoJWK) loadFromFile() error {
	jwksPath := filepath.Join("internal", "domains_api", "jwk", "jwks.json")
	jwksBytes, err := os.ReadFile(jwksPath)
	if err != nil {
		return fmt.Errorf("failed to read JWKS file: %w", err)
	}

	keySet, err := jwk.Parse(jwksBytes)
	if err != nil {
		return fmt.Errorf("failed to parse JWKS: %w", err)
	}

	c.updateLock.Lock()
	c.keySet = keySet
	c.updateLock.Unlock()

	return nil
}

func (c *CognitoJWK) refreshKeys() error {
	jwksURL := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json",
		c.Region, c.PoolID)

	resp, err := http.Get(jwksURL)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch JWKS: status code %d", resp.StatusCode)
	}

	// Parse the new key set
	newKeySet, err := jwk.ParseReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to parse JWKS: %w", err)
	}

	// Save to file
	jwksBytes, err := json.MarshalIndent(newKeySet, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JWKS: %w", err)
	}

	jwksPath := filepath.Join("internal", "domains_api", "jwk", "jwks.json")
	if err := os.WriteFile(jwksPath, jwksBytes, 0644); err != nil {
		return fmt.Errorf("failed to save JWKS file: %w", err)
	}

	// Update the in-memory key set
	c.updateLock.Lock()
	c.keySet = newKeySet
	c.updateLock.Unlock()

	log.Info().Msg("JWKS refreshed successfully")
	return nil
}

func (c *CognitoJWK) startPeriodicRefresh() {
	ticker := time.NewTicker(12 * time.Hour) // Refresh every 12 hours
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.refreshKeys(); err != nil {
				log.Error().Err(err).Msg("Failed to refresh JWKS")
			}
		case <-c.stopChan:
			return
		}
	}
}

// Stop stops the periodic refresh
func (c *CognitoJWK) Stop() {
	close(c.stopChan)
}

func (c *CognitoJWK) GetKeySet() jwk.Set {
	c.updateLock.RLock()
	defer c.updateLock.RUnlock()
	return c.keySet
}

func (c *CognitoJWK) GetIssuer() string {
	return fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", c.Region, c.PoolID)
}
