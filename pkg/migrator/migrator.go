package migrator

import (
	"database/sql"
	"fmt"
	"github.com/hchenc/migrator/pkg/backends"
	"github.com/hchenc/migrator/pkg/constants"
	"github.com/hchenc/migrator/pkg/drivers"
	_ "github.com/hchenc/migrator/pkg/drivers"
	"github.com/hchenc/migrator/pkg/utils"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

type Migrator struct {
	backend            backends.Interface
	AutoDumpSchema     bool
	SchemaFile         string
	DatabaseUrl        *url.URL
	MigrationsLocation string
	MigrationsTable    string
	Verbose            bool
	Log                io.Writer
}

type StatusResult struct {
	Filename string
	Applied  bool
}

func New(databaseURL *url.URL) *Migrator {
	return &Migrator{
		AutoDumpSchema:     true,
		DatabaseUrl:        databaseURL,
		MigrationsLocation: constants.DefaultMigrationsLocation,
		MigrationsTable:    constants.DefaultMigrationsTable,
		Log:                os.Stdout,
	}
}

func (migrator *Migrator) New(name string) error {
	// new migration name
	timestamp := time.Now().UTC().Format("20060102150405")
	if name == "" {
		return fmt.Errorf("please specify a name for the new migration")
	}
	name = fmt.Sprintf("%s_%s.sql", timestamp, name)

	// create migrations dir if missing
	if err := utils.EnsureDir(migrator.MigrationsLocation); err != nil {
		return err
	}

	// check file does not already exist
	path := filepath.Join(migrator.MigrationsLocation, name)
	fmt.Fprintf(migrator.Log, "Creating migration: %s\n", path)

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return fmt.Errorf("file already exists")
	}

	// write new migration
	file, err := os.Create(path)
	defer file.Close()
	if err != nil {
		return err
	}

	_, err = file.WriteString(constants.MigrationTemplate)
	return err
}

func NewMigrator(databaseUrl *url.URL, user, pass, location, table string, log io.Writer, dump bool) *Migrator {
	migrator := &Migrator{
		AutoDumpSchema:     dump,
		DatabaseUrl:        databaseUrl,
		MigrationsLocation: location,
		MigrationsTable:    table,
		Log:                os.Stdout,
	}
	if databaseUrl == nil || databaseUrl.Scheme == "" {
		panic("invalid url")
	}
	generator := drivers.DriverMap[drivers.DriverType(databaseUrl.Scheme)]
	driver := generator(&backends.BackendConfig{
		DatabaseUrl:     databaseUrl,
		DatabaseUser:    user,
		DatabasePass:    pass,
		MigrationsTable: table,
		Log:             log,
	})
	migrator.backend = drivers.NewBackendService(driver)
	return migrator
}

func (migrator *Migrator) Migrate() error {

	return migrator.migrate(migrator.backend)
}

func (migrator *Migrator) migrate(backend backends.Interface) error {
	files, err := utils.FindMigrationFiles(migrator.MigrationsLocation, utils.MigrationFileRegexp)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no migration files found")
	}

	sqlDB, err := migrator.openDatabaseForMigration(backend)
	defer sqlDB.Close()
	if err != nil {
		return err
	}

	applied, err := backend.SelectMigrations(sqlDB, -1)
	if err != nil {
		return err
	}

	for _, filename := range files {
		ver := utils.MigrationVersion(filename)
		if ok := applied[ver]; ok {
			// migration already applied
			continue
		}

		fmt.Fprintf(migrator.Log, "Try to migrate: %s\n", filename)

		up, _, err := parseMigration(filepath.Join(migrator.MigrationsLocation, filename))
		if err != nil {
			return err
		}

		execMigration := func(tx backends.Transaction) error {
			// run actual migration
			result, err := tx.Exec(up.Contents)
			if err != nil {
				return err
			} else if migrator.Verbose {
				migrator.printVerbose(result)
			}

			// record migration
			return backend.InsertMigration(tx, ver)
		}

		if up.Options.Transaction() {
			// begin transaction
			err = doTransaction(sqlDB, execMigration)
		} else {
			// run outside of transaction
			err = execMigration(sqlDB)
		}

		if err != nil {
			return err
		}
	}

	// automatically update schema file, silence errors
	if migrator.AutoDumpSchema {
		_ = migrator.dumpSchema(backend)
	}

	return nil
}

func (migrator *Migrator) openDatabaseForMigration(backend backends.Interface) (*sql.DB, error) {
	sqlDB, err := backend.OpenDatabase()
	if err != nil {
		return nil, err
	}

	if err := backend.CreateMigrationsTable(sqlDB); err != nil {
		defer sqlDB.Close()
		return nil, err
	}

	return sqlDB, nil
}

func parseMigration(path string) (Migration, Migration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return NewMigration(), NewMigration(), err
	}
	up, down, err := parseMigrationContents(string(data))
	return up, down, err
}

func (migrator *Migrator) printVerbose(result sql.Result) {
	lastInsertID, err := result.LastInsertId()
	if err == nil {
		fmt.Fprintf(migrator.Log, "Last insert ID: %d\n", lastInsertID)
	}
	rowsAffected, err := result.RowsAffected()
	if err == nil {
		fmt.Fprintf(migrator.Log, "Rows affected: %d\n", rowsAffected)
	}
}

func doTransaction(sqlDB *sql.DB, txFunc func(backends.Transaction) error) error {
	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}

	if err := txFunc(tx); err != nil {
		if err1 := tx.Rollback(); err1 != nil {
			return err1
		}

		return err
	}

	return tx.Commit()
}

func (migrator *Migrator) dumpSchema(backend backends.Interface) error {

	sqlDB, err := migrator.openDatabaseForMigration(backend)
	defer sqlDB.Close()
	if err != nil {
		return err
	}

	schema, err := backend.DumpSchema(sqlDB)
	if err != nil {
		return err
	}

	fmt.Fprintf(migrator.Log, "Writing: %s\n", migrator.SchemaFile)

	// ensure schema directory exists
	if err = utils.EnsureDir(filepath.Dir(migrator.SchemaFile)); err != nil {
		return err
	}

	// write schema to file
	return os.WriteFile(migrator.SchemaFile, schema, 0644)
}
