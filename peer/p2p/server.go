package p2p

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/ShreyamKundu/peernet/peer/file" // Import the file package to access ChunkSize and VerifyChunk
	pb "github.com/ShreyamKundu/peernet/proto"
	"google.golang.org/grpc"
)

// Server implements the gRPC PeerService.
type Server struct {
	pb.UnimplementedPeerServiceServer
	sharedFilePath string              // The full path to the file being served
	fileHash       string              // The overall hash of the file being served
	totalChunks    int                 // Total number of chunks for the file
	chunkHashes    map[int]string      // Maps chunkIndex to its expected chunkHash (metadata only)
}

// NewGRPCServer creates a new gRPC server instance.
// It now takes the full file path, its overall hash, total chunks, and a slice of ChunkInfo
// (which contains chunk indices and their hashes, but not the actual data for the server to store).
func NewGRPCServer(filePath, fileHash string, totalChunks int, chunks []file.ChunkInfo) *Server {
	chunkMap := make(map[int]string)
	for _, c := range chunks {
		chunkMap[c.Index] = c.Hash // Store only the chunk hash, not the data
	}
	return &Server{
		sharedFilePath: filePath,
		fileHash:       fileHash,
		totalChunks:    totalChunks,
		chunkHashes:    chunkMap,
	}
}

// DownloadChunk serves a requested file chunk by reading it directly from disk.
func (s *Server) DownloadChunk(ctx context.Context, in *pb.ChunkRequest) (*pb.ChunkResponse, error) {
	log.Printf("Received request for chunk %d of file %s", in.GetChunkIndex(), in.GetFileHash())

	requestedFileHash := in.GetFileHash()
	requestedChunkIndex := int(in.GetChunkIndex())

	// 1. Validate the request against the file this server is sharing
	if requestedFileHash != s.fileHash {
		return nil, fmt.Errorf("this peer is not sharing file with hash %s", requestedFileHash)
	}

	if requestedChunkIndex < 0 || requestedChunkIndex >= s.totalChunks {
		return nil, fmt.Errorf("invalid chunk index %d for file %s (total chunks: %d)", requestedChunkIndex, requestedFileHash, s.totalChunks)
	}

	// Get the expected hash for verification
	expectedChunkHash, ok := s.chunkHashes[requestedChunkIndex]
	if !ok {
		// This indicates an inconsistency in the server's file metadata
		return nil, fmt.Errorf("metadata for chunk %d of file %s not found on this peer", requestedChunkIndex, requestedFileHash)
	}

	// 2. Open the file from disk
	fileHandle, err := os.Open(s.sharedFilePath)
	if err != nil {
		log.Printf("Error opening shared file %s: %v", s.sharedFilePath, err)
		return nil, fmt.Errorf("failed to open shared file on disk")
	}
	defer fileHandle.Close()

	// 3. Calculate the offset and read the chunk data
	offset := int64(requestedChunkIndex) * file.ChunkSize // Use the exported ChunkSize
	buffer := make([]byte, file.ChunkSize)

	bytesRead, err := fileHandle.ReadAt(buffer, offset)
	if err != nil && err != io.EOF {
		log.Printf("Error reading chunk %d from file %s at offset %d: %v", requestedChunkIndex, s.sharedFilePath, offset, err)
		return nil, fmt.Errorf("failed to read chunk data from disk")
	}

	chunkData := buffer[:bytesRead]

	// 4. Verify the chunk data's integrity before sending
	if !file.VerifyChunk(chunkData, expectedChunkHash) {
		log.Printf("Chunk %d of file %s hash mismatch. Expected %s, calculated %s.", requestedChunkIndex, requestedFileHash, expectedChunkHash, file.CalculateChunkHash(chunkData))
		return nil, fmt.Errorf("chunk data integrity check failed on server side")
	}

	return &pb.ChunkResponse{ChunkData: chunkData}, nil
}

// Start begins listening for gRPC requests.
func (s *Server) Start(port string) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterPeerServiceServer(grpcServer, s)
	log.Printf("gRPC server started on port %s, serving file %s (hash: %s)", port, s.sharedFilePath, s.fileHash)
	return grpcServer.Serve(lis)
}
