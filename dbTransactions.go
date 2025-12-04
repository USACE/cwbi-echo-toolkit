package cwbiechotoolkit

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/labstack/echo/v4"
)

type DbConfigurtion struct {
	ID    uuid.UUID `db:"id" json:"id"`
	Key   string    `db:"key" json:"key"`
	Value string    `db:"value" json:"value"`
	Type  string    `db:"type" json:"type"`
}

type (
	// ResourceAccessConfig struct defining needed fields to validate and authorize
	DatabaseTransaction struct {
		// Skipper defines a function to skip middleware.
		// Returning true skips processing the middleware.
		Skipper func(c echo.Context) bool

		// sql query
		SQL *string

		// key used in the default query
		// required when SQL not provided
		Key *string

		// Application Configuration
		Config *any

		// Application Configuration attribute
		ConfigFieldName *string

		// database connection pool
		Connection *pgxpool.Pool
	}
)

var (
	DefaultDatabaseTransaction = ResourceAccessConfig{
		Skipper: DefaultDatabaseSkipper,
	}
)

// DefaultDatabaseSkipper returns false which processes the middleware.
func DefaultDatabaseSkipper(echo.Context) bool {
	return false
}

func DatabaseTransactionWithConfig(dbCfg DatabaseTransaction) echo.MiddlewareFunc {
	if dbCfg.Skipper == nil {
		dbCfg.Skipper = DefaultDatabaseTransaction.Skipper
	}
	if dbCfg.SQL == nil {
		sql := fmt.Sprintf(`SELECT "id" , "key" , value , "type" FROM configuration WHERE key = '%s'`, *dbCfg.Key)
		dbCfg.SQL = &sql
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			// skip if true
			if dbCfg.Skipper(c) {
				return next(c)
			}

			// get the state of the database for the office user admin
			var dbConfiguration DbConfigurtion
			if err = pgxscan.Get(context.TODO(), dbCfg.Connection, &dbConfiguration, *dbCfg.SQL); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
			}

			v, err := strconv.ParseBool(dbConfiguration.Value)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
			}
			if v {
				msg := fmt.Sprintf("Lock transactions configuration set to '%s'.  No transactions allowed at this time.", dbConfiguration.Value)
				return c.JSON(http.StatusUnauthorized, map[string]string{"message": msg})
			}

			return next(c)
		}
	}
}

func DatabaseSetApplicationWithConfig(dbCfg DatabaseTransaction) echo.MiddlewareFunc {
	if dbCfg.Skipper == nil {
		dbCfg.Skipper = DefaultDatabaseTransaction.Skipper
	}
	if dbCfg.SQL == nil {
		sql := fmt.Sprintf(`SELECT "id" , "key" , value , "type" FROM configuration WHERE key = '%s'`, *dbCfg.Key)
		dbCfg.SQL = &sql
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			// skip if true
			if dbCfg.Skipper(c) {
				return next(c)
			}

			// get the state of the database for the office user admin
			var dbConfiguration DbConfigurtion
			if err = pgxscan.Get(context.TODO(), dbCfg.Connection, &dbConfiguration, *dbCfg.SQL); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
			}

			var v any
			switch dbConfiguration.Type {
			case "string":
				v = dbConfiguration.Value
			case "boolean":
				v, err = strconv.ParseBool(dbConfiguration.Value)
				if err != nil {
					return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
				}
			case "integer":
				v, err = strconv.Atoi(dbConfiguration.Value)
				if err != nil {
					return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
				}
			default:
				msg := fmt.Sprintf("Key: %s, Value: %v (Type: unknown)", *dbCfg.Key, v)
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": msg})
			}

			if dbCfg.ConfigFieldName != nil {
				assignFieldValue(dbCfg.Config, *dbCfg.ConfigFieldName, v)
			}
			return next(c)
		}
	}
}

// assignFieldValue sets a value to the struct field
func assignFieldValue(p any, fieldName string, value any) error {
	v := reflect.ValueOf(p).Elem()
	field := v.FieldByName(fieldName)

	if !field.IsValid() {
		return fmt.Errorf("field %s not found", fieldName)
	}

	if !field.CanSet() {
		return fmt.Errorf("cannot set field %s", fieldName)
	}

	val := reflect.ValueOf(value)

	if field.Type() != val.Type() {
		return fmt.Errorf("type mismatch: expected %s, got %s", field.Type(), val.Type())
	}

	field.Set(val)

	return nil
}
