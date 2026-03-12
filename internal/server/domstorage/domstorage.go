// Package domstorage implements the gRPC DOMStorageService by bridging to CDP.
package domstorage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/domstorage"
)

type Server struct {
	pb.UnimplementedDOMStorageServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.client.Send(ctx, "DOMStorage.enable", nil); err != nil {
		return nil, fmt.Errorf("DOMStorage.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "DOMStorage.disable", nil); err != nil {
		return nil, fmt.Errorf("DOMStorage.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) GetDOMStorageItems(ctx context.Context, req *pb.GetDOMStorageItemsRequest) (*pb.GetDOMStorageItemsResponse, error) {
	params := map[string]interface{}{}
	if req.StorageId != nil {
		params["storageId"] = storageIdToMap(req.StorageId)
	}
	result, err := s.client.Send(ctx, "DOMStorage.getDOMStorageItems", params)
	if err != nil {
		return nil, fmt.Errorf("DOMStorage.getDOMStorageItems: %w", err)
	}
	var resp struct {
		Entries [][]string `json:"entries"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("DOMStorage.getDOMStorageItems: unmarshal: %w", err)
	}
	items := make([]*pb.Item, len(resp.Entries))
	for i, entry := range resp.Entries {
		item := &pb.Item{}
		if len(entry) > 0 {
			item.Key = entry[0]
		}
		if len(entry) > 1 {
			item.Value = entry[1]
		}
		items[i] = item
	}
	return &pb.GetDOMStorageItemsResponse{Entries: items}, nil
}

func (s *Server) SetDOMStorageItem(ctx context.Context, req *pb.SetDOMStorageItemRequest) (*pb.SetDOMStorageItemResponse, error) {
	params := map[string]interface{}{
		"key":   req.Key,
		"value": req.Value,
	}
	if req.StorageId != nil {
		params["storageId"] = storageIdToMap(req.StorageId)
	}
	if _, err := s.client.Send(ctx, "DOMStorage.setDOMStorageItem", params); err != nil {
		return nil, fmt.Errorf("DOMStorage.setDOMStorageItem: %w", err)
	}
	return &pb.SetDOMStorageItemResponse{}, nil
}

func (s *Server) RemoveDOMStorageItem(ctx context.Context, req *pb.RemoveDOMStorageItemRequest) (*pb.RemoveDOMStorageItemResponse, error) {
	params := map[string]interface{}{
		"key": req.Key,
	}
	if req.StorageId != nil {
		params["storageId"] = storageIdToMap(req.StorageId)
	}
	if _, err := s.client.Send(ctx, "DOMStorage.removeDOMStorageItem", params); err != nil {
		return nil, fmt.Errorf("DOMStorage.removeDOMStorageItem: %w", err)
	}
	return &pb.RemoveDOMStorageItemResponse{}, nil
}

func (s *Server) Clear(ctx context.Context, req *pb.ClearRequest) (*pb.ClearResponse, error) {
	params := map[string]interface{}{}
	if req.StorageId != nil {
		params["storageId"] = storageIdToMap(req.StorageId)
	}
	if _, err := s.client.Send(ctx, "DOMStorage.clear", params); err != nil {
		return nil, fmt.Errorf("DOMStorage.clear: %w", err)
	}
	return &pb.ClearResponse{}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeEventsRequest, stream pb.DOMStorageService_SubscribeEventsServer) error {
	ch := make(chan *pb.DOMStorageEvent, 64)
	defer close(ch)

	events := []string{
		"DOMStorage.domStorageItemAdded",
		"DOMStorage.domStorageItemRemoved",
		"DOMStorage.domStorageItemUpdated",
		"DOMStorage.domStorageItemsCleared",
	}
	var unsubs []func()
	for _, evt := range events {
		evt := evt
		unsub := s.client.On(evt, func(_ string, params json.RawMessage, _ string) {
			converted := convertDOMStorageEvent(evt, params)
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

func convertDOMStorageEvent(method string, params json.RawMessage) *pb.DOMStorageEvent {
	switch method {
	case "DOMStorage.domStorageItemAdded":
		var raw struct {
			StorageID rawStorageID `json:"storageId"`
			Key       string       `json:"key"`
			NewValue  string       `json:"newValue"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		return &pb.DOMStorageEvent{Event: &pb.DOMStorageEvent_DomStorageItemAdded{
			DomStorageItemAdded: &pb.DOMStorageItemAddedEvent{
				StorageId: raw.StorageID.toProto(), Key: raw.Key, NewValue: raw.NewValue,
			},
		}}
	case "DOMStorage.domStorageItemRemoved":
		var raw struct {
			StorageID rawStorageID `json:"storageId"`
			Key       string       `json:"key"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		return &pb.DOMStorageEvent{Event: &pb.DOMStorageEvent_DomStorageItemRemoved{
			DomStorageItemRemoved: &pb.DOMStorageItemRemovedEvent{
				StorageId: raw.StorageID.toProto(), Key: raw.Key,
			},
		}}
	case "DOMStorage.domStorageItemUpdated":
		var raw struct {
			StorageID rawStorageID `json:"storageId"`
			Key       string       `json:"key"`
			OldValue  string       `json:"oldValue"`
			NewValue  string       `json:"newValue"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		return &pb.DOMStorageEvent{Event: &pb.DOMStorageEvent_DomStorageItemUpdated{
			DomStorageItemUpdated: &pb.DOMStorageItemUpdatedEvent{
				StorageId: raw.StorageID.toProto(), Key: raw.Key,
				OldValue: raw.OldValue, NewValue: raw.NewValue,
			},
		}}
	case "DOMStorage.domStorageItemsCleared":
		var raw struct {
			StorageID rawStorageID `json:"storageId"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		return &pb.DOMStorageEvent{Event: &pb.DOMStorageEvent_DomStorageItemsCleared{
			DomStorageItemsCleared: &pb.DOMStorageItemsClearedEvent{
				StorageId: raw.StorageID.toProto(),
			},
		}}
	}
	return nil
}

// --- helpers ---

type rawStorageID struct {
	SecurityOrigin string `json:"securityOrigin"`
	StorageKey     string `json:"storageKey"`
	IsLocalStorage bool   `json:"isLocalStorage"`
}

func (r *rawStorageID) toProto() *pb.StorageId {
	return &pb.StorageId{
		SecurityOrigin: r.SecurityOrigin,
		StorageKey:     r.StorageKey,
		IsLocalStorage: r.IsLocalStorage,
	}
}

func storageIdToMap(sid *pb.StorageId) map[string]interface{} {
	m := map[string]interface{}{
		"isLocalStorage": sid.IsLocalStorage,
	}
	if sid.SecurityOrigin != "" {
		m["securityOrigin"] = sid.SecurityOrigin
	}
	if sid.StorageKey != "" {
		m["storageKey"] = sid.StorageKey
	}
	return m
}
