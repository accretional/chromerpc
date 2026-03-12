// Package filesystem implements the gRPC FileSystemService by bridging to CDP.
package filesystem

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/filesystem"
)

type Server struct {
	pb.UnimplementedFileSystemServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) GetDirectory(ctx context.Context, req *pb.GetDirectoryRequest) (*pb.GetDirectoryResponse, error) {
	locator := map[string]interface{}{
		"storageKey":     req.BucketFileSystemLocator.GetStorageKey(),
		"pathComponents": req.BucketFileSystemLocator.GetPathComponents(),
	}
	if req.BucketFileSystemLocator.BucketName != nil {
		locator["bucketName"] = *req.BucketFileSystemLocator.BucketName
	}
	params := map[string]interface{}{
		"bucketFileSystemLocator": locator,
	}
	result, err := s.client.Send(ctx, "FileSystem.getDirectory", params)
	if err != nil {
		return nil, fmt.Errorf("FileSystem.getDirectory: %w", err)
	}
	var resp struct {
		Directory json.RawMessage `json:"directory"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("FileSystem.getDirectory: unmarshal: %w", err)
	}
	return &pb.GetDirectoryResponse{DirectoryJson: resp.Directory}, nil
}
