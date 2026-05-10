package database

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB(dataSourceName string) {
	var err error
	DB, err = sql.Open("sqlite3", dataSourceName+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// SQLite supports concurrent reads but only one writer at a time
	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(time.Hour)

	if err = DB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	DB.Exec("PRAGMA foreign_keys=ON;")

	createTables()
	log.Println("Database initialized successfully")
}

func createTables() {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS matches (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		creator_id TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'waiting',
		current_round INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (creator_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS match_players (
		match_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		active INTEGER NOT NULL DEFAULT 1,
		joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (match_id, user_id),
		FOREIGN KEY (match_id) REFERENCES matches(id),
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS join_requests (
		id TEXT PRIMARY KEY,
		match_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		user_name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (match_id) REFERENCES matches(id),
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS invites (
		id TEXT PRIMARY KEY,
		match_id TEXT NOT NULL,
		match_name TEXT NOT NULL,
		inviter_name TEXT NOT NULL,
		target_user_id TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (match_id) REFERENCES matches(id),
		FOREIGN KEY (target_user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS rounds (
		id TEXT PRIMARY KEY,
		match_id TEXT NOT NULL,
		round_number INTEGER NOT NULL,
		letter TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'playing',
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		ends_at DATETIME,
		FOREIGN KEY (match_id) REFERENCES matches(id)
	);

	CREATE TABLE IF NOT EXISTS answers (
		id TEXT PRIMARY KEY,
		round_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		color TEXT,
		fruit TEXT,
		object TEXT,
		movie TEXT,
		city TEXT,
		score INTEGER DEFAULT 0,
		submitted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (round_id) REFERENCES rounds(id),
		FOREIGN KEY (user_id) REFERENCES users(id),
		UNIQUE(round_id, user_id)
	);
	`

	_, err := DB.Exec(schema)
	if err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}
}

// Cleanup removes old finished matches and expired data
func Cleanup() {
	// Remove finished matches older than 24 hours and their related data
	cutoff := time.Now().Add(-24 * time.Hour)

	// Get old match IDs
	rows, err := DB.Query("SELECT id FROM matches WHERE status = 'finished' AND created_at < ?", cutoff)
	if err != nil {
		return
	}
	defer rows.Close()

	var matchIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			matchIDs = append(matchIDs, id)
		}
	}

	for _, matchID := range matchIDs {
		DB.Exec("DELETE FROM answers WHERE round_id IN (SELECT id FROM rounds WHERE match_id = ?)", matchID)
		DB.Exec("DELETE FROM rounds WHERE match_id = ?", matchID)
		DB.Exec("DELETE FROM match_players WHERE match_id = ?", matchID)
		DB.Exec("DELETE FROM join_requests WHERE match_id = ?", matchID)
		DB.Exec("DELETE FROM invites WHERE match_id = ?", matchID)
		DB.Exec("DELETE FROM matches WHERE id = ?", matchID)
	}

	// Remove expired/rejected invites older than 1 hour
	DB.Exec("DELETE FROM invites WHERE status != 'pending' AND created_at < ?", time.Now().Add(-1*time.Hour))

	// Remove processed join requests older than 1 hour
	DB.Exec("DELETE FROM join_requests WHERE status != 'pending' AND created_at < ?", time.Now().Add(-1*time.Hour))
}
