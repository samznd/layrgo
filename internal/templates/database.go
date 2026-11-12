package templates

import (
	"fmt"
	"strings"
)

// dbSpec is the per-database bits the config templates need.
type dbSpec struct {
	driverImport  string // raw database/sql + xorm driver import line
	driverName    string // sql.Open / xorm.NewEngine driver string
	dsnBlock      string // statement(s) producing a `dsn` variable
	gormImport    string
	gormDialector string
}

var dbSpecs = map[string]dbSpec{
	"postgres": {
		driverImport: `_ "github.com/lib/pq"`,
		driverName:   `"postgres"`,
		dsnBlock: `dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))`,
		gormImport:    `"gorm.io/driver/postgres"`,
		gormDialector: `postgres.Open(dsn)`,
	},
	"mysql": {
		driverImport: `_ "github.com/go-sql-driver/mysql"`,
		driverName:   `"mysql"`,
		dsnBlock: `dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))`,
		gormImport:    `"gorm.io/driver/mysql"`,
		gormDialector: `mysql.Open(dsn)`,
	},
	"sqlite": {
		driverImport: `_ "github.com/mattn/go-sqlite3"`,
		driverName:   `"sqlite3"`,
		dsnBlock: `dsn := os.Getenv("DB_NAME")
	if dsn == "" {
		dsn = "mydb.db"
	}`,
		gormImport:    `"gorm.io/driver/sqlite"`,
		gormDialector: `sqlite.Open(dsn)`,
	},
}

// DatabaseFile renders config/database.go for the given database and ORM.
func DatabaseFile(database, orm string) (string, error) {
	database = strings.ToLower(database)
	orm = strings.ToLower(orm)

	spec, ok := dbSpecs[database]
	if !ok {
		return "", fmt.Errorf("unsupported database %q", database)
	}
	if !orms[orm] {
		return "", fmt.Errorf("unsupported orm %q", orm)
	}

	data := Data{
		Database:      database,
		ORM:           orm,
		DriverImport:  spec.driverImport,
		DriverName:    spec.driverName,
		DSNBlock:      spec.dsnBlock,
		GormDialector: spec.gormDialector,
	}

	switch orm {
	case "gorm":
		// GORM has its own driver.
		data.DriverImport = spec.gormImport
		return render("config/gorm.go.tmpl", data)
	case "xorm":
		return render("config/xorm.go.tmpl", data)
	default:
		// "none" and "ent" both use database/sql.
		return render("config/sql.go.tmpl", data)
	}
}
