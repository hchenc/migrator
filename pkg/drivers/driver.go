package drivers

import (
	"database/sql"
	"github.com/hchenc/migrator/pkg/backends"
)

type DriverType string

type DriverService interface {
	Open() (*sql.DB, error)
	SchemaExists() (bool, error)
	CreateSchema() error
	DropSchema() error
	DumpSchema(db *sql.DB) ([]byte, error)
	CreateMigrationsTable(db *sql.DB) error
	SelectMigrations(db *sql.DB, id int) (map[string]bool, error)
	InsertMigration(tx backends.Transaction, version string) error
	DeleteMigration(tx backends.Transaction, version string) error
	Ping() error
}

type backend struct {
	ds DriverService
}

func (b *backend) OpenDatabase() (*sql.DB, error) {
	return b.ds.Open()
}

func (b *backend) DatabaseExists() (bool, error) {
	return b.ds.SchemaExists()
}

func (b *backend) CreateDatabase() error {
	panic("implement me")
}

func (b *backend) DropDatabase() error {
	return b.ds.CreateSchema()
}

func (b *backend) DumpSchema(db *sql.DB) ([]byte, error) {
	return b.ds.DumpSchema(db)
}

func (b *backend) CreateMigrationsTable(db *sql.DB) error {
	return b.ds.CreateMigrationsTable(db)
}

func (b *backend) SelectMigrations(db *sql.DB, id int) (map[string]bool, error) {
	return b.ds.SelectMigrations(db, id)
}

func (b *backend) InsertMigration(transation backends.Transaction, migration string) error {
	return b.ds.InsertMigration(transation, migration)
}

func (b *backend) DeleteMigration(transation backends.Transaction, migration string) error {
	return b.ds.DeleteMigration(transation, migration)
}

func (b *backend) Ping() error {
	return b.ds.Ping()
}

func NewBackendService(ds DriverService) backends.Interface {
	return &backend{ds: ds}
}

type DriverGenerator func(config *backends.BackendConfig) DriverService

var DriverMap = map[DriverType]DriverGenerator{}

func RegisterDriver(generator DriverGenerator, driverType DriverType) {
	DriverMap[driverType] = generator
}
