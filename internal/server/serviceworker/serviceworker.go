// Package serviceworker implements the gRPC ServiceWorkerService by bridging to CDP.
package serviceworker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/serviceworker"
)

type Server struct {
	pb.UnimplementedServiceWorkerServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

// sendBrowser sends at browser level (no session ID).
func (s *Server) sendBrowser(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	return s.client.SendWithSession(ctx, method, params, "")
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.sendBrowser(ctx, "ServiceWorker.enable", nil); err != nil {
		return nil, fmt.Errorf("ServiceWorker.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.sendBrowser(ctx, "ServiceWorker.disable", nil); err != nil {
		return nil, fmt.Errorf("ServiceWorker.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) DeliverPushMessage(ctx context.Context, req *pb.DeliverPushMessageRequest) (*pb.DeliverPushMessageResponse, error) {
	params := map[string]interface{}{
		"origin":         req.Origin,
		"registrationId": req.RegistrationId,
		"data":           req.Data,
	}
	if _, err := s.sendBrowser(ctx, "ServiceWorker.deliverPushMessage", params); err != nil {
		return nil, fmt.Errorf("ServiceWorker.deliverPushMessage: %w", err)
	}
	return &pb.DeliverPushMessageResponse{}, nil
}

func (s *Server) DispatchSyncEvent(ctx context.Context, req *pb.DispatchSyncEventRequest) (*pb.DispatchSyncEventResponse, error) {
	params := map[string]interface{}{
		"origin":         req.Origin,
		"registrationId": req.RegistrationId,
		"tag":            req.Tag,
		"lastChance":     req.LastChance,
	}
	if _, err := s.sendBrowser(ctx, "ServiceWorker.dispatchSyncEvent", params); err != nil {
		return nil, fmt.Errorf("ServiceWorker.dispatchSyncEvent: %w", err)
	}
	return &pb.DispatchSyncEventResponse{}, nil
}

func (s *Server) DispatchPeriodicSyncEvent(ctx context.Context, req *pb.DispatchPeriodicSyncEventRequest) (*pb.DispatchPeriodicSyncEventResponse, error) {
	params := map[string]interface{}{
		"origin":         req.Origin,
		"registrationId": req.RegistrationId,
		"tag":            req.Tag,
	}
	if _, err := s.sendBrowser(ctx, "ServiceWorker.dispatchPeriodicSyncEvent", params); err != nil {
		return nil, fmt.Errorf("ServiceWorker.dispatchPeriodicSyncEvent: %w", err)
	}
	return &pb.DispatchPeriodicSyncEventResponse{}, nil
}

func (s *Server) InspectWorker(ctx context.Context, req *pb.InspectWorkerRequest) (*pb.InspectWorkerResponse, error) {
	params := map[string]interface{}{
		"versionId": req.VersionId,
	}
	if _, err := s.sendBrowser(ctx, "ServiceWorker.inspectWorker", params); err != nil {
		return nil, fmt.Errorf("ServiceWorker.inspectWorker: %w", err)
	}
	return &pb.InspectWorkerResponse{}, nil
}

func (s *Server) SetForceUpdateOnPageLoad(ctx context.Context, req *pb.SetForceUpdateOnPageLoadRequest) (*pb.SetForceUpdateOnPageLoadResponse, error) {
	params := map[string]interface{}{
		"forceUpdateOnPageLoad": req.ForceUpdateOnPageLoad,
	}
	if _, err := s.sendBrowser(ctx, "ServiceWorker.setForceUpdateOnPageLoad", params); err != nil {
		return nil, fmt.Errorf("ServiceWorker.setForceUpdateOnPageLoad: %w", err)
	}
	return &pb.SetForceUpdateOnPageLoadResponse{}, nil
}

func (s *Server) SkipWaiting(ctx context.Context, req *pb.SkipWaitingRequest) (*pb.SkipWaitingResponse, error) {
	params := map[string]interface{}{
		"scopeURL": req.ScopeUrl,
	}
	if _, err := s.sendBrowser(ctx, "ServiceWorker.skipWaiting", params); err != nil {
		return nil, fmt.Errorf("ServiceWorker.skipWaiting: %w", err)
	}
	return &pb.SkipWaitingResponse{}, nil
}

func (s *Server) StartWorker(ctx context.Context, req *pb.StartWorkerRequest) (*pb.StartWorkerResponse, error) {
	params := map[string]interface{}{
		"scopeURL": req.ScopeUrl,
	}
	if _, err := s.sendBrowser(ctx, "ServiceWorker.startWorker", params); err != nil {
		return nil, fmt.Errorf("ServiceWorker.startWorker: %w", err)
	}
	return &pb.StartWorkerResponse{}, nil
}

func (s *Server) StopAllWorkers(ctx context.Context, req *pb.StopAllWorkersRequest) (*pb.StopAllWorkersResponse, error) {
	if _, err := s.sendBrowser(ctx, "ServiceWorker.stopAllWorkers", nil); err != nil {
		return nil, fmt.Errorf("ServiceWorker.stopAllWorkers: %w", err)
	}
	return &pb.StopAllWorkersResponse{}, nil
}

func (s *Server) StopWorker(ctx context.Context, req *pb.StopWorkerRequest) (*pb.StopWorkerResponse, error) {
	params := map[string]interface{}{
		"versionId": req.VersionId,
	}
	if _, err := s.sendBrowser(ctx, "ServiceWorker.stopWorker", params); err != nil {
		return nil, fmt.Errorf("ServiceWorker.stopWorker: %w", err)
	}
	return &pb.StopWorkerResponse{}, nil
}

func (s *Server) Unregister(ctx context.Context, req *pb.UnregisterRequest) (*pb.UnregisterResponse, error) {
	params := map[string]interface{}{
		"scopeURL": req.ScopeUrl,
	}
	if _, err := s.sendBrowser(ctx, "ServiceWorker.unregister", params); err != nil {
		return nil, fmt.Errorf("ServiceWorker.unregister: %w", err)
	}
	return &pb.UnregisterResponse{}, nil
}

func (s *Server) UpdateRegistration(ctx context.Context, req *pb.UpdateRegistrationRequest) (*pb.UpdateRegistrationResponse, error) {
	params := map[string]interface{}{
		"scopeURL": req.ScopeUrl,
	}
	if _, err := s.sendBrowser(ctx, "ServiceWorker.updateRegistration", params); err != nil {
		return nil, fmt.Errorf("ServiceWorker.updateRegistration: %w", err)
	}
	return &pb.UpdateRegistrationResponse{}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeEventsRequest, stream pb.ServiceWorkerService_SubscribeEventsServer) error {
	ch := make(chan *pb.ServiceWorkerEvent, 64)
	defer close(ch)

	events := []string{
		"ServiceWorker.workerErrorReported",
		"ServiceWorker.workerRegistrationUpdated",
		"ServiceWorker.workerVersionUpdated",
	}
	var unsubs []func()
	for _, evt := range events {
		evt := evt
		unsub := s.client.On(evt, func(_ string, params json.RawMessage, _ string) {
			converted := convertServiceWorkerEvent(evt, params)
			if converted != nil {
				ch <- converted
			}
		})
		unsubs = append(unsubs, unsub)
	}
	defer func() {
		for _, u := range unsubs {
			u()
		}
	}()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case evt := <-ch:
			if err := stream.Send(evt); err != nil {
				return err
			}
		}
	}
}

func convertServiceWorkerEvent(method string, params json.RawMessage) *pb.ServiceWorkerEvent {
	switch method {
	case "ServiceWorker.workerErrorReported":
		var raw struct {
			ErrorMessage struct {
				ErrorMessage   string `json:"errorMessage"`
				RegistrationID string `json:"registrationId"`
				VersionID      string `json:"versionId"`
				SourceURL      string `json:"sourceURL"`
				LineNumber     int32  `json:"lineNumber"`
				ColumnNumber   int32  `json:"columnNumber"`
			} `json:"errorMessage"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		return &pb.ServiceWorkerEvent{Event: &pb.ServiceWorkerEvent_WorkerErrorReported{
			WorkerErrorReported: &pb.WorkerErrorReportedEvent{
				ErrorMessage: &pb.ServiceWorkerErrorMessage{
					ErrorMessage:   raw.ErrorMessage.ErrorMessage,
					RegistrationId: raw.ErrorMessage.RegistrationID,
					VersionId:      raw.ErrorMessage.VersionID,
					SourceUrl:      raw.ErrorMessage.SourceURL,
					LineNumber:     raw.ErrorMessage.LineNumber,
					ColumnNumber:   raw.ErrorMessage.ColumnNumber,
				},
			},
		}}
	case "ServiceWorker.workerRegistrationUpdated":
		var raw struct {
			Registrations []struct {
				RegistrationID string `json:"registrationId"`
				ScopeURL       string `json:"scopeURL"`
				IsDeleted      bool   `json:"isDeleted"`
			} `json:"registrations"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		regs := make([]*pb.ServiceWorkerRegistration, len(raw.Registrations))
		for i, r := range raw.Registrations {
			regs[i] = &pb.ServiceWorkerRegistration{
				RegistrationId: r.RegistrationID,
				ScopeUrl:       r.ScopeURL,
				IsDeleted:      r.IsDeleted,
			}
		}
		return &pb.ServiceWorkerEvent{Event: &pb.ServiceWorkerEvent_WorkerRegistrationUpdated{
			WorkerRegistrationUpdated: &pb.WorkerRegistrationUpdatedEvent{
				Registrations: regs,
			},
		}}
	case "ServiceWorker.workerVersionUpdated":
		var raw struct {
			Versions []struct {
				VersionID          string   `json:"versionId"`
				RegistrationID     string   `json:"registrationId"`
				ScriptURL          string   `json:"scriptURL"`
				RunningStatus      string   `json:"runningStatus"`
				Status             string   `json:"status"`
				ScriptLastModified *float64 `json:"scriptLastModified"`
				ScriptResponseTime *float64 `json:"scriptResponseTime"`
				ControlledClients  []string `json:"controlledClients"`
				TargetID           *string  `json:"targetId"`
				RouterRules        *string  `json:"routerRules"`
			} `json:"versions"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		versions := make([]*pb.ServiceWorkerVersion, len(raw.Versions))
		for i, v := range raw.Versions {
			ver := &pb.ServiceWorkerVersion{
				VersionId:         v.VersionID,
				RegistrationId:    v.RegistrationID,
				ScriptUrl:         v.ScriptURL,
				RunningStatus:     v.RunningStatus,
				Status:            v.Status,
				ControlledClients: v.ControlledClients,
			}
			if v.ScriptLastModified != nil {
				ver.ScriptLastModified = v.ScriptLastModified
			}
			if v.ScriptResponseTime != nil {
				ver.ScriptResponseTime = v.ScriptResponseTime
			}
			if v.TargetID != nil {
				ver.TargetId = v.TargetID
			}
			if v.RouterRules != nil {
				ver.RouterRules = v.RouterRules
			}
			versions[i] = ver
		}
		return &pb.ServiceWorkerEvent{Event: &pb.ServiceWorkerEvent_WorkerVersionUpdated{
			WorkerVersionUpdated: &pb.WorkerVersionUpdatedEvent{
				Versions: versions,
			},
		}}
	}
	return nil
}
