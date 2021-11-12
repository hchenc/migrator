package client

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

func NewMigratorClient(databaseUrl *url.URL, user, pass, location, table string, log io.Writer, dump bool) *Migrator {
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
	return migrator.migrate(0)
}

func (migrator *Migrator) Status(quiet bool) (int, error) {
	results, err := migrator.CheckMigrationsStatus()
	if err != nil {
		return -1, err
	}
	var totalApplied int
	var line string

	for _, res := range results {
		if res.Applied {
			line = fmt.Sprintf("[V] %s", res.Filename)
			totalApplied++
		} else {
			line = fmt.Sprintf("[X] %s", res.Filename)
		}
		if !quiet {
			fmt.Fprintln(migrator.Log, line)
		}
	}

	totalPending := len(results) - totalApplied
	if !quiet {
		fmt.Fprintln(migrator.Log)
		fmt.Fprintf(migrator.Log, "Applied: %d\n", totalApplied)
		fmt.Fprintf(migrator.Log, "Pending: %d\n", totalPending)
	}

	return totalPending, nil
}

func (migrator *Migrator) Up(step uint) error {
	return migrator.migrate(int(step))
}

func (migrator *Migrator) CheckMigrationsStatus() ([]StatusResult, error) {
	files := utils.MustFindMigrationFiles(migrator.MigrationsLocation, utils.MigrationFileRegexp)

	sqlDB, err := migrator.openDatabaseForMigration()
	defer sqlDB.Close()
	if err != nil {
		return nil, err
	}

	applied, err := migrator.backend.SelectMigrations(sqlDB, -1)
	if err != nil {
		return nil, err
	}

	var results []StatusResult

	for _, filename := range files {
		ver := utils.MigrationVersion(filename)
		res := StatusResult{Filename: filename}
		if ok := applied[ver]; ok {
			res.Applied = true
		} else {
			res.Applied = false
		}

		results = append(results, res)
	}

	return results, nil
}

func (migrator *Migrator) migrate(step int) error {
	files := utils.MustFindMigrationFiles(migrator.MigrationsLocation, utils.MigrationFileRegexp)

	sqlDB, err := migrator.openDatabaseForMigration()
	defer sqlDB.Close()
	if err != nil {
		return err
	}

	applied, err := migrator.backend.SelectMigrations(sqlDB, -1)
	if err != nil {
		return err
	}

	if step == 0 {
	} else if slice := len(applied) + step; slice < len(files) {
		files = files[0:slice]
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
			return migrator.backend.InsertMigration(tx, ver)
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
		_ = migrator.dumpSchema()
	}

	return nil
}

func (migrator *Migrator) Rollback() error {
	return migrator.down(1)
}

func (migrator *Migrator) Down(step uint) error {
	return migrator.down(int(step))
}

func (migrator *Migrator) down(step int) error {
	sqlDB, err := migrator.openDatabaseForMigration()
	defer sqlDB.Close()
	if err != nil {
		return err
	}
	for s := 0; s < step; s++ {
		applied, err := migrator.backend.SelectMigrations(sqlDB, 1)
		// grab most recent applied migration (applied has len=1)
		latest := ""
		for ver := range applied {
			latest = ver
		}
		if latest == "" {
			return fmt.Errorf("can't rollback: no migrations have been applied")
		}

		filename := utils.MustFindMigrationFile(migrator.MigrationsLocation, latest)

		fmt.Fprintf(migrator.Log, "Rolling back: %s\n", filename)

		_, down, err := parseMigration(filepath.Join(migrator.MigrationsLocation, filename))
		if err != nil {
			return err
		}

		execMigration := func(tx backends.Transaction) error {
			// rollback migration
			result, err := tx.Exec(down.Contents)
			if err != nil {
				return err
			} else if migrator.Verbose {
				migrator.printVerbose(result)
			}

			// remove migration record
			return migrator.backend.DeleteMigration(tx, latest)
		}

		if down.Options.Transaction() {
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
		_ = migrator.dumpSchema()
	}

	return nil
}

func (migrator *Migrator) openDatabaseForMigration() (*sql.DB, error) {
	sqlDB, err := migrator.backend.OpenDatabase()
	if err != nil {
		return nil, err
	}

	if err := migrator.backend.CreateMigrationsTable(sqlDB); err != nil {
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

func (migrator *Migrator) dumpSchema() error {

	sqlDB, err := migrator.openDatabaseForMigration()
	defer sqlDB.Close()
	if err != nil {
		return err
	}

	schema, err := migrator.backend.DumpSchema(sqlDB)
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
