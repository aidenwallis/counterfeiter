package sql

import (
	"database/sql"
)

//go:generate go run github.com/aidenwallis/counterfeiter/v7 . DB

type DB interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}
