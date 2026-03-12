// Package schema implements the gRPC SchemaService by bridging to CDP.
package schema

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/schema"
)

type Server struct {
	pb.UnimplementedSchemaServiceServer
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


func (s *Server) GetDomains(ctx context.Context, req *pb.GetDomainsRequest) (*pb.GetDomainsResponse, error) {
	result, err := s.send(ctx, req.SessionId, "Schema.getDomains", nil)
	if err != nil {
		return nil, fmt.Errorf("Schema.getDomains: %w", err)
	}
	var resp struct {
		Domains []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"domains"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Schema.getDomains: unmarshal: %w", err)
	}
	domains := make([]*pb.Domain, len(resp.Domains))
	for i, d := range resp.Domains {
		domains[i] = &pb.Domain{
			Name:    d.Name,
			Version: d.Version,
		}
	}
	return &pb.GetDomainsResponse{Domains: domains}, nil
}
