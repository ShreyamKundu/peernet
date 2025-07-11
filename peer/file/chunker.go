package file

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// ChunkSize defines the fixed size of each file chunk in bytes.
const ChunkSize = 1 * 1024 * 1024 // 1 MB

// ChunkInfo holds metadata about a single file chunk.
type ChunkInfo struct {
	Index int
	Hash  string
	Data  []byte 
}

// ChunkFile splits a file into chunks and returns their info and the overall file hash.
// The 'Data' field in ChunkInfo will contain the actual chunk data when returned.
func ChunkFile(filePath string) ([]ChunkInfo, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	// Calculate overall file hash
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, "", err
	}
	fileHash := hex.EncodeToString(hasher.Sum(nil))

	// Seek back to the beginning of the file to read chunks
	file.Seek(0, 0)

	var chunks []ChunkInfo
	buffer := make([]byte, ChunkSize) // Use exported ChunkSize
	chunkIndex := 0

	for {
		bytesRead, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err.Error(), err // Return error if reading fails
		}

		chunkData := make([]byte, bytesRead) // Create a slice of exact size
		copy(chunkData, buffer[:bytesRead])

		chunkHasher := sha256.New()
		chunkHasher.Write(chunkData)
		chunkHash := hex.EncodeToString(chunkHasher.Sum(nil))

		chunks = append(chunks, ChunkInfo{
			Index: chunkIndex,
			Hash:  chunkHash,
			Data:  chunkData,
		})
		chunkIndex++
	}

	return chunks, fileHash, nil
}

// VerifyChunk compares the hash of data with an expected hash.
func VerifyChunk(data []byte, expectedHash string) bool {
	return CalculateChunkHash(data) == expectedHash
}

// CalculateChunkHash computes the SHA256 hash of a byte slice.
func CalculateChunkHash(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}
