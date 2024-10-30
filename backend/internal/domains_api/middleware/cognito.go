package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/shannevie/unofficial_cybertrap/backend/internal/domains_api/jwk"
)

type CognitoMiddleware struct {
	jwk *jwk.CognitoJWK
}

func NewCognitoMiddleware(cognitoJWK *jwk.CognitoJWK) *CognitoMiddleware {
	return &CognitoMiddleware{
		jwk: cognitoJWK,
	}
}

func (cm *CognitoMiddleware) Verify() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

			// Parse and validate the token
			token, err := jwt.Parse(
				[]byte(tokenString),
				jwt.WithKeySet(cm.jwk.GetKeySet()),
				jwt.WithValidate(true),
				jwt.WithIssuer(cm.jwk.GetIssuer()),
				jwt.WithAudience(cm.jwk.ClientID),
			)
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Verify token use claim
			if tokenUse, ok := token.Get("token_use"); !ok || tokenUse != "access" {
				http.Error(w, "Invalid token use", http.StatusUnauthorized)
				return
			}

			// Store the parsed token in the context
			ctx := context.WithValue(r.Context(), "jwt", token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TokenFromContext retrieves the JWT token from the context
func TokenFromContext(ctx context.Context) (jwt.Token, bool) {
	token, ok := ctx.Value("jwt").(jwt.Token)
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
