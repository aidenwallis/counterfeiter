package sql

import (
	"database/sql"
)

//go:generate go run github.com/aidenwallis/counterfeiter/v6 . DB

type DB interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}
