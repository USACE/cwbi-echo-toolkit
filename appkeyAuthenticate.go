package cwbiechotoolkit

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// DefaultAuthAppkeyConfig implements Echo middleware.KeyAuthConfig
// configuration with default values
func DefaultAuthAppkeyConfig(appkey string) middleware.KeyAuthConfig {
	return middleware.KeyAuthConfig{
		Skipper:      DefaultAppkeySkipper,
		KeyLookup:    "header:Authorization",
		AuthScheme:   "Appkey",
		Validator:    DefaultAppkeyValidator(appkey),
		ErrorHandler: DefaultErrorHandler(),
	}
}

// DefaultAppkeySkipper function returns a boolean for the Appkey Skipper
// and the value is false.
func DefaultAppkeySkipper(c echo.Context) bool {
	return false
}

// DefaultAppkeyValidator implements Echo middleware.KeyAuthValidator returning
// boolean and error
//
// Parameters:
// appkey is the application key like "bearer abcdefghijklmnop123456789"
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

// DefaultErrorHandler implements Echo middleware KeyAuthErrorHandler
func DefaultErrorHandler() middleware.KeyAuthErrorHandler {
	return func(err error, c echo.Context) error {
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": err.Error()})
	}
}
