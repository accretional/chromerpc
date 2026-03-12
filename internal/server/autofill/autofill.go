// Package autofill implements the gRPC AutofillService by bridging to CDP.
package autofill

import (
	"context"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/autofill"
)

type Server struct {
	pb.UnimplementedAutofillServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.client.Send(ctx, "Autofill.enable", nil); err != nil {
		return nil, fmt.Errorf("Autofill.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "Autofill.disable", nil); err != nil {
		return nil, fmt.Errorf("Autofill.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) Trigger(ctx context.Context, req *pb.TriggerRequest) (*pb.TriggerResponse, error) {
	params := map[string]interface{}{
		"fieldId": req.FieldId,
	}
	if req.FrameId != "" {
		params["frameId"] = req.FrameId
	}
	if req.Card != nil {
		params["card"] = map[string]interface{}{
			"number":      req.Card.Number,
			"name":        req.Card.Name,
			"expiryMonth": req.Card.ExpiryMonth,
			"expiryYear":  req.Card.ExpiryYear,
			"cvc":         req.Card.Cvc,
		}
	}
	if _, err := s.client.Send(ctx, "Autofill.trigger", params); err != nil {
		return nil, fmt.Errorf("Autofill.trigger: %w", err)
	}
	return &pb.TriggerResponse{}, nil
}

func (s *Server) SetAddresses(ctx context.Context, req *pb.SetAddressesRequest) (*pb.SetAddressesResponse, error) {
	addresses := make([]map[string]interface{}, len(req.Addresses))
	for i, addr := range req.Addresses {
		fields := make([]map[string]interface{}, len(addr.Fields))
		for j, f := range addr.Fields {
			fields[j] = map[string]interface{}{
				"name":  f.Name,
				"value": f.Value,
			}
		}
		addresses[i] = map[string]interface{}{
			"fields": fields,
		}
	}
	params := map[string]interface{}{
		"addresses": addresses,
	}
	if _, err := s.client.Send(ctx, "Autofill.setAddresses", params); err != nil {
		return nil, fmt.Errorf("Autofill.setAddresses: %w", err)
	}
	return &pb.SetAddressesResponse{}, nil
}
