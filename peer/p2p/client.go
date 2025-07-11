package p2p

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/ShreyamKundu/peernet/peer/file"
	pb "github.com/ShreyamKundu/peernet/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TrackerClient communicates with the tracker's REST API.
type TrackerClient struct {
	baseURL string
	token   string
	client  *http.Client
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

// LookupResult is the structure of the response from the /lookup endpoint.
type LookupResult struct {
	Chunks map[int]ChunkLookupInfo `json:"chunks"` // CHANGED: Now maps chunk index to ChunkLookupInfo
}

// NewTrackerClient creates a new client for the tracker.
func NewTrackerClient(baseURL, token string) *TrackerClient {
	return &TrackerClient{
		baseURL: baseURL,
		token:   token,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// Announce tells the tracker that this peer has a specific chunk.
// It now includes the chunkHash for verification by downloaders.
func (c *TrackerClient) Announce(filePath, fileHash string, totalChunks, chunkIndex int, chunkHash string) error {
	payload := map[string]interface{}{
		"file_hash":    fileHash,
		"file_name":    filepath.Base(filePath),
		"total_chunks": totalChunks,
		"chunk_index":  chunkIndex,
		"chunk_hash":   chunkHash, // ADDED: Send the chunk hash to the tracker
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/files/announce", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("announce failed with status: %s, body: %s", resp.Status, string(bodyBytes))
	}
	return nil
}

// Lookup asks the tracker for peers that have chunks for a given file hash,
// now including the expected chunk hashes.
func (c *TrackerClient) Lookup(fileHash string) (*LookupResult, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/v1/files/lookup/"+fileHash, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("lookup failed with status: %s, body: %s", resp.Status, string(bodyBytes))
	}

	var result LookupResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SubmitFeedback sends a performance report to the tracker.
func (c *TrackerClient) SubmitFeedback(targetPeerID, fileHash string, chunkIndex int, eventType string) {
	payload := map[string]interface{}{
		"target_peer_id": targetPeerID,
		"file_hash":      fileHash,
		"chunk_index":    chunkIndex,
		"event_type":     eventType,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/peers/feedback", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error creating feedback request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		log.Printf("Error submitting feedback: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Feedback submission failed: %s - %s", resp.Status, string(bodyBytes))
	}
}

// Downloader manages the concurrent download of file chunks.
type Downloader struct {
	trackerClient *TrackerClient
}

func NewDownloader(client *TrackerClient) *Downloader {
	return &Downloader{trackerClient: client}
}

// DownloadFile coordinates the entire file download process, now with chunk verification.
func (d *Downloader) DownloadFile(fileHash string, lookupResult *LookupResult) ([]byte, error) {
	totalChunks := len(lookupResult.Chunks)
	if totalChunks == 0 {
		return nil, fmt.Errorf("no chunks available for file")
	}

	// Use a map to store chunks as they are downloaded, to handle out-of-order completion
	downloadedChunks := make(map[int][]byte)
	var wg sync.WaitGroup
	var mu sync.Mutex // Mutex to protect access to downloadedChunks and errs channel
	errs := make(chan error, totalChunks)

	for i := 0; i < totalChunks; i++ {
		wg.Add(1)
		go func(chunkIndex int) {
			defer wg.Done()

			chunkLookupInfo, ok := lookupResult.Chunks[chunkIndex]
			if !ok || len(chunkLookupInfo.Peers) == 0 {
				mu.Lock()
				errs <- fmt.Errorf("no peers or chunk hash found for chunk %d", chunkIndex)
				mu.Unlock()
				return
			}
			peers := chunkLookupInfo.Peers
			expectedChunkHash := chunkLookupInfo.ChunkHash // Get the expected hash for this chunk from tracker response

			// Try peers in order of reputation
			for _, peer := range peers {
				log.Printf("Attempting to download chunk %d from peer %s (%s)", chunkIndex, peer.ID, peer.Address)
				data, err := downloadChunkFromPeer(peer, fileHash, chunkIndex)
				if err != nil {
					log.Printf("Failed to download chunk %d from %s: %v. Trying next peer.", chunkIndex, peer.Address, err)
					d.trackerClient.SubmitFeedback(peer.ID, fileHash, chunkIndex, "FAILED_UPLOAD")
					continue
				}

				if !file.VerifyChunk(data, expectedChunkHash) {
					log.Printf("Downloaded chunk %d from %s failed hash verification. Expected %s, got data with hash %s. Trying next peer.",
						chunkIndex, peer.Address, expectedChunkHash, file.CalculateChunkHash(data))
					d.trackerClient.SubmitFeedback(peer.ID, fileHash, chunkIndex, "FAILED_UPLOAD")
					continue // Try next peer if verification fails
				}

				mu.Lock()
				downloadedChunks[chunkIndex] = data
				mu.Unlock()

				log.Printf("Successfully downloaded and verified chunk %d from peer %s", chunkIndex, peer.ID)
				d.trackerClient.SubmitFeedback(peer.ID, fileHash, chunkIndex, "SUCCESS_UPLOAD")
				return // Success, exit the loop for this chunk
			}
			mu.Lock()
			errs <- fmt.Errorf("failed to download and verify chunk %d from any peer", chunkIndex)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	close(errs)

	// Check for any errors that occurred during concurrent downloads
	for err := range errs {
		if err != nil {
			return nil, err // Return on the first error
		}
	}

	// Reassemble the file from chunks in correct order
	var fileBuffer bytes.Buffer
	for i := 0; i < totalChunks; i++ {
		mu.Lock() // Protect access to downloadedChunks map
		chunk, ok := downloadedChunks[i]
		mu.Unlock()
		if !ok || chunk == nil {
			return nil, fmt.Errorf("missing chunk %d after download completion", i)
		}
		fileBuffer.Write(chunk)
	}

	return fileBuffer.Bytes(), nil
}

// downloadChunkFromPeer connects to a single peer via gRPC and downloads one chunk.
func downloadChunkFromPeer(peer PeerInfo, fileHash string, chunkIndex int) ([]byte, error) {
	// Using WithTransportCredentials(insecure.NewCredentials()) for simplicity in demo.
	// In production, this should be grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, ""))
	conn, err := grpc.NewClient(peer.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("did not connect to peer %s (%s): %v", peer.ID, peer.Address, err)
	}
	defer conn.Close()

	c := pb.NewPeerServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	r, err := c.DownloadChunk(ctx, &pb.ChunkRequest{FileHash: fileHash, ChunkIndex: int32(chunkIndex)})
	if err != nil {
		return nil, fmt.Errorf("could not download chunk %d from peer %s: %v", chunkIndex, peer.ID, err)
	}

	return r.GetChunkData(), nil
}
