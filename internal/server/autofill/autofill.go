// Package autofill implements the gRPC AutofillService by bridging to CDP.
package autofill

import (
	"context"
	"encoding/json"
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

// send routes a CDP command through the specified session, falling back
// to the client's default session if sessionID is empty.
func (s *Server) send(ctx context.Context, sessionID string, method string, params interface{}) (json.RawMessage, error) {
	if sessionID != "" {
		return s.client.SendWithSession(ctx, method, params, sessionID)
	}
	return s.client.Send(ctx, method, params)
}


func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Autofill.enable", nil); err != nil {
		return nil, fmt.Errorf("Autofill.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Autofill.disable", nil); err != nil {
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
	if _, err := s.send(ctx, req.SessionId, "Autofill.trigger", params); err != nil {
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
	if _, err := s.send(ctx, req.SessionId, "Autofill.setAddresses", params); err != nil {
		return nil, fmt.Errorf("Autofill.setAddresses: %w", err)
	}
	return &pb.SetAddressesResponse{}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeAutofillEventsRequest, stream pb.AutofillService_SubscribeEventsServer) error {
	eventCh := make(chan *pb.AutofillEvent, 128)
	ctx := stream.Context()

	// Register a wildcard handler for all Autofill.* events.
	unregister := s.client.On("Autofill.", func(method string, params json.RawMessage, sessionID string) {
		if req.SessionId != "" && sessionID != req.SessionId {
			return
		}
		evt := convertAutofillEvent(method, params)
		if evt != nil {
			select {
			case eventCh <- evt:
			default:
			}
		}
	})

	// Also register specific events since the On() matching is exact.
	autofillEvents := []string{
		"Autofill.addressFormFilled",
	}
	unregisters := make([]func(), 0, len(autofillEvents)+1)
	unregisters = append(unregisters, unregister)
	for _, method := range autofillEvents {
		method := method
		unreg := s.client.On(method, func(m string, params json.RawMessage, sessionID string) {
			if req.SessionId != "" && sessionID != req.SessionId {
				return
			}
			evt := convertAutofillEvent(method, params)
			if evt != nil {
				select {
				case eventCh <- evt:
				default:
				}
			}
		})
		unregisters = append(unregisters, unreg)
	}
	defer func() {
		for _, unreg := range unregisters {
			unreg()
		}
	}()

	for {
		select {
		case evt := <-eventCh:
			if err := stream.Send(evt); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		case <-s.client.Done():
			return fmt.Errorf("CDP connection closed")
		}
	}
}

func convertAutofillEvent(method string, params json.RawMessage) *pb.AutofillEvent {
	switch method {
	case "Autofill.addressFormFilled":
		var d struct {
			FilledFields []struct {
				HtmlType        string `json:"htmlType"`
				ID              string `json:"id"`
				Name            string `json:"name"`
				Value           string `json:"value"`
				AutofillType    string `json:"autofillType"`
				FillingStrategy string `json:"fillingStrategy"`
				FrameID         string `json:"frameId"`
				FieldID         int32  `json:"fieldId"`
			} `json:"filledFields"`
			AddressUI struct {
				AddressFields []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"addressFields"`
			} `json:"addressUi"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		filledFields := make([]*pb.FilledField, len(d.FilledFields))
		for i, f := range d.FilledFields {
			filledFields[i] = &pb.FilledField{
				HtmlType:        f.HtmlType,
				Id:              f.ID,
				Name:            f.Name,
				Value:           f.Value,
				AutofillType:    f.AutofillType,
				FillingStrategy: f.FillingStrategy,
				FrameId:         f.FrameID,
				FieldId:         f.FieldID,
			}
		}
		addressFields := make([]*pb.AddressField, len(d.AddressUI.AddressFields))
		for i, af := range d.AddressUI.AddressFields {
			addressFields[i] = &pb.AddressField{
				Name:  af.Name,
				Value: af.Value,
			}
		}
		return &pb.AutofillEvent{Event: &pb.AutofillEvent_AddressFormFilled{
			AddressFormFilled: &pb.AddressFormFilledEvent{
				FilledFields: filledFields,
				AddressUi:    &pb.AddressUI{AddressFields: addressFields},
			},
		}}

	default:
		return nil
	}
}
