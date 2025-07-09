package p2p

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/ShreyamKundu/peernet/peer/file"
	pb "github.com/ShreyamKundu/peernet/proto"
	"google.golang.org/grpc"
)

// Server implements the gRPC PeerService.
type Server struct {
	pb.UnimplementedPeerServiceServer
	filePath string
	chunks   map[int][]byte // In a real app, you'd read from disk on demand.
}

// NewGRPCServer creates a new gRPC server instance.
func NewGRPCServer(filePath string, chunks []file.ChunkInfo) *Server {
	chunkMap := make(map[int][]byte)
	for _, c := range chunks {
		chunkMap[c.Index] = c.Data
	}
	return &Server{
		filePath: filePath,
		chunks:   chunkMap,
	}
}

// DownloadChunk serves a requested file chunk.
func (s *Server) DownloadChunk(ctx context.Context, in *pb.ChunkRequest) (*pb.ChunkResponse, error) {
	log.Printf("Received request for chunk %d of file %s", in.GetChunkIndex(), in.GetFileHash())

	chunkIndex := int(in.GetChunkIndex())
	data, ok := s.chunks[chunkIndex]
	if !ok {
		return nil, fmt.Errorf("chunk %d not found for file %s", chunkIndex, in.GetFileHash())
	}

	return &pb.ChunkResponse{ChunkData: data}, nil
}

// Start begins listening for gRPC requests.
func (s *Server) Start(port string) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterPeerServiceServer(grpcServer, s)
	return grpcServer.Serve(lis)
}
