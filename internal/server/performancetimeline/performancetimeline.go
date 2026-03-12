// Package performancetimeline implements the gRPC PerformanceTimelineService by bridging to CDP.
package performancetimeline

import (
	"context"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/performancetimeline"
)

type Server struct {
	pb.UnimplementedPerformanceTimelineServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	params := map[string]interface{}{
		"eventTypes": req.EventTypes,
	}
	if _, err := s.client.Send(ctx, "PerformanceTimeline.enable", params); err != nil {
		return nil, fmt.Errorf("PerformanceTimeline.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}
