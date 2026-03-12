// Package io implements the gRPC IOService by bridging to CDP.
package io

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/io"
)

type Server struct {
	pb.UnimplementedIOServiceServer
	client *cdpclient.Client
}

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


func (s *Server) Read(ctx context.Context, req *pb.ReadRequest) (*pb.ReadResponse, error) {
	params := map[string]interface{}{"handle": req.Handle}
	if req.Offset != 0 {
		params["offset"] = req.Offset
	}
	if req.Size != 0 {
		params["size"] = req.Size
	}
	result, err := s.send(ctx, req.SessionId, "IO.read", params)
	if err != nil {
		return nil, fmt.Errorf("IO.read: %w", err)
	}
	var resp struct {
		Data          string `json:"data"`
		Base64Encoded bool   `json:"base64Encoded"`
		EOF           bool   `json:"eof"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("IO.read: unmarshal: %w", err)
	}
	return &pb.ReadResponse{
		Data:          resp.Data,
		Base64Encoded: resp.Base64Encoded,
		Eof:           resp.EOF,
	}, nil
}

func (s *Server) Close(ctx context.Context, req *pb.CloseRequest) (*pb.CloseResponse, error) {
	params := map[string]interface{}{"handle": req.Handle}
	if _, err := s.send(ctx, req.SessionId, "IO.close", params); err != nil {
		return nil, fmt.Errorf("IO.close: %w", err)
	}
	return &pb.CloseResponse{}, nil
}

func (s *Server) ResolveBlob(ctx context.Context, req *pb.ResolveBlobRequest) (*pb.ResolveBlobResponse, error) {
	params := map[string]interface{}{"objectId": req.ObjectId}
	result, err := s.send(ctx, req.SessionId, "IO.resolveBlob", params)
	if err != nil {
		return nil, fmt.Errorf("IO.resolveBlob: %w", err)
	}
	var resp struct {
		UUID string `json:"uuid"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("IO.resolveBlob: unmarshal: %w", err)
	}
	return &pb.ResolveBlobResponse{Uuid: resp.UUID}, nil
}
