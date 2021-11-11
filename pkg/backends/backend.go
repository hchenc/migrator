package backends

import (
	"database/sql"
	"io"
	"net/url"
)

type Interface interface {
	OpenDatabase() (*sql.DB, error)
	DatabaseExists() (bool, error)
	CreateDatabase() error
	DropDatabase() error
	DumpSchema(db *sql.DB) ([]byte, error)
	CreateMigrationsTable(db *sql.DB) error
	SelectMigrations(db *sql.DB, id int) (map[string]bool, error)
	InsertMigration(tx Transaction, version string) error
	DeleteMigration(tx Transaction, version string) error
	Ping() error
}

// Transaction to abstract tx from sql and sqlx
type Transaction interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

type BackendConfig struct {
	DatabaseUrl     *url.URL
	DatabaseUser    string
	DatabasePass    string
	MigrationsTable string
	Log             io.Writer
}
