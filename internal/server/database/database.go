// Package database implements the gRPC DatabaseService by bridging to CDP.
package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/database"
)

type Server struct {
	pb.UnimplementedDatabaseServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.client.Send(ctx, "Database.enable", nil); err != nil {
		return nil, fmt.Errorf("Database.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "Database.disable", nil); err != nil {
		return nil, fmt.Errorf("Database.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) GetDatabaseTableNames(ctx context.Context, req *pb.GetDatabaseTableNamesRequest) (*pb.GetDatabaseTableNamesResponse, error) {
	params := map[string]interface{}{
		"databaseId": req.DatabaseId,
	}
	result, err := s.client.Send(ctx, "Database.getDatabaseTableNames", params)
	if err != nil {
		return nil, fmt.Errorf("Database.getDatabaseTableNames: %w", err)
	}
	var resp struct {
		TableNames []string `json:"tableNames"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Database.getDatabaseTableNames: unmarshal: %w", err)
	}
	return &pb.GetDatabaseTableNamesResponse{TableNames: resp.TableNames}, nil
}

func (s *Server) ExecuteSQL(ctx context.Context, req *pb.ExecuteSQLRequest) (*pb.ExecuteSQLResponse, error) {
	params := map[string]interface{}{
		"databaseId": req.DatabaseId,
		"query":      req.Query,
	}
	result, err := s.client.Send(ctx, "Database.executeSQL", params)
	if err != nil {
		return nil, fmt.Errorf("Database.executeSQL: %w", err)
	}
	var resp struct {
		ColumnNames []string          `json:"columnNames"`
		Values      []json.RawMessage `json:"values"`
		SqlError    *struct {
			Message string `json:"message"`
			Code    int32  `json:"code"`
		} `json:"sqlError"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Database.executeSQL: unmarshal: %w", err)
	}
	out := &pb.ExecuteSQLResponse{
		ColumnNames: resp.ColumnNames,
	}
	for _, v := range resp.Values {
		out.Values = append(out.Values, string(v))
	}
	if resp.SqlError != nil {
		out.SqlError = &pb.SqlError{
			Message: resp.SqlError.Message,
			Code:    resp.SqlError.Code,
		}
	}
	return out, nil
}
