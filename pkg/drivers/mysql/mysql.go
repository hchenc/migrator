package mysql

import (
	"database/sql"
	"fmt"
	"github.com/hchenc/migrator/pkg/backends"
	"github.com/hchenc/migrator/pkg/drivers"
	"github.com/hchenc/migrator/pkg/utils"

	_ "github.com/go-sql-driver/mysql"
)

func init() {
	drivers.RegisterDriver(newMysqlDriver, "mysql")
}

func newMysqlDriver(config *backends.BackendConfig) drivers.DriverService {
	config.MigrationsTable = utils.FormateDatabaseStr(config.MigrationsTable)
	return &mysqlDriver{
		config:     config,
		driverType: "mysql",
	}
}

type mysqlDriver struct {
	config     *backends.BackendConfig
	driverType drivers.DriverType
}

func (m *mysqlDriver) Open() (*sql.DB, error) {
	connStr := m.getConn("")

	return sql.Open(string(m.driverType), connStr)
}

func (m *mysqlDriver) getConn(schema string) string {
	if len(schema) == 0 {
		schema = m.config.DatabaseUrl.Path
	}
	query := m.config.DatabaseUrl.Query()
	query.Set("multiStatements", "true")

	host := m.config.DatabaseUrl.Host
	protocol := "tcp"

	if query.Get("socket") != "" {
		protocol = "unix"
		host = query.Get("socket")
		query.Del("socket")
	} else if m.config.DatabaseUrl.Port() == "" {
		// set default port
		host = fmt.Sprintf("%s:3306", host)
	}

	// Get decoded user:pass
	user := m.config.DatabaseUser
	pass := m.config.DatabasePass

	// Build DSN w/ user:pass percent-decoded
	connStr := ""

	if pass != "" { // user:pass can be empty
		connStr = user + ":" + pass + "@"
	}

	// connection string format required by go-sql-driver/mysql
	connStr = fmt.Sprintf("%s%s(%s)%s?%s", connStr,
		protocol, host, schema, query.Encode())
	return connStr
}

func (m *mysqlDriver) SchemaExists() (bool, error) {
	schema := utils.GetSchameName(m.config.DatabaseUrl)
	db, err := sql.Open(string(m.driverType), m.getConn("/"))
	defer db.Close()
	if err != nil {
		return false, err
	}

	exists := false
	err = db.QueryRow("select true from information_schema.schemata "+
		"where schema_name = ?", schema).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}

	return exists, err


}

func (m *mysqlDriver) CreateSchema() error {
	panic("implement me")
}

func (m *mysqlDriver) DropSchema() error {
	panic("implement me")
}

func (m *mysqlDriver) DumpSchema(db *sql.DB) ([]byte, error) {
	panic("implement me")
}

func (m *mysqlDriver) CreateMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(
		fmt.Sprintf("create table if not exists %s "+
			"(version varchar(255) primary key) character set latin1 collate latin1_bin", m.config.MigrationsTable))
	return err
}

func (m *mysqlDriver) SelectMigrations(db *sql.DB, id int) (map[string]bool, error) {
	query := fmt.Sprintf("select version from %s order by version desc", m.config.MigrationsTable)

	if id >= 0 {
		query = fmt.Sprintf("%s limit %d", query, id)
	}
	rows, err := db.Query(query)
	defer rows.Close()

	if err != nil {
		return nil, err
	}

	migrations := map[string]bool{}
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		migrations[version] = true
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return migrations, nil
}

func (m *mysqlDriver) InsertMigration(tx backends.Transaction, version string) error {
	_, err := tx.Exec(
		fmt.Sprintf("insert into %s (version) values (?)", m.config.MigrationsTable),
		version)

	return err
}

func (m *mysqlDriver) DeleteMigration(tx backends.Transaction, version string) error {
	_, err := tx.Exec(
		fmt.Sprintf("delete from %s where version = ?", m.config.MigrationsTable),
		version)

	return err
}

func (m *mysqlDriver) Ping() error {
	db, err := sql.Open(string(m.driverType), m.getConn("/"))
	defer db.Close()
	if err != nil {
		return err
	}

	return db.Ping()
}


