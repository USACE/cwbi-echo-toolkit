package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func DefaultAuthAppkeyConfig(appkey string) middleware.KeyAuthConfig {
	return middleware.KeyAuthConfig{
		Skipper:      DefaultAppkeySkipper,
		KeyLookup:    "header:Authorization",
		AuthScheme:   "Appkey",
		Validator:    DefaultAppkeyValidator(appkey),
		ErrorHandler: DefaultErrorHandler(),
	}
}

// DefaultSkipper returns false which processes the middleware.
func DefaultAppkeySkipper(c echo.Context) bool {
	return false
}

// DefaultAppkeyValidator validates the application key
func DefaultAppkeyValidator(appkey string) middleware.KeyAuthValidator {
	return func(auth string, c echo.Context) (bool, error) {
		if auth == "" {
			return false, errors.New("missing API authentication")
		}
		// split the header and get the auth
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 {
			return false, errors.New("need format '<AuthScheme> <Token>'")
		}

		if parts[1] == appkey {
			return true, nil
		}

		return false, errors.New("invalid API key")
	}
}

// DefaultErrorHandler
func DefaultErrorHandler() middleware.KeyAuthErrorHandler {
	return func(err error, c echo.Context) error {
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": err.Error()})
	}
}
