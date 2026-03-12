// Package preload implements the gRPC PreloadService by bridging to CDP.
package preload

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/preload"
)

type Server struct {
	pb.UnimplementedPreloadServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.client.Send(ctx, "Preload.enable", nil); err != nil {
		return nil, fmt.Errorf("Preload.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "Preload.disable", nil); err != nil {
		return nil, fmt.Errorf("Preload.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribePreloadEventsRequest, stream pb.PreloadService_SubscribeEventsServer) error {
	eventCh := make(chan *pb.PreloadEvent, 128)
	ctx := stream.Context()

	// Register a wildcard handler for all Preload.* events.
	unregister := s.client.On("Preload.", func(method string, params json.RawMessage, sessionID string) {
		if req.SessionId != "" && sessionID != req.SessionId {
			return
		}
		evt := convertPreloadEvent(method, params)
		if evt != nil {
			select {
			case eventCh <- evt:
			default:
			}
		}
	})

	// Also register specific events since the On() matching is exact.
	preloadEvents := []string{
		"Preload.ruleSetUpdated", "Preload.ruleSetRemoved",
		"Preload.preloadingAttemptSourcesUpdated",
		"Preload.prefetchStatusUpdated", "Preload.prerenderStatusUpdated",
		"Preload.preloadEnabledStateUpdated",
	}
	unregisters := make([]func(), 0, len(preloadEvents)+1)
	unregisters = append(unregisters, unregister)
	for _, method := range preloadEvents {
		method := method
		unreg := s.client.On(method, func(m string, params json.RawMessage, sessionID string) {
			if req.SessionId != "" && sessionID != req.SessionId {
				return
			}
			evt := convertPreloadEvent(method, params)
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

func convertPreloadEvent(method string, params json.RawMessage) *pb.PreloadEvent {
	switch method {
	case "Preload.ruleSetUpdated":
		var d struct {
			RuleSet struct {
				ID             string `json:"id"`
				LoaderID       string `json:"loaderId"`
				SourceText     string `json:"sourceText"`
				BackendNodeID  string `json:"backendNodeId"`
				URL            string `json:"url"`
				RequestID      string `json:"requestId"`
				ErrorType      string `json:"errorType"`
				ErrorMessage   string `json:"errorMessage"`
			} `json:"ruleSet"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PreloadEvent{Event: &pb.PreloadEvent_RuleSetUpdated{
			RuleSetUpdated: &pb.RuleSetUpdatedEvent{
				RuleSet: &pb.RuleSet{
					Id:            d.RuleSet.ID,
					LoaderId:      d.RuleSet.LoaderID,
					SourceText:    d.RuleSet.SourceText,
					BackendNodeId: d.RuleSet.BackendNodeID,
					Url:           d.RuleSet.URL,
					RequestId:     d.RuleSet.RequestID,
					ErrorType:     d.RuleSet.ErrorType,
					ErrorMessage:  d.RuleSet.ErrorMessage,
				},
			},
		}}

	case "Preload.ruleSetRemoved":
		var d struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PreloadEvent{Event: &pb.PreloadEvent_RuleSetRemoved{
			RuleSetRemoved: &pb.RuleSetRemovedEvent{
				Id: d.ID,
			},
		}}

	case "Preload.preloadingAttemptSourcesUpdated":
		// Complex type - store entire params as JSON bytes.
		return &pb.PreloadEvent{Event: &pb.PreloadEvent_PreloadingAttemptSourcesUpdated{
			PreloadingAttemptSourcesUpdated: &pb.PreloadingAttemptSourcesUpdatedEvent{
				LoadingSourceJson: params,
			},
		}}

	case "Preload.prefetchStatusUpdated":
		var d struct {
			Key            json.RawMessage `json:"key"`
			Status         string          `json:"status"`
			PrefetchStatus string          `json:"prefetchStatus"`
			RequestID      string          `json:"requestId"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PreloadEvent{Event: &pb.PreloadEvent_PrefetchStatusUpdated{
			PrefetchStatusUpdated: &pb.PrefetchStatusUpdatedEvent{
				KeyJson:        string(d.Key),
				Status:         d.Status,
				PrefetchStatus: d.PrefetchStatus,
				RequestId:      d.RequestID,
			},
		}}

	case "Preload.prerenderStatusUpdated":
		var d struct {
			Key              json.RawMessage `json:"key"`
			Status           string          `json:"status"`
			PrerenderStatus  string          `json:"prerenderStatus"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PreloadEvent{Event: &pb.PreloadEvent_PrerenderStatusUpdated{
			PrerenderStatusUpdated: &pb.PrerenderStatusUpdatedEvent{
				KeyJson:         string(d.Key),
				Status:          d.Status,
				PrerenderStatus: d.PrerenderStatus,
			},
		}}

	case "Preload.preloadEnabledStateUpdated":
		var d struct {
			DisabledByPreference                      bool `json:"disabledByPreference"`
			DisabledByDataSaver                       bool `json:"disabledByDataSaver"`
			DisabledByBatterySaver                    bool `json:"disabledByBatterySaver"`
			DisabledByHoldbackPrefetchSpeculationRules  bool `json:"disabledByHoldbackPrefetchSpeculationRules"`
			DisabledByHoldbackPrerenderSpeculationRules bool `json:"disabledByHoldbackPrerenderSpeculationRules"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PreloadEvent{Event: &pb.PreloadEvent_PreloadEnabledStateUpdated{
			PreloadEnabledStateUpdated: &pb.PreloadEnabledStateUpdatedEvent{
				DisabledByPreference:                      d.DisabledByPreference,
				DisabledByDataSaver:                       d.DisabledByDataSaver,
				DisabledByBatterySaver:                    d.DisabledByBatterySaver,
				DisabledByHoldbackPrefetchSpeculationRules:  d.DisabledByHoldbackPrefetchSpeculationRules,
				DisabledByHoldbackPrerenderSpeculationRules: d.DisabledByHoldbackPrerenderSpeculationRules,
			},
		}}
	}
	return nil
}
