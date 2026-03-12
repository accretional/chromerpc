// Package cachestorage implements the gRPC CacheStorageService by bridging to CDP.
package cachestorage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/cachestorage"
)

type Server struct {
	pb.UnimplementedCacheStorageServiceServer
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


func (s *Server) RequestCacheNames(ctx context.Context, req *pb.RequestCacheNamesRequest) (*pb.RequestCacheNamesResponse, error) {
	params := map[string]interface{}{}
	if req.SecurityOrigin != nil {
		params["securityOrigin"] = *req.SecurityOrigin
	}
	if req.StorageKey != nil {
		params["storageKey"] = *req.StorageKey
	}
	if req.StorageBucket != nil {
		params["storageBucket"] = *req.StorageBucket
	}
	result, err := s.send(ctx, req.SessionId, "CacheStorage.requestCacheNames", params)
	if err != nil {
		return nil, fmt.Errorf("CacheStorage.requestCacheNames: %w", err)
	}
	var resp struct {
		Caches []rawCache `json:"caches"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("CacheStorage.requestCacheNames: unmarshal: %w", err)
	}
	caches := make([]*pb.Cache, len(resp.Caches))
	for i, c := range resp.Caches {
		caches[i] = c.toProto()
	}
	return &pb.RequestCacheNamesResponse{Caches: caches}, nil
}

func (s *Server) RequestEntries(ctx context.Context, req *pb.RequestEntriesRequest) (*pb.RequestEntriesResponse, error) {
	params := map[string]interface{}{
		"cacheId": req.CacheId,
	}
	if req.SkipCount != nil {
		params["skipCount"] = *req.SkipCount
	}
	if req.PageSize != nil {
		params["pageSize"] = *req.PageSize
	}
	if req.PathFilter != nil {
		params["pathFilter"] = *req.PathFilter
	}
	result, err := s.send(ctx, req.SessionId, "CacheStorage.requestEntries", params)
	if err != nil {
		return nil, fmt.Errorf("CacheStorage.requestEntries: %w", err)
	}
	var resp struct {
		CacheDataEntries []rawDataEntry `json:"cacheDataEntries"`
		ReturnCount      float64        `json:"returnCount"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("CacheStorage.requestEntries: unmarshal: %w", err)
	}
	entries := make([]*pb.DataEntry, len(resp.CacheDataEntries))
	for i, e := range resp.CacheDataEntries {
		entries[i] = e.toProto()
	}
	return &pb.RequestEntriesResponse{
		CacheDataEntries: entries,
		ReturnCount:      resp.ReturnCount,
	}, nil
}

func (s *Server) RequestCachedResponse(ctx context.Context, req *pb.RequestCachedResponseRequest) (*pb.RequestCachedResponseResponse, error) {
	params := map[string]interface{}{
		"cacheId":    req.CacheId,
		"requestURL": req.RequestUrl,
	}
	if len(req.RequestHeaders) > 0 {
		params["requestHeaders"] = headersToSlice(req.RequestHeaders)
	}
	result, err := s.send(ctx, req.SessionId, "CacheStorage.requestCachedResponse", params)
	if err != nil {
		return nil, fmt.Errorf("CacheStorage.requestCachedResponse: %w", err)
	}
	var resp struct {
		Body struct {
			Body string `json:"body"`
		} `json:"body"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("CacheStorage.requestCachedResponse: unmarshal: %w", err)
	}
	bodyBytes, err := base64.StdEncoding.DecodeString(resp.Body.Body)
	if err != nil {
		return nil, fmt.Errorf("CacheStorage.requestCachedResponse: decode body: %w", err)
	}
	return &pb.RequestCachedResponseResponse{
		Body: &pb.CachedResponse{Body: bodyBytes},
	}, nil
}

func (s *Server) DeleteCache(ctx context.Context, req *pb.DeleteCacheRequest) (*pb.DeleteCacheResponse, error) {
	params := map[string]interface{}{
		"cacheId": req.CacheId,
	}
	if _, err := s.send(ctx, req.SessionId, "CacheStorage.deleteCache", params); err != nil {
		return nil, fmt.Errorf("CacheStorage.deleteCache: %w", err)
	}
	return &pb.DeleteCacheResponse{}, nil
}

func (s *Server) DeleteEntry(ctx context.Context, req *pb.DeleteEntryRequest) (*pb.DeleteEntryResponse, error) {
	params := map[string]interface{}{
		"cacheId": req.CacheId,
		"request": req.Request,
	}
	if _, err := s.send(ctx, req.SessionId, "CacheStorage.deleteEntry", params); err != nil {
		return nil, fmt.Errorf("CacheStorage.deleteEntry: %w", err)
	}
	return &pb.DeleteEntryResponse{}, nil
}

// --- helpers ---

type rawCache struct {
	CacheID        string `json:"cacheId"`
	SecurityOrigin string `json:"securityOrigin"`
	StorageKey     string `json:"storageKey"`
	StorageBucket  string `json:"storageBucket,omitempty"`
	CacheName      string `json:"cacheName"`
}

func (r *rawCache) toProto() *pb.Cache {
	c := &pb.Cache{
		CacheId:        r.CacheID,
		SecurityOrigin: r.SecurityOrigin,
		StorageKey:     r.StorageKey,
		CacheName:      r.CacheName,
	}
	if r.StorageBucket != "" {
		c.StorageBucket = &r.StorageBucket
	}
	return c
}

type rawHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type rawDataEntry struct {
	RequestURL         string      `json:"requestURL"`
	RequestMethod      string      `json:"requestMethod"`
	RequestHeaders     []rawHeader `json:"requestHeaders"`
	ResponseTime       float64     `json:"responseTime"`
	ResponseStatus     int32       `json:"responseStatus"`
	ResponseStatusText string      `json:"responseStatusText"`
	ResponseType       string      `json:"responseType"`
	ResponseHeaders    []rawHeader `json:"responseHeaders"`
}

func (r *rawDataEntry) toProto() *pb.DataEntry {
	return &pb.DataEntry{
		RequestUrl:         r.RequestURL,
		RequestMethod:      r.RequestMethod,
		RequestHeaders:     rawHeadersToProto(r.RequestHeaders),
		ResponseTime:       r.ResponseTime,
		ResponseStatus:     r.ResponseStatus,
		ResponseStatusText: r.ResponseStatusText,
		ResponseType:       r.ResponseType,
		ResponseHeaders:    rawHeadersToProto(r.ResponseHeaders),
	}
}

func rawHeadersToProto(headers []rawHeader) []*pb.Header {
	out := make([]*pb.Header, len(headers))
	for i, h := range headers {
		out[i] = &pb.Header{Name: h.Name, Value: h.Value}
	}
	return out
}

func headersToSlice(headers []*pb.Header) []map[string]interface{} {
	out := make([]map[string]interface{}, len(headers))
	for i, h := range headers {
		out[i] = map[string]interface{}{
			"name":  h.Name,
			"value": h.Value,
		}
	}
	return out
}
