// Package cast implements the gRPC CastService by bridging to CDP.
package cast

import (
	"context"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/cast"
)

type Server struct {
	pb.UnimplementedCastServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	var params map[string]interface{}
	if req.PresentationUrl != nil {
		params = map[string]interface{}{
			"presentationUrl": *req.PresentationUrl,
		}
	}
	if _, err := s.client.Send(ctx, "Cast.enable", params); err != nil {
		return nil, fmt.Errorf("Cast.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "Cast.disable", nil); err != nil {
		return nil, fmt.Errorf("Cast.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) SetSinkToUse(ctx context.Context, req *pb.SetSinkToUseRequest) (*pb.SetSinkToUseResponse, error) {
	params := map[string]interface{}{
		"sinkName": req.SinkName,
	}
	if _, err := s.client.Send(ctx, "Cast.setSinkToUse", params); err != nil {
		return nil, fmt.Errorf("Cast.setSinkToUse: %w", err)
	}
	return &pb.SetSinkToUseResponse{}, nil
}

func (s *Server) StartDesktopMirroring(ctx context.Context, req *pb.StartDesktopMirroringRequest) (*pb.StartDesktopMirroringResponse, error) {
	params := map[string]interface{}{
		"sinkName": req.SinkName,
	}
	if _, err := s.client.Send(ctx, "Cast.startDesktopMirroring", params); err != nil {
		return nil, fmt.Errorf("Cast.startDesktopMirroring: %w", err)
	}
	return &pb.StartDesktopMirroringResponse{}, nil
}

func (s *Server) StartTabMirroring(ctx context.Context, req *pb.StartTabMirroringRequest) (*pb.StartTabMirroringResponse, error) {
	params := map[string]interface{}{
		"sinkName": req.SinkName,
	}
	if _, err := s.client.Send(ctx, "Cast.startTabMirroring", params); err != nil {
		return nil, fmt.Errorf("Cast.startTabMirroring: %w", err)
	}
	return &pb.StartTabMirroringResponse{}, nil
}

func (s *Server) StopCasting(ctx context.Context, req *pb.StopCastingRequest) (*pb.StopCastingResponse, error) {
	params := map[string]interface{}{
		"sinkName": req.SinkName,
	}
	if _, err := s.client.Send(ctx, "Cast.stopCasting", params); err != nil {
		return nil, fmt.Errorf("Cast.stopCasting: %w", err)
	}
	return &pb.StopCastingResponse{}, nil
}
