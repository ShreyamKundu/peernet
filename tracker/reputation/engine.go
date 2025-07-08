package reputation

import (
	"database/sql"
	"log"
	"time"

	"github.com/lib/pq"
)

const (
	processingInterval = 1 * time.Minute // How often to process events
	scoreForSuccess    = 0.1
	scoreForFailure    = -0.5
	tokenForSuccess    = 5
	tokenForFailure    = -10
)

// Engine processes reputation events and updates peer scores.
type Engine struct {
	db     *sql.DB
	ticker *time.Ticker
	done   chan bool
}

// NewEngine creates a new reputation engine.
func NewEngine(db *sql.DB) *Engine {
	return &Engine{
		db:   db,
		done: make(chan bool),
	}
}

// Start begins the periodic processing of reputation events.
func (e *Engine) Start() {
	log.Println("Starting reputation engine...")
	e.ticker = time.NewTicker(processingInterval)
	for {
		select {
		case <-e.done:
			e.ticker.Stop()
			log.Println("Reputation engine stopped.")
			return
		case <-e.ticker.C:
			log.Println("Processing reputation events...")
			if err := e.processEvents(); err != nil {
				log.Printf("Error processing reputation events: %v", err)
			}
		}
	}
}

// Stop halts the reputation engine.
func (e *Engine) Stop() {
	e.done <- true
}

// processEvents fetches unprocessed events and updates peer scores and tokens.
func (e *Engine) processEvents() error {
	tx, err := e.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Rollback on error

	rows, err := tx.Query("SELECT id, target_peer_id, event_type FROM reputation_events WHERE processed = FALSE")
	if err != nil {
		return err
	}
	defer rows.Close()

	var eventIDsToMark []int
	peerUpdates := make(map[string]struct {
		scoreChange float64
		tokenChange int
	})

	for rows.Next() {
		var eventID, tokenChange int
		var targetPeerID, eventType string
		var scoreChange float64

		if err := rows.Scan(&eventID, &targetPeerID, &eventType); err != nil {
			log.Printf("Error scanning event row: %v", err)
			continue
		}

		switch eventType {
		case "SUCCESS_UPLOAD":
			scoreChange = scoreForSuccess
			tokenChange = tokenForSuccess
		case "FAILED_UPLOAD":
			scoreChange = scoreForFailure
			tokenChange = tokenForFailure
		}

		update := peerUpdates[targetPeerID]
		update.scoreChange += scoreChange
		update.tokenChange += tokenChange
		peerUpdates[targetPeerID] = update
		eventIDsToMark = append(eventIDsToMark, eventID)
	}

	for peerID, update := range peerUpdates {
		_, err := tx.Exec(`
			UPDATE peers
			SET reputation_score = reputation_score + $1, token_balance = token_balance + $2
			WHERE id = $3
		`, update.scoreChange, update.tokenChange, peerID)
		if err != nil {
			log.Printf("Failed to update peer %s: %v", peerID, err)
			// Continue to process other peers
		}
	}

	if len(eventIDsToMark) > 0 {
		// Mark events as processed
		stmt, err := tx.Prepare("UPDATE reputation_events SET processed = TRUE WHERE id = ANY($1::int[])")
		if err != nil {
			return err
		}
		defer stmt.Close()
		_, err = stmt.Exec(pq.Array(eventIDsToMark))
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
