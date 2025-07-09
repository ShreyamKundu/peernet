package file

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

const chunkSize = 1 * 1024 * 1024 // 1 MB

// ChunkInfo holds metadata about a single file chunk.
type ChunkInfo struct {
	Index int
	Hash  string
	Data  []byte
}

// ChunkFile splits a file into chunks and returns their info and the overall file hash.
func ChunkFile(filePath string) ([]ChunkInfo, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, "", err
	}
	fileHash := hex.EncodeToString(hasher.Sum(nil))

	// Seek back to the beginning of the file to read chunks
	file.Seek(0, 0)

	var chunks []ChunkInfo
	buffer := make([]byte, chunkSize)
	chunkIndex := 0

	for {
		bytesRead, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", err
		}

		chunkData := buffer[:bytesRead]
		chunkHasher := sha256.New()
		chunkHasher.Write(chunkData)
		chunkHash := hex.EncodeToString(chunkHasher.Sum(nil))

		chunks = append(chunks, ChunkInfo{
			Index: chunkIndex,
			Hash:  chunkHash,
			Data:  chunkData, // Note: In a real large-file app, you wouldn't hold all data in memory.
		})
		chunkIndex++
	}

	return chunks, fileHash, nil
}

// VerifyChunk compares the hash of data with an expected hash.
func VerifyChunk(data []byte, expectedHash string) bool {
	hasher := sha256.New()
	hasher.Write(data)
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	return actualHash == expectedHash
}
