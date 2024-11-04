package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/rs/zerolog/log"
)

type contextKey string
const jwtContextKey contextKey = "jwt"

type CognitoMiddleware struct {
	jwksURL  string
	clientID string
	issuer   string
	keyCache struct {
			keys jwk.Set
			mu   sync.RWMutex
			exp  time.Time
	}
}

func NewCognitoMiddleware(region, userPoolID, clientID string) *CognitoMiddleware {
	return &CognitoMiddleware{
		jwksURL:  fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", region, userPoolID),
		clientID: clientID,
		issuer:   fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", region, userPoolID),
	}
}

func (cm *CognitoMiddleware) refreshKeys() error {
	cm.keyCache.mu.Lock()
	defer cm.keyCache.mu.Unlock()

	// Check if keys are still valid (cache for 24 hours)
	if cm.keyCache.keys != nil && time.Now().Before(cm.keyCache.exp) {
		return nil
	}

	// Fetch new keys
	keySet, err := jwk.Fetch(context.Background(), cm.jwksURL)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	cm.keyCache.keys = keySet
	cm.keyCache.exp = time.Now().Add(24 * time.Hour)
	return nil
}

func (cm *CognitoMiddleware) getKeySet() (jwk.Set, error) {
	cm.keyCache.mu.RLock()
	if cm.keyCache.keys != nil && time.Now().Before(cm.keyCache.exp) {
		defer cm.keyCache.mu.RUnlock()
		return cm.keyCache.keys, nil
	}
	cm.keyCache.mu.RUnlock()

	if err := cm.refreshKeys(); err != nil {
		return nil, err
	}

	cm.keyCache.mu.RLock()
	defer cm.keyCache.mu.RUnlock()
	return cm.keyCache.keys, nil
}

func (cm *CognitoMiddleware) Verify() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				http.Error(w, "Invalid token format", http.StatusUnauthorized)
				return
			}

			// Verify JWT structure
			parts := strings.Split(tokenString, ".")
			if len(parts) != 3 {
				http.Error(w, "Invalid JWT format", http.StatusUnauthorized)
				return
			}

			// Get current key set
			keySet, err := cm.getKeySet()
			if err != nil {
				log.Error().Err(err).Msg("Failed to get key set")
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Parse and validate the token
			token, err := jwt.Parse(
				[]byte(tokenString),
				jwt.WithKeySet(keySet),
				jwt.WithValidate(true),
				jwt.WithIssuer(cm.issuer),
			)
			if err != nil {
				// If token validation fails, try refreshing keys once
				if err := cm.refreshKeys(); err != nil {
					log.Error().Err(err).Msg("Failed to refresh keys")
					http.Error(w, "Invalid token", http.StatusUnauthorized)
					return
				}

				// Try parsing again with new keys
				keySet, _ = cm.getKeySet()
				token, err = jwt.Parse(
					[]byte(tokenString),
					jwt.WithKeySet(keySet),
					jwt.WithValidate(true),
					jwt.WithIssuer(cm.issuer),
				)
				if err != nil {
					log.Error().Err(err).Msg("Token validation failed after key refresh")
					http.Error(w, "Invalid token", http.StatusUnauthorized)
					return
				}
			}

			log.Info().Msgf("Token: %v", token)

			// Verify token use claim
			tokenUse, ok := token.Get("token_use")
			if !ok || tokenUse != "access" {
				http.Error(w, "Invalid token use", http.StatusUnauthorized)
				return
			}

			// Store the parsed token in the context
			ctx := context.WithValue(r.Context(), jwtContextKey, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TokenFromContext retrieves the JWT token from the context
func TokenFromContext(ctx context.Context) (jwt.Token, bool) {
	token, ok := ctx.Value(jwtContextKey).(jwt.Token)
	return token, ok
}

// GetClaim is a helper to get a claim value from the token
func GetClaim(ctx context.Context, claim string) (interface{}, bool) {
	token, ok := TokenFromContext(ctx)
	if !ok {
		return nil, false
	}
	return token.Get(claim)
}
