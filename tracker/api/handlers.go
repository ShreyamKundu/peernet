package api

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ShreyamKundu/peernet/tracker/auth"
	"golang.org/x/crypto/bcrypt"
)

type peerRegistrationRequest struct {
	Address  string `json:"address" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type fileAnnouncementRequest struct {
	FileHash    string `json:"file_hash" binding:"required"`
	FileName    string `json:"file_name" binding:"required"`
	TotalChunks int    `json:"total_chunks" binding:"required"`
	ChunkIndex  int    `json:"chunk_index"`
	ChunkHash   string `json:"chunk_hash" binding:"required"` // ADDED: Required for chunk verification
}

type feedbackRequest struct {
	TargetPeerID string `json:"target_peer_id" binding:"required"`
	FileHash     string `json:"file_hash" binding:"required"`
	ChunkIndex   int    `json:"chunk_index"`
	EventType    string `json:"event_type" binding:"required"` // e.g., 'SUCCESS_UPLOAD', 'FAILED_UPLOAD'
}

// RegisterRoutes registers all API routes.
func RegisterRoutes(router *gin.RouterGroup, db *sql.DB, jwtSecret string) {
	// Public route
	router.POST("/peers/register", registerPeer(db, jwtSecret))

	// Authenticated routes
	authed := router.Group("/")
	authed.Use(AuthMiddleware(jwtSecret))
	{
		authed.POST("/files/announce", announceFile(db))
		authed.GET("/files/lookup/:fileHash", lookupFile(db))
		authed.POST("/peers/feedback", submitFeedback(db))
	}
}

func registerPeer(db *sql.DB, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req peerRegistrationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}

		peerID := uuid.New()
		_, err = db.Exec("INSERT INTO peers (id, address, password_hash) VALUES ($1, $2, $3)", peerID, req.Address, string(hashedPassword))
		if err != nil {
			log.Printf("Failed to register peer: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not register peer"})
			return
		}

		token, err := auth.GenerateToken(peerID.String(), jwtSecret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"peer_id": peerID, "token": token})
	}
}

func announceFile(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req fileAnnouncementRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		peerID, _ := c.Get("peerID")

		// Use a transaction
		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		defer tx.Rollback() // Rollback on error

		// Insert or update file info
		_, err = tx.Exec(`
            INSERT INTO files (file_hash, file_name, total_chunks) VALUES ($1, $2, $3)
            ON CONFLICT (file_hash) DO NOTHING;`, req.FileHash, req.FileName, req.TotalChunks)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to announce file"})
			return
		}

		// Insert chunk-peer mapping with chunk_hash
		_, err = tx.Exec(`
            INSERT INTO file_chunk_peers (file_hash, chunk_index, peer_id, chunk_hash)
            VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING;`, req.FileHash, req.ChunkIndex, peerID.(string), req.ChunkHash) // ADDED req.ChunkHash
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to announce chunk"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "announced"})
	}
}

// PeerInfo holds information about a peer that has a chunk.
type PeerInfo struct {
	ID              string  `json:"id"`
	Address         string  `json:"address"`
	ReputationScore float64 `json:"reputation_score"`
}

// ChunkLookupInfo holds information for a specific chunk, including its hash and available peers.
type ChunkLookupInfo struct {
	ChunkHash string     `json:"chunk_hash"`
	Peers     []PeerInfo `json:"peers"`
}

func lookupFile(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		fileHash := c.Param("fileHash")
		rows, err := db.Query(`
            SELECT p.id, p.address, fcp.chunk_index, p.reputation_score, fcp.chunk_hash -- ADDED fcp.chunk_hash
            FROM file_chunk_peers fcp
            JOIN peers p ON fcp.peer_id = p.id
            WHERE fcp.file_hash = $1
            ORDER BY fcp.chunk_index ASC, p.reputation_score DESC, p.last_seen DESC; -- Order by chunk_index first for consistency
        `, fileHash)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
			return
		}
		defer rows.Close()

		// map[chunk_index] -> ChunkLookupInfo
		chunkPeers := make(map[int]ChunkLookupInfo)

		for rows.Next() {
			var peerID, address, chunkHash string // ADDED chunkHash
			var chunkIndex int
			var reputationScore float64
			if err := rows.Scan(&peerID, &address, &chunkIndex, &reputationScore, &chunkHash); err != nil { // ADDED &chunkHash
				log.Printf("Error scanning lookup row: %v", err)
				continue
			}

			// Get or create the ChunkLookupInfo for this chunkIndex
			chunkInfo := chunkPeers[chunkIndex]
			if chunkInfo.Peers == nil { // Initialize if first peer for this chunk
				chunkInfo.Peers = make([]PeerInfo, 0)
				chunkInfo.ChunkHash = chunkHash // Set the chunk hash for this chunk index
			}
			chunkInfo.Peers = append(chunkInfo.Peers, PeerInfo{ID: peerID, Address: address, ReputationScore: reputationScore})
			chunkPeers[chunkIndex] = chunkInfo
		}

		c.JSON(http.StatusOK, gin.H{"chunks": chunkPeers})
	}
}

func submitFeedback(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req feedbackRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		reporterPeerID, _ := c.Get("peerID")

		_, err := db.Exec(`
            INSERT INTO reputation_events (reporter_peer_id, target_peer_id, file_hash, chunk_index, event_type)
            VALUES ($1, $2, $3, $4, $5);
        `, reporterPeerID, req.TargetPeerID, req.FileHash, req.ChunkIndex, req.EventType)

		if err != nil {
			log.Printf("Failed to record feedback: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not record feedback"})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{"status": "feedback received"})
	}
}
