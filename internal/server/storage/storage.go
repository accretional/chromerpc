// Package storage implements the gRPC StorageService by bridging to CDP.
package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/storage"
)

type Server struct {
	pb.UnimplementedStorageServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

// sendBrowser sends at browser level (no session ID).
func (s *Server) sendBrowser(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	return s.client.SendWithSession(ctx, method, params, "")
}

func (s *Server) ClearDataForOrigin(ctx context.Context, req *pb.ClearDataForOriginRequest) (*pb.ClearDataForOriginResponse, error) {
	params := map[string]interface{}{
		"origin":       req.Origin,
		"storageTypes": req.StorageTypes,
	}
	if _, err := s.sendBrowser(ctx, "Storage.clearDataForOrigin", params); err != nil {
		return nil, fmt.Errorf("Storage.clearDataForOrigin: %w", err)
	}
	return &pb.ClearDataForOriginResponse{}, nil
}

func (s *Server) ClearDataForStorageKey(ctx context.Context, req *pb.ClearDataForStorageKeyRequest) (*pb.ClearDataForStorageKeyResponse, error) {
	params := map[string]interface{}{
		"storageKey":   req.StorageKey,
		"storageTypes": req.StorageTypes,
	}
	if _, err := s.sendBrowser(ctx, "Storage.clearDataForStorageKey", params); err != nil {
		return nil, fmt.Errorf("Storage.clearDataForStorageKey: %w", err)
	}
	return &pb.ClearDataForStorageKeyResponse{}, nil
}

func (s *Server) GetUsageAndQuota(ctx context.Context, req *pb.GetUsageAndQuotaRequest) (*pb.GetUsageAndQuotaResponse, error) {
	params := map[string]interface{}{"origin": req.Origin}
	result, err := s.sendBrowser(ctx, "Storage.getUsageAndQuota", params)
	if err != nil {
		return nil, fmt.Errorf("Storage.getUsageAndQuota: %w", err)
	}
	var resp struct {
		Usage          float64 `json:"usage"`
		Quota          float64 `json:"quota"`
		OverrideActive bool    `json:"overrideActive"`
		UsageBreakdown []struct {
			StorageType string  `json:"storageType"`
			Usage       float64 `json:"usage"`
		} `json:"usageBreakdown"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Storage.getUsageAndQuota: unmarshal: %w", err)
	}
	breakdown := make([]*pb.UsageForType, len(resp.UsageBreakdown))
	for i, u := range resp.UsageBreakdown {
		breakdown[i] = &pb.UsageForType{StorageType: u.StorageType, Usage: u.Usage}
	}
	return &pb.GetUsageAndQuotaResponse{
		Usage:          resp.Usage,
		Quota:          resp.Quota,
		OverrideActive: resp.OverrideActive,
		UsageBreakdown: breakdown,
	}, nil
}

func (s *Server) GetCookies(ctx context.Context, req *pb.GetCookiesRequest) (*pb.GetCookiesResponse, error) {
	params := map[string]interface{}{}
	if req.BrowserContextId != "" {
		params["browserContextId"] = req.BrowserContextId
	}
	var result json.RawMessage
	var err error
	if len(params) > 0 {
		result, err = s.sendBrowser(ctx, "Storage.getCookies", params)
	} else {
		result, err = s.sendBrowser(ctx, "Storage.getCookies", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Storage.getCookies: %w", err)
	}
	var resp struct {
		Cookies []cdpStorageCookie `json:"cookies"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Storage.getCookies: unmarshal: %w", err)
	}
	cookies := make([]*pb.StorageCookie, len(resp.Cookies))
	for i, c := range resp.Cookies {
		cookies[i] = c.toProto()
	}
	return &pb.GetCookiesResponse{Cookies: cookies}, nil
}

func (s *Server) SetCookies(ctx context.Context, req *pb.SetCookiesRequest) (*pb.SetCookiesResponse, error) {
	cookies := make([]map[string]interface{}, len(req.Cookies))
	for i, c := range req.Cookies {
		cookie := map[string]interface{}{"name": c.Name, "value": c.Value}
		if c.Url != "" {
			cookie["url"] = c.Url
		}
		if c.Domain != "" {
			cookie["domain"] = c.Domain
		}
		if c.Path != "" {
			cookie["path"] = c.Path
		}
		if c.Secure {
			cookie["secure"] = true
		}
		if c.HttpOnly {
			cookie["httpOnly"] = true
		}
		if c.SameSite != "" {
			cookie["sameSite"] = c.SameSite
		}
		if c.Expires != 0 {
			cookie["expires"] = c.Expires
		}
		if c.Priority != "" {
			cookie["priority"] = c.Priority
		}
		if c.SameParty {
			cookie["sameParty"] = true
		}
		if c.SourceScheme != "" {
			cookie["sourceScheme"] = c.SourceScheme
		}
		if c.SourcePort != 0 {
			cookie["sourcePort"] = c.SourcePort
		}
		cookies[i] = cookie
	}
	params := map[string]interface{}{"cookies": cookies}
	if req.BrowserContextId != "" {
		params["browserContextId"] = req.BrowserContextId
	}
	if _, err := s.sendBrowser(ctx, "Storage.setCookies", params); err != nil {
		return nil, fmt.Errorf("Storage.setCookies: %w", err)
	}
	return &pb.SetCookiesResponse{}, nil
}

func (s *Server) ClearCookies(ctx context.Context, req *pb.ClearCookiesRequest) (*pb.ClearCookiesResponse, error) {
	params := map[string]interface{}{}
	if req.BrowserContextId != "" {
		params["browserContextId"] = req.BrowserContextId
	}
	if len(params) > 0 {
		_, err := s.sendBrowser(ctx, "Storage.clearCookies", params)
		if err != nil {
			return nil, fmt.Errorf("Storage.clearCookies: %w", err)
		}
	} else {
		if _, err := s.sendBrowser(ctx, "Storage.clearCookies", nil); err != nil {
			return nil, fmt.Errorf("Storage.clearCookies: %w", err)
		}
	}
	return &pb.ClearCookiesResponse{}, nil
}

func (s *Server) TrackIndexedDBForOrigin(ctx context.Context, req *pb.TrackIndexedDBForOriginRequest) (*pb.TrackIndexedDBForOriginResponse, error) {
	params := map[string]interface{}{"origin": req.Origin}
	if _, err := s.sendBrowser(ctx, "Storage.trackIndexedDBForOrigin", params); err != nil {
		return nil, fmt.Errorf("Storage.trackIndexedDBForOrigin: %w", err)
	}
	return &pb.TrackIndexedDBForOriginResponse{}, nil
}

func (s *Server) UntrackIndexedDBForOrigin(ctx context.Context, req *pb.UntrackIndexedDBForOriginRequest) (*pb.UntrackIndexedDBForOriginResponse, error) {
	params := map[string]interface{}{"origin": req.Origin}
	if _, err := s.sendBrowser(ctx, "Storage.untrackIndexedDBForOrigin", params); err != nil {
		return nil, fmt.Errorf("Storage.untrackIndexedDBForOrigin: %w", err)
	}
	return &pb.UntrackIndexedDBForOriginResponse{}, nil
}

func (s *Server) TrackCacheStorageForOrigin(ctx context.Context, req *pb.TrackCacheStorageForOriginRequest) (*pb.TrackCacheStorageForOriginResponse, error) {
	params := map[string]interface{}{"origin": req.Origin}
	if _, err := s.sendBrowser(ctx, "Storage.trackCacheStorageForOrigin", params); err != nil {
		return nil, fmt.Errorf("Storage.trackCacheStorageForOrigin: %w", err)
	}
	return &pb.TrackCacheStorageForOriginResponse{}, nil
}

func (s *Server) UntrackCacheStorageForOrigin(ctx context.Context, req *pb.UntrackCacheStorageForOriginRequest) (*pb.UntrackCacheStorageForOriginResponse, error) {
	params := map[string]interface{}{"origin": req.Origin}
	if _, err := s.sendBrowser(ctx, "Storage.untrackCacheStorageForOrigin", params); err != nil {
		return nil, fmt.Errorf("Storage.untrackCacheStorageForOrigin: %w", err)
	}
	return &pb.UntrackCacheStorageForOriginResponse{}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeEventsRequest, stream pb.StorageService_SubscribeEventsServer) error {
	ch := make(chan *pb.StorageEvent, 64)
	defer close(ch)

	events := []string{
		"Storage.cacheStorageContentUpdated",
		"Storage.cacheStorageListUpdated",
		"Storage.indexedDBContentUpdated",
		"Storage.indexedDBListUpdated",
	}
	var unsubs []func()
	for _, evt := range events {
		evt := evt
		unsub := s.client.On(evt, func(_ string, params json.RawMessage, _ string) {
			converted := convertStorageEvent(evt, params)
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

func convertStorageEvent(method string, params json.RawMessage) *pb.StorageEvent {
	switch method {
	case "Storage.cacheStorageContentUpdated":
		var raw struct {
			Origin     string `json:"origin"`
			StorageKey string `json:"storageKey"`
			BucketID   string `json:"bucketId"`
			CacheName  string `json:"cacheName"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		return &pb.StorageEvent{Event: &pb.StorageEvent_CacheStorageContentUpdated{
			CacheStorageContentUpdated: &pb.CacheStorageContentUpdatedEvent{
				Origin: raw.Origin, StorageKey: raw.StorageKey, BucketId: raw.BucketID, CacheName: raw.CacheName,
			},
		}}
	case "Storage.cacheStorageListUpdated":
		var raw struct {
			Origin     string `json:"origin"`
			StorageKey string `json:"storageKey"`
			BucketID   string `json:"bucketId"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		return &pb.StorageEvent{Event: &pb.StorageEvent_CacheStorageListUpdated{
			CacheStorageListUpdated: &pb.CacheStorageListUpdatedEvent{
				Origin: raw.Origin, StorageKey: raw.StorageKey, BucketId: raw.BucketID,
			},
		}}
	case "Storage.indexedDBContentUpdated":
		var raw struct {
			Origin          string `json:"origin"`
			StorageKey      string `json:"storageKey"`
			BucketID        string `json:"bucketId"`
			DatabaseName    string `json:"databaseName"`
			ObjectStoreName string `json:"objectStoreName"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		return &pb.StorageEvent{Event: &pb.StorageEvent_IndexedDbContentUpdated{
			IndexedDbContentUpdated: &pb.IndexedDBContentUpdatedEvent{
				Origin: raw.Origin, StorageKey: raw.StorageKey, BucketId: raw.BucketID,
				DatabaseName: raw.DatabaseName, ObjectStoreName: raw.ObjectStoreName,
			},
		}}
	case "Storage.indexedDBListUpdated":
		var raw struct {
			Origin     string `json:"origin"`
			StorageKey string `json:"storageKey"`
			BucketID   string `json:"bucketId"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		return &pb.StorageEvent{Event: &pb.StorageEvent_IndexedDbListUpdated{
			IndexedDbListUpdated: &pb.IndexedDBListUpdatedEvent{
				Origin: raw.Origin, StorageKey: raw.StorageKey, BucketId: raw.BucketID,
			},
		}}
	}
	return nil
}

// --- internal helpers ---

type cdpStorageCookie struct {
	Name         string  `json:"name"`
	Value        string  `json:"value"`
	Domain       string  `json:"domain"`
	Path         string  `json:"path"`
	Expires      float64 `json:"expires"`
	Size         int32   `json:"size"`
	HTTPOnly     bool    `json:"httpOnly"`
	Secure       bool    `json:"secure"`
	SameSite     string  `json:"sameSite"`
	Priority     string  `json:"priority"`
	SameParty    bool    `json:"sameParty"`
	SourceScheme string  `json:"sourceScheme"`
	SourcePort   int32   `json:"sourcePort"`
	PartitionKey string  `json:"partitionKey"`
}

func (c *cdpStorageCookie) toProto() *pb.StorageCookie {
	return &pb.StorageCookie{
		Name:         c.Name,
		Value:        c.Value,
		Domain:       c.Domain,
		Path:         c.Path,
		Expires:      c.Expires,
		Size:         c.Size,
		HttpOnly:     c.HTTPOnly,
		Secure:       c.Secure,
		SameSite:     c.SameSite,
		Priority:     c.Priority,
		SameParty:    c.SameParty,
		SourceScheme: c.SourceScheme,
		SourcePort:   c.SourcePort,
		PartitionKey: c.PartitionKey,
	}
}
