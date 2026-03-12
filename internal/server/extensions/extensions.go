// Package extensions implements the gRPC ExtensionsService by bridging to CDP.
package extensions

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/extensions"
)

// Server implements the cdp.extensions.ExtensionsService gRPC service.
type Server struct {
	pb.UnimplementedExtensionsServiceServer
	client *cdpclient.Client
}

// New creates a new Extensions gRPC server backed by the given CDP client.
func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

// send routes a CDP command through the specified session, falling back
// to the client's default session if sessionID is empty.
func (s *Server) send(ctx context.Context, sessionID string, method string, params interface{}) (json.RawMessage, error) {
	if sessionID != "" {
		return s.client.SendWithSession(ctx, method, params, sessionID)
	}
	return s.client.Send(ctx, method, params)
}


func (s *Server) LoadUnpacked(ctx context.Context, req *pb.LoadUnpackedRequest) (*pb.LoadUnpackedResponse, error) {
	params := map[string]interface{}{
		"path": req.Path,
	}
	result, err := s.send(ctx, req.SessionId, "Extensions.loadUnpacked", params)
	if err != nil {
		return nil, fmt.Errorf("Extensions.loadUnpacked: %w", err)
	}
	var resp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Extensions.loadUnpacked: unmarshal: %w", err)
	}
	return &pb.LoadUnpackedResponse{Id: resp.ID}, nil
}

func (s *Server) GetStorageItems(ctx context.Context, req *pb.GetStorageItemsRequest) (*pb.GetStorageItemsResponse, error) {
	params := map[string]interface{}{
		"id":          req.Id,
		"storageArea": req.StorageArea,
	}
	if len(req.Keys) > 0 {
		params["keys"] = req.Keys
	}
	result, err := s.send(ctx, req.SessionId, "Extensions.getStorageItems", params)
	if err != nil {
		return nil, fmt.Errorf("Extensions.getStorageItems: %w", err)
	}
	var resp struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Extensions.getStorageItems: unmarshal: %w", err)
	}
	return &pb.GetStorageItemsResponse{Data: resp.Data}, nil
}

func (s *Server) RemoveStorageItems(ctx context.Context, req *pb.RemoveStorageItemsRequest) (*pb.RemoveStorageItemsResponse, error) {
	params := map[string]interface{}{
		"id":          req.Id,
		"storageArea": req.StorageArea,
		"keys":        req.Keys,
	}
	if _, err := s.send(ctx, req.SessionId, "Extensions.removeStorageItems", params); err != nil {
		return nil, fmt.Errorf("Extensions.removeStorageItems: %w", err)
	}
	return &pb.RemoveStorageItemsResponse{}, nil
}

func (s *Server) ClearStorageItems(ctx context.Context, req *pb.ClearStorageItemsRequest) (*pb.ClearStorageItemsResponse, error) {
	params := map[string]interface{}{
		"id":          req.Id,
		"storageArea": req.StorageArea,
	}
	if _, err := s.send(ctx, req.SessionId, "Extensions.clearStorageItems", params); err != nil {
		return nil, fmt.Errorf("Extensions.clearStorageItems: %w", err)
	}
	return &pb.ClearStorageItemsResponse{}, nil
}

func (s *Server) SetStorageItems(ctx context.Context, req *pb.SetStorageItemsRequest) (*pb.SetStorageItemsResponse, error) {
	params := map[string]interface{}{
		"id":          req.Id,
		"storageArea": req.StorageArea,
		"values":      req.Values,
	}
	if _, err := s.send(ctx, req.SessionId, "Extensions.setStorageItems", params); err != nil {
		return nil, fmt.Errorf("Extensions.setStorageItems: %w", err)
	}
	return &pb.SetStorageItemsResponse{}, nil
}
