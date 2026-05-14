package database

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://adedanha:adedanha@db:5432/adedanha?sslmode=disable"
	}

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	DB.SetMaxOpenConns(20)
	DB.SetMaxIdleConns(10)
	DB.SetConnMaxLifetime(time.Hour)

	// Retry connection (wait for postgres to be ready)
	for i := 0; i < 30; i++ {
		if err = DB.Ping(); err == nil {
			break
		}
		log.Printf("Waiting for database... (%d/30)", i+1)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to database after 30 attempts: %v", err)
	}

	createTables()
	createIndexes()
	log.Println("Database initialized successfully")
}

func createTables() {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		avatar TEXT DEFAULT '',
		created_at TIMESTAMP DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS matches (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		creator_id TEXT NOT NULL REFERENCES users(id),
		status TEXT NOT NULL DEFAULT 'waiting',
		current_round INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS match_players (
		match_id TEXT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
		user_id TEXT NOT NULL REFERENCES users(id),
		active BOOLEAN NOT NULL DEFAULT TRUE,
		joined_at TIMESTAMP DEFAULT NOW(),
		PRIMARY KEY (match_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS join_requests (
		id TEXT PRIMARY KEY,
		match_id TEXT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
		user_id TEXT NOT NULL REFERENCES users(id),
		user_name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS invites (
		id TEXT PRIMARY KEY,
		match_id TEXT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
		match_name TEXT NOT NULL,
		inviter_name TEXT NOT NULL,
		target_user_id TEXT NOT NULL REFERENCES users(id),
		status TEXT NOT NULL DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS rounds (
		id TEXT PRIMARY KEY,
		match_id TEXT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
		round_number INTEGER NOT NULL,
		letter TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'playing',
		started_at TIMESTAMP DEFAULT NOW(),
		ends_at TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS answers (
		id TEXT PRIMARY KEY,
		round_id TEXT NOT NULL REFERENCES rounds(id) ON DELETE CASCADE,
		user_id TEXT NOT NULL REFERENCES users(id),
		color TEXT,
		fruit TEXT,
		object TEXT,
		movie TEXT,
		city TEXT,
		animal TEXT,
		name TEXT,
		score INTEGER DEFAULT 0,
		submitted_at TIMESTAMP DEFAULT NOW(),
		UNIQUE(round_id, user_id)
	);
	`

	_, err := DB.Exec(schema)
	if err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}
}

func createIndexes() {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_match_players_user_active ON match_players(user_id, active)",
		"CREATE INDEX IF NOT EXISTS idx_matches_status ON matches(status)",
		"CREATE INDEX IF NOT EXISTS idx_matches_creator ON matches(creator_id)",
		"CREATE INDEX IF NOT EXISTS idx_rounds_match_id ON rounds(match_id)",
		"CREATE INDEX IF NOT EXISTS idx_answers_round_id ON answers(round_id)",
		"CREATE INDEX IF NOT EXISTS idx_invites_target_status ON invites(target_user_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_join_requests_match_status ON join_requests(match_id, status)",
	}

	for _, idx := range indexes {
		if _, err := DB.Exec(idx); err != nil {
			log.Printf("Warning: failed to create index: %v", err)
		}
	}
}

// Cleanup removes old finished matches and expired data
func Cleanup() {
	tx, err := DB.Begin()
	if err != nil {
		log.Printf("Cleanup: failed to begin transaction: %v", err)
		return
	}
	defer tx.Rollback()

	cutoff := time.Now().Add(-24 * time.Hour)

	// With ON DELETE CASCADE, deleting matches cascades to related tables
	result, err := tx.Exec("DELETE FROM matches WHERE status = 'finished' AND created_at < $1", cutoff)
	if err != nil {
		log.Printf("Cleanup: failed to delete old matches: %v", err)
		return
	}
	if rows, _ := result.RowsAffected(); rows > 0 {
		log.Printf("Cleanup: deleted %d old finished matches", rows)
	}

	// Remove expired/rejected invites older than 1 hour
	tx.Exec("DELETE FROM invites WHERE status != 'pending' AND created_at < $1", time.Now().Add(-1*time.Hour))

	// Remove processed join requests older than 1 hour
	tx.Exec("DELETE FROM join_requests WHERE status != 'pending' AND created_at < $1", time.Now().Add(-1*time.Hour))

	if err := tx.Commit(); err != nil {
		log.Printf("Cleanup: failed to commit: %v", err)
	}
}
