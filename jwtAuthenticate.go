package cwbiechotoolkit

import (
	"crypto/rsa"
	"fmt"
	"log"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
	echojwt "github.com/labstack/echo-jwt"
	"github.com/labstack/echo/v4"
)

type (
	// JwtAuthenticateConfig struct defining needed fields to validate and authorize
	JwtAuthenticateConfig struct {
		// Skipper defines a function to skip middleware
		// Returning true skips processing the middleware.
		Skipper func(c echo.Context) bool

		// ApplicationKey is the key to validate the application
		ApplicationKey string

		// PublicKeyRaw is the public key to validate the token
		PublicKeyRaw string

		// PublicKey is the public key to validate the token as rsa.PublicKey
		// This is typically set by the middleware
		PublicKey *rsa.PublicKey

		// TokenRoles is the role(s) from the token
		TokenRoles []any

		// SigningMethod is the signing method for the token algorithm
		// Determined from the token
		SigningMethod string
	}

	// AuthorizeStandardClaims struct for ParseWithClaims
	AuthorizeCustomClaims struct {
		AuthrorizedParty  string         `json:"azp,omitempty"`
		ResourceAccess    map[string]any `json:"resource_access,omitempty"`
		PreferredUsername string         `json:"preferred_username,omitempty"`
		Name              string         `json:"name,omitempty"`
		GivenName         string         `json:"given_name,omitempty"`
		FamilyName        string         `json:"family_name,omitempty"`
		Subject           string         `json:"sub,omitempty"`
		Audience          []string       `json:"aud,omitempty"`
		// jwt.StandardClaims  // this is for 'aud' that is a string
		jwt.MapClaims  // start using this with custom claims
	}
)

var (
	DefaultJwtAuthenticateConfig = JwtAuthenticateConfig{
		Skipper: DefaultJwtAuthSkipper,
	}
)

// DefaultSkipper returns false which processes the middleware.
func DefaultJwtAuthSkipper(echo.Context) bool {
	return false
}

// AuthParseWithClaims
func (a *JwtAuthenticateConfig) AuthParseWithClaims() echo.MiddlewareFunc {
	return echojwt.WithConfig(echojwt.Config{
		ParseTokenFunc: func(c echo.Context, auth string) (any, error) {
			token, err := jwt.ParseWithClaims(auth, &AuthorizeCustomClaims{}, func(token *jwt.Token) (any, error) {
				a.SigningMethod = token.Header["alg"].(string)
				switch token.Method.(type) {
				case *jwt.SigningMethodRSA:
					k, err := jwt.ParseRSAPublicKeyFromPEM([]byte("-----BEGIN PUBLIC KEY-----\n" + a.PublicKeyRaw + "\n-----END PUBLIC KEY-----"))
					if err != nil {
						log.Printf("Error: %s\n", err)
						return nil, err
					}
					a.PublicKey = k
					return a.PublicKey, nil
				case *jwt.SigningMethodHMAC:
					return []byte(a.PublicKeyRaw), nil
				default:
					return nil, fmt.Errorf("unexpected signing method: %s", a.SigningMethod)
				}
			})
			if err != nil {
				return nil, err
			}

			// is token valid and checking claims
			if !token.Valid {
				return nil, fmt.Errorf("token not valid")
			}

			return token, nil
		},
		ErrorHandler: func(c echo.Context, err error) error {
			return c.JSON(http.StatusUnauthorized, map[string]string{"message": err.Error()})
		},
		SigningMethod: a.SigningMethod,
		TokenLookup:   "header:Authorization:Bearer ",
	})
}
