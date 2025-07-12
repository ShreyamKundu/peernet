package file

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// ChunkSize defines the fixed size of each file chunk in bytes.
const ChunkSize = 1 * 1024 * 1024 // 1 MB

// ChunkInfo holds metadata about a single file chunk.
type ChunkInfo struct {
	Index int
	Hash  string
	Data  []byte // Note: In a real large-file app, you wouldn't hold all data in memory for sharing server.
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
			return nil, "", err // Return error if reading fails
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

// WriteChunkAtOffset writes a chunk of data to a file at a specific byte offset.
// It creates the file if it doesn't exist or truncates it if it's smaller than needed.
// This function is designed for incremental writing of file chunks.
func WriteChunkAtOffset(filePath string, data []byte, chunkIndex int) error {
	// Open the file for reading and writing. Create if not exists, truncate if needed.
	// os.O_CREATE: Create the file if it does not exist.
	// os.O_RDWR: Open the file for reading and writing.
	// os.O_TRUNC: If the file exists and is a regular file, it is truncated to zero size.
	//             This is important for ensuring the file is the correct size if it was partially downloaded before.
	//             However, for incremental writing, we need to be careful not to truncate on every write.
	//             Instead, we open for RDWR and rely on Seek and WriteAt.
	//             We'll ensure the file size is correct at the start of the download process.

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	offset := int64(chunkIndex) * ChunkSize
	n, err := file.WriteAt(data, offset)
	if err != nil {
		return fmt.Errorf("failed to write chunk at offset %d: %v", offset, err)
	}
	if n != len(data) {
		return fmt.Errorf("failed to write all bytes of chunk: wrote %d, expected %d", n, len(data))
	}

	return nil
}
