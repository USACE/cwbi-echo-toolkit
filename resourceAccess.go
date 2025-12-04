package cwbiechotoolkit

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

type (
	// ResourceAccessConfig struct defining needed fields to validate and authorize
	ResourceAccessConfig struct {
		// Skipper defines a function to skip middleware.
		// Returning true skips processing the middleware.
		Skipper func(c echo.Context) bool

		// Roles is the list of roles to authorize
		Roles []string

		// Role separator.
		// Optional.  Default value ""
		RoleSeparator string

		// Typically the office of the user defined in routes
		Scope string

		// Scope Variable
		// Optional.  Default value "symbol"
		ScopeVariable string

		// ScopeFromContext func to get the scope
		// Typically from the route and is most likely an office symbol
		ScopeFromContext func(c echo.Context, scopeVariable string) string

		// Context key.  Use this if jwtAuth sets something different than 'user'
		// Optional.  Default value "user"
		ContextKey string

		// Claims.
		Claims AuthorizeCustomClaims
	}
)

var (
	DefaultResourceAccessConfig = ResourceAccessConfig{
		Skipper: DefaultResourceAcessSkipper,
	}
)

// DefaultResourceAcessSkipper returns false which processes the middleware.
func DefaultResourceAcessSkipper(echo.Context) bool {
	return false
}

// DefaultScopeFromContext
func DefaultScopeFromContext(c echo.Context, scopeVariable string) string {
	paramNames := c.ParamNames()
	for _, name := range paramNames {
		if name == scopeVariable {
			return c.Param(name)
		}
	}
	return ""
}

// AuthResourceAccessWithConfig
func ResourceAccessWithConfig(accessConfig ResourceAccessConfig) echo.MiddlewareFunc {
	if accessConfig.Skipper == nil {
		accessConfig.Skipper = DefaultResourceAccessConfig.Skipper
	}

	if accessConfig.ScopeVariable == "" {
		accessConfig.ScopeVariable = "symbol"
	}

	if accessConfig.ScopeFromContext == nil {
		accessConfig.ScopeFromContext = DefaultScopeFromContext
	}

	if accessConfig.ContextKey == "" {
		accessConfig.ContextKey = "user"
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			// skip if true
			if accessConfig.Skipper(c) {
				return next(c)
			}

			// define Scope from context func
			accessConfig.Scope = accessConfig.ScopeFromContext(c, accessConfig.ScopeVariable)
			log.Printf("\nThe scope (the office code) from context (from the route) is '%s'\n", accessConfig.Scope)

			// getting the valid token from the context key 'user'
			user, ok := c.Get(accessConfig.ContextKey).(*jwt.Token)
			if !ok {
				return fmt.Errorf("error getting %s from context", accessConfig.ContextKey)
			}

			// get the claims from the token
			claims, ok := user.Claims.(*AuthorizeCustomClaims)
			if !ok {
				return fmt.Errorf("error cast claims")
			}

			// get the resource access and the application roles making sure the app name is the same as the AuthorizedParty
			resource_access := claims.ResourceAccess
			app_roles, ok := resource_access[claims.AuthrorizedParty].(map[string]any)
			if !ok {
				return fmt.Errorf("failed to cast resource_access[app] as map[string]interface{}")
			}

			// getting the roles
			tokenRoles, ok := app_roles["roles"].([]any)
			if !ok {
				return fmt.Errorf("failed to cast app roles")
			}
			log.Printf("\nRoles from user's token: %v\n", tokenRoles...)

			// loop through the roles from the token and compare to the source of truth defined by the API's middleware configurations
			for _, tokenRole := range tokenRoles {
				tokenRoleString := tokenRole.(string)                                   // token roles are interfaces that need to be cast to string
				scopeRole := strings.Split(tokenRoleString, accessConfig.RoleSeparator) // split the incoming role by the defined role separeator
				for _, authRole := range accessConfig.Roles {
					authRole := strings.TrimSpace(authRole)
					switch len(scopeRole) {
					case 1: // with no split meaning no rolw sepe
						if strings.EqualFold(tokenRoleString, authRole) {
							log.Printf("\nThe user's token role '%s' equals the authentication role '%s'\n", tokenRoleString, authRole)
							return next(c)
						}
					case 2:
						scope := scopeRole[0]
						role := scopeRole[1]
						if (strings.EqualFold(accessConfig.Scope, scope) || scope == "application") && strings.EqualFold(authRole, role) {
							log.Printf("\nThe user's scope and role '%s, %s' equals the authentication scope and role '%s, %s'\n", scope, role, accessConfig.Scope, authRole)
							return next(c)
						}
					default:
						msg := fmt.Sprintf("\nThe user's scope and role '%s' not matching the authentication scope and role '%s, %s'\n", tokenRoleString, accessConfig.Scope, authRole)
						log.Print(msg)
						return c.JSON(http.StatusUnauthorized, map[string]string{"message": msg})
					}
				}
			}
			return c.JSON(http.StatusUnauthorized, map[string]string{"message": "resource access not allowed with current roles"})
		}
	}
}
