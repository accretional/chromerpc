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

func (s *Server) GetDomains(ctx context.Context, req *pb.GetDomainsRequest) (*pb.GetDomainsResponse, error) {
	result, err := s.client.Send(ctx, "Schema.getDomains", nil)
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
