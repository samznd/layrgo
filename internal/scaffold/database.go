package scaffold

import "strings"

func databaseFile(database, orm string) string {
	if strings.ToLower(orm) != "none" {
		return ormDatabaseFile(orm, database)
	}
	return rawDatabaseFile(database)
}

func rawDatabaseFile(database string) string {
	switch strings.ToLower(database) {
	case "postgres":
		return `package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Connect() {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err = DB.Ping(); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("connected to database")
}`
	case "mysql":
		return `package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func Connect() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))

	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err = DB.Ping(); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("connected to database")
}`
	case "sqlite":
		return `package config

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Connect() {
	var err error
	DB, err = sql.Open("sqlite3", os.Getenv("DB_NAME"))
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err = DB.Ping(); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("connected to database")
}`
	default:
		return ""
	}
}

func ormDatabaseFile(orm, database string) string {
	switch strings.ToLower(orm) {
	case "gorm":
		return gormDatabaseFile(database)
	case "xorm":
		return xormDatabaseFile(database)
	case "ent":
		// Ent schema is project-specific; scaffold uses sql.DB until ent init is run.
		return rawDatabaseFile(database)
	default:
		return ""
	}
}

func gormDatabaseFile(database string) string {
	switch strings.ToLower(database) {
	case "postgres":
		return `package config

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	log.Println("connected to database")
}`
	case "mysql":
		return `package config

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	log.Println("connected to database")
}`
	case "sqlite":
		return `package config

import (
	"log"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "mydb.db"
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	log.Println("connected to database")
}`
	default:
		return ""
	}
}

func xormDatabaseFile(database string) string {
	switch strings.ToLower(database) {
	case "postgres":
		return `package config

import (
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"xorm.io/xorm"
)

var DB *xorm.Engine

func Connect() {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	var err error
	DB, err = xorm.NewEngine("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err = DB.Ping(); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("connected to database")
}`
	case "mysql":
		return `package config

import (
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"xorm.io/xorm"
)

var DB *xorm.Engine

func Connect() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4",
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))

	var err error
	DB, err = xorm.NewEngine("mysql", dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err = DB.Ping(); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("connected to database")
}`
	case "sqlite":
		return `package config

import (
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

var DB *xorm.Engine

func Connect() {
	var err error
	DB, err = xorm.NewEngine("sqlite3", os.Getenv("DB_NAME"))
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err = DB.Ping(); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("connected to database")
}`
	default:
		return ""
	}
}
