package db

import (
	"database/sql"
	"log"
)


func InitDatabase(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	log.Println("Successfully connected to the database.")
	if err = createSchema(db); err != nil {
		return nil, err
	}

	return db, nil
}


// createSchema creates the necessary tables for the tracker.
func createSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS peers (
		id UUID PRIMARY KEY,
		address TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		reputation_score FLOAT DEFAULT 1.0,
		token_balance INT DEFAULT 100,
		last_seen TIMESTAMPTZ DEFAULT NOW(),
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS files (
		file_hash TEXT PRIMARY KEY,
		file_name TEXT NOT NULL,
		total_chunks INT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS file_chunk_peers (
		file_hash TEXT NOT NULL REFERENCES files(file_hash) ON DELETE CASCADE,
		chunk_index INT NOT NULL,
		peer_id UUID NOT NULL REFERENCES peers(id) ON DELETE CASCADE,
		PRIMARY KEY (file_hash, chunk_index, peer_id)
	);

	CREATE TABLE IF NOT EXISTS reputation_events (
		id SERIAL PRIMARY KEY,
		reporter_peer_id UUID NOT NULL REFERENCES peers(id) ON DELETE CASCADE,
		target_peer_id UUID NOT NULL REFERENCES peers(id) ON DELETE CASCADE,
		file_hash TEXT NOT NULL,
		chunk_index INT NOT NULL,
		event_type TEXT NOT NULL, -- 'SUCCESS_UPLOAD', 'FAILED_UPLOAD'
		created_at TIMESTAMPTZ DEFAULT NOW(),
		processed BOOLEAN DEFAULT FALSE
	);
	`
	_, err := db.Exec(schema)
	if err != nil {
		log.Printf("Error creating database schema: %v", err)
		return err
	}

	log.Println("Database schema is ready.")
	return nil
}
