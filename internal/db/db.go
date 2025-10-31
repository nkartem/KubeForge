package db

import (
	"fmt"
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database instance
var DB *gorm.DB

// Config holds database configuration
type Config struct {
	Driver string
	DSN    string
}

// Init initializes the database connection
func Init(config Config) error {
	var dialector gorm.Dialector

	switch config.Driver {
	case "sqlite":
		dialector = sqlite.Open(config.DSN)
	case "postgres":
		dialector = postgres.Open(config.DSN)
	case "mysql":
		dialector = mysql.Open(config.DSN)
	default:
		return fmt.Errorf("unsupported database driver: %s", config.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	DB = db

	// Run migrations
	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database initialized successfully")
	return nil
}

// runMigrations runs all database migrations
func runMigrations() error {
	return DB.AutoMigrate(
		&Cluster{},
		&Node{},
		&Event{},
		&SSHKey{},
		&User{},
		&Job{},
	)
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}
