// Package fedcm implements the gRPC FedCmService by bridging to CDP.
package fedcm

import (
	"context"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/fedcm"
)

type Server struct {
	pb.UnimplementedFedCmServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	params := map[string]interface{}{}
	if req.DisableRejectionDelay != nil {
		params["disableRejectionDelay"] = *req.DisableRejectionDelay
	}
	if _, err := s.client.Send(ctx, "FedCm.enable", params); err != nil {
		return nil, fmt.Errorf("FedCm.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "FedCm.disable", nil); err != nil {
		return nil, fmt.Errorf("FedCm.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) SelectAccount(ctx context.Context, req *pb.SelectAccountRequest) (*pb.SelectAccountResponse, error) {
	params := map[string]interface{}{
		"dialogId":     req.DialogId,
		"accountIndex": req.AccountIndex,
	}
	if _, err := s.client.Send(ctx, "FedCm.selectAccount", params); err != nil {
		return nil, fmt.Errorf("FedCm.selectAccount: %w", err)
	}
	return &pb.SelectAccountResponse{}, nil
}

func (s *Server) ClickDialogButton(ctx context.Context, req *pb.ClickDialogButtonRequest) (*pb.ClickDialogButtonResponse, error) {
	params := map[string]interface{}{
		"dialogId":     req.DialogId,
		"dialogButton": req.DialogButton,
	}
	if _, err := s.client.Send(ctx, "FedCm.clickDialogButton", params); err != nil {
		return nil, fmt.Errorf("FedCm.clickDialogButton: %w", err)
	}
	return &pb.ClickDialogButtonResponse{}, nil
}

func (s *Server) OpenUrl(ctx context.Context, req *pb.OpenUrlRequest) (*pb.OpenUrlResponse, error) {
	params := map[string]interface{}{
		"dialogId":       req.DialogId,
		"accountIndex":   req.AccountIndex,
		"accountUrlType": req.AccountUrlType,
	}
	if _, err := s.client.Send(ctx, "FedCm.openUrl", params); err != nil {
		return nil, fmt.Errorf("FedCm.openUrl: %w", err)
	}
	return &pb.OpenUrlResponse{}, nil
}

func (s *Server) DismissDialog(ctx context.Context, req *pb.DismissDialogRequest) (*pb.DismissDialogResponse, error) {
	params := map[string]interface{}{
		"dialogId": req.DialogId,
	}
	if req.TriggerCooldown != nil {
		params["triggerCooldown"] = *req.TriggerCooldown
	}
	if _, err := s.client.Send(ctx, "FedCm.dismissDialog", params); err != nil {
		return nil, fmt.Errorf("FedCm.dismissDialog: %w", err)
	}
	return &pb.DismissDialogResponse{}, nil
}

func (s *Server) ResetCooldown(ctx context.Context, req *pb.ResetCooldownRequest) (*pb.ResetCooldownResponse, error) {
	if _, err := s.client.Send(ctx, "FedCm.resetCooldown", nil); err != nil {
		return nil, fmt.Errorf("FedCm.resetCooldown: %w", err)
	}
	return &pb.ResetCooldownResponse{}, nil
}
