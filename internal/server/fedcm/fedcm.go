// Package fedcm implements the gRPC FedCmService by bridging to CDP.
package fedcm

import (
	"context"
	"encoding/json"
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

func (s *Server) SubscribeEvents(req *pb.SubscribeFedCmEventsRequest, stream pb.FedCmService_SubscribeEventsServer) error {
	eventCh := make(chan *pb.FedCmEvent, 128)
	ctx := stream.Context()

	// Register a wildcard handler for all FedCm.* events.
	unregister := s.client.On("FedCm.", func(method string, params json.RawMessage, sessionID string) {
		// Only forward events for the requested session (or all if empty).
		if req.SessionId != "" && sessionID != req.SessionId {
			return
		}
		evt := convertFedCmEvent(method, params)
		if evt != nil {
			select {
			case eventCh <- evt:
			default:
			}
		}
	})

	// Also register specific events since the On() matching is exact.
	fedCmEvents := []string{
		"FedCm.dialogShown", "FedCm.dialogClosed",
	}
	unregisters := make([]func(), 0, len(fedCmEvents)+1)
	unregisters = append(unregisters, unregister)
	for _, method := range fedCmEvents {
		method := method
		unreg := s.client.On(method, func(m string, params json.RawMessage, sessionID string) {
			if req.SessionId != "" && sessionID != req.SessionId {
				return
			}
			evt := convertFedCmEvent(method, params)
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

func convertFedCmEvent(method string, params json.RawMessage) *pb.FedCmEvent {
	switch method {
	case "FedCm.dialogShown":
		var d struct {
			DialogID   string `json:"dialogId"`
			DialogType string `json:"dialogType"`
			Accounts   []struct {
				AccountID        string `json:"accountId"`
				Email            string `json:"email"`
				Name             string `json:"name"`
				GivenName        string `json:"givenName"`
				PictureURL       string `json:"pictureUrl"`
				IdpConfigURL     string `json:"idpConfigUrl"`
				IdpLoginURL      string `json:"idpLoginUrl"`
				LoginState       string `json:"loginState"`
				TermsOfServiceURL string `json:"termsOfServiceUrl"`
				PrivacyPolicyURL string `json:"privacyPolicyUrl"`
			} `json:"accounts"`
			Title    string `json:"title"`
			Subtitle string `json:"subtitle"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		accounts := make([]*pb.Account, len(d.Accounts))
		for i, a := range d.Accounts {
			accounts[i] = &pb.Account{
				AccountId:        a.AccountID,
				Email:            a.Email,
				Name:             a.Name,
				GivenName:        a.GivenName,
				PictureUrl:       a.PictureURL,
				IdpConfigUrl:     a.IdpConfigURL,
				IdpLoginUrl:      a.IdpLoginURL,
				LoginState:       a.LoginState,
				TermsOfServiceUrl: a.TermsOfServiceURL,
				PrivacyPolicyUrl: a.PrivacyPolicyURL,
			}
		}
		return &pb.FedCmEvent{Event: &pb.FedCmEvent_DialogShown{
			DialogShown: &pb.DialogShownEvent{
				DialogId:   d.DialogID,
				DialogType: d.DialogType,
				Accounts:   accounts,
				Title:      d.Title,
				Subtitle:   d.Subtitle,
			},
		}}

	case "FedCm.dialogClosed":
		var d struct {
			DialogID string `json:"dialogId"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.FedCmEvent{Event: &pb.FedCmEvent_DialogClosed{
			DialogClosed: &pb.DialogClosedEvent{
				DialogId: d.DialogID,
			},
		}}
	}
	return nil
}
