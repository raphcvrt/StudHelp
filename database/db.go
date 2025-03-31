package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB is a global database connection that can be used throughout the application
var DB *sql.DB

// InitDB initializes the database connection
func InitDB() (*sql.DB, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/forum.db"
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL&_timeout=5000")
	if err == nil {
		db.SetConnMaxLifetime(time.Minute * 3)
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(10)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set the global DB variable for use by handlers
	DB = db

	return db, nil
}

// GetDB returns the global database connection
func GetDB() *sql.DB {
	return DB
}

// RunMigrations runs the database migration
func RunMigrations(db *sql.DB) error {
	migrationPath := "./database/000_initial_schema.sql"
	log.Printf("Applying initial database schema")

	migration, err := os.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Désactivez le mode transaction automatique
	db.SetMaxOpenConns(1)

	// Exécutez directement les commandes SQL sans transaction explicite
	if _, err := db.Exec(string(migration)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	log.Printf("Database schema applied successfully")
	return nil
}
