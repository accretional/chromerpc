// Package indexeddb implements the gRPC IndexedDBService by bridging to CDP.
package indexeddb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/indexeddb"
)

type Server struct {
	pb.UnimplementedIndexedDBServiceServer
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
	if _, err := s.send(ctx, req.SessionId, "IndexedDB.enable", nil); err != nil {
		return nil, fmt.Errorf("IndexedDB.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "IndexedDB.disable", nil); err != nil {
		return nil, fmt.Errorf("IndexedDB.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) RequestDatabaseNames(ctx context.Context, req *pb.RequestDatabaseNamesRequest) (*pb.RequestDatabaseNamesResponse, error) {
	params := map[string]interface{}{}
	addOriginParams(params, req.SecurityOrigin, req.StorageKey, req.StorageBucket)
	result, err := s.send(ctx, req.SessionId, "IndexedDB.requestDatabaseNames", params)
	if err != nil {
		return nil, fmt.Errorf("IndexedDB.requestDatabaseNames: %w", err)
	}
	var resp struct {
		DatabaseNames []string `json:"databaseNames"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("IndexedDB.requestDatabaseNames: unmarshal: %w", err)
	}
	return &pb.RequestDatabaseNamesResponse{DatabaseNames: resp.DatabaseNames}, nil
}

func (s *Server) RequestDatabase(ctx context.Context, req *pb.RequestDatabaseRequest) (*pb.RequestDatabaseResponse, error) {
	params := map[string]interface{}{
		"databaseName": req.DatabaseName,
	}
	addOriginParams(params, req.SecurityOrigin, req.StorageKey, req.StorageBucket)
	result, err := s.send(ctx, req.SessionId, "IndexedDB.requestDatabase", params)
	if err != nil {
		return nil, fmt.Errorf("IndexedDB.requestDatabase: %w", err)
	}
	var resp struct {
		DatabaseWithObjectStores rawDatabase `json:"databaseWithObjectStores"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("IndexedDB.requestDatabase: unmarshal: %w", err)
	}
	return &pb.RequestDatabaseResponse{
		DatabaseWithObjectStores: resp.DatabaseWithObjectStores.toProto(),
	}, nil
}

func (s *Server) RequestData(ctx context.Context, req *pb.RequestDataRequest) (*pb.RequestDataResponse, error) {
	params := map[string]interface{}{
		"databaseName":    req.DatabaseName,
		"objectStoreName": req.ObjectStoreName,
		"indexName":       req.IndexName,
		"skipCount":       req.SkipCount,
		"pageSize":        req.PageSize,
	}
	addOriginParams(params, req.SecurityOrigin, req.StorageKey, req.StorageBucket)
	if req.KeyRange != nil {
		params["keyRange"] = keyRangeToMap(req.KeyRange)
	}
	result, err := s.send(ctx, req.SessionId, "IndexedDB.requestData", params)
	if err != nil {
		return nil, fmt.Errorf("IndexedDB.requestData: %w", err)
	}
	var resp struct {
		ObjectStoreDataEntries []rawDataEntry `json:"objectStoreDataEntries"`
		HasMore                bool           `json:"hasMore"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("IndexedDB.requestData: unmarshal: %w", err)
	}
	entries := make([]*pb.DataEntry, len(resp.ObjectStoreDataEntries))
	for i, e := range resp.ObjectStoreDataEntries {
		entries[i] = &pb.DataEntry{
			Key:        string(e.Key),
			PrimaryKey: string(e.PrimaryKey),
			Value:      string(e.Value),
		}
	}
	return &pb.RequestDataResponse{
		ObjectStoreDataEntries: entries,
		HasMore:                resp.HasMore,
	}, nil
}

func (s *Server) GetMetadata(ctx context.Context, req *pb.GetMetadataRequest) (*pb.GetMetadataResponse, error) {
	params := map[string]interface{}{
		"databaseName":    req.DatabaseName,
		"objectStoreName": req.ObjectStoreName,
	}
	addOriginParams(params, req.SecurityOrigin, req.StorageKey, req.StorageBucket)
	result, err := s.send(ctx, req.SessionId, "IndexedDB.getMetadata", params)
	if err != nil {
		return nil, fmt.Errorf("IndexedDB.getMetadata: %w", err)
	}
	var resp struct {
		EntriesCount      float64 `json:"entriesCount"`
		KeyGeneratorValue float64 `json:"keyGeneratorValue"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("IndexedDB.getMetadata: unmarshal: %w", err)
	}
	return &pb.GetMetadataResponse{
		EntriesCount:      resp.EntriesCount,
		KeyGeneratorValue: resp.KeyGeneratorValue,
	}, nil
}

func (s *Server) DeleteDatabase(ctx context.Context, req *pb.DeleteDatabaseRequest) (*pb.DeleteDatabaseResponse, error) {
	params := map[string]interface{}{
		"databaseName": req.DatabaseName,
	}
	addOriginParams(params, req.SecurityOrigin, req.StorageKey, req.StorageBucket)
	if _, err := s.send(ctx, req.SessionId, "IndexedDB.deleteDatabase", params); err != nil {
		return nil, fmt.Errorf("IndexedDB.deleteDatabase: %w", err)
	}
	return &pb.DeleteDatabaseResponse{}, nil
}

func (s *Server) ClearObjectStore(ctx context.Context, req *pb.ClearObjectStoreRequest) (*pb.ClearObjectStoreResponse, error) {
	params := map[string]interface{}{
		"databaseName":    req.DatabaseName,
		"objectStoreName": req.ObjectStoreName,
	}
	addOriginParams(params, req.SecurityOrigin, req.StorageKey, req.StorageBucket)
	if _, err := s.send(ctx, req.SessionId, "IndexedDB.clearObjectStore", params); err != nil {
		return nil, fmt.Errorf("IndexedDB.clearObjectStore: %w", err)
	}
	return &pb.ClearObjectStoreResponse{}, nil
}

func (s *Server) DeleteObjectStoreEntries(ctx context.Context, req *pb.DeleteObjectStoreEntriesRequest) (*pb.DeleteObjectStoreEntriesResponse, error) {
	params := map[string]interface{}{
		"databaseName":    req.DatabaseName,
		"objectStoreName": req.ObjectStoreName,
	}
	addOriginParams(params, req.SecurityOrigin, req.StorageKey, req.StorageBucket)
	if req.KeyRange != nil {
		params["keyRange"] = keyRangeToMap(req.KeyRange)
	}
	if _, err := s.send(ctx, req.SessionId, "IndexedDB.deleteObjectStoreEntries", params); err != nil {
		return nil, fmt.Errorf("IndexedDB.deleteObjectStoreEntries: %w", err)
	}
	return &pb.DeleteObjectStoreEntriesResponse{}, nil
}

// --- helpers ---

func addOriginParams(params map[string]interface{}, securityOrigin, storageKey, storageBucket string) {
	if securityOrigin != "" {
		params["securityOrigin"] = securityOrigin
	}
	if storageKey != "" {
		params["storageKey"] = storageKey
	}
	if storageBucket != "" {
		params["storageBucket"] = storageBucket
	}
}

func keyRangeToMap(kr *pb.KeyRange) map[string]interface{} {
	m := map[string]interface{}{
		"lowerOpen": kr.LowerOpen,
		"upperOpen": kr.UpperOpen,
	}
	if kr.Lower != "" {
		m["lower"] = json.RawMessage(kr.Lower)
	}
	if kr.Upper != "" {
		m["upper"] = json.RawMessage(kr.Upper)
	}
	return m
}

type rawKeyPath struct {
	Type        string   `json:"type"`
	String      string   `json:"string"`
	Array       []string `json:"array"`
}

func (r *rawKeyPath) toProto() *pb.KeyPath {
	if r == nil {
		return nil
	}
	return &pb.KeyPath{
		Type:        r.Type,
		StringValue: r.String,
		ArrayValues: r.Array,
	}
}

type rawObjectStoreIndex struct {
	Name       string      `json:"name"`
	KeyPath    *rawKeyPath `json:"keyPath"`
	Unique     bool        `json:"unique"`
	MultiEntry bool        `json:"multiEntry"`
}

type rawObjectStore struct {
	Name          string                `json:"name"`
	KeyPath       *rawKeyPath           `json:"keyPath"`
	AutoIncrement bool                  `json:"autoIncrement"`
	Indexes       []rawObjectStoreIndex `json:"indexes"`
}

type rawDatabase struct {
	Name         string           `json:"name"`
	Version      float64          `json:"version"`
	ObjectStores []rawObjectStore `json:"objectStores"`
}

func (r *rawDatabase) toProto() *pb.DatabaseWithObjectStores {
	if r == nil {
		return nil
	}
	stores := make([]*pb.ObjectStore, len(r.ObjectStores))
	for i, os := range r.ObjectStores {
		indexes := make([]*pb.ObjectStoreIndex, len(os.Indexes))
		for j, idx := range os.Indexes {
			indexes[j] = &pb.ObjectStoreIndex{
				Name:       idx.Name,
				KeyPath:    idx.KeyPath.toProto(),
				Unique:     idx.Unique,
				MultiEntry: idx.MultiEntry,
			}
		}
		stores[i] = &pb.ObjectStore{
			Name:          os.Name,
			KeyPath:       os.KeyPath.toProto(),
			AutoIncrement: os.AutoIncrement,
			Indexes:       indexes,
		}
	}
	return &pb.DatabaseWithObjectStores{
		Name:         r.Name,
		Version:      r.Version,
		ObjectStores: stores,
	}
}

type rawDataEntry struct {
	Key        json.RawMessage `json:"key"`
	PrimaryKey json.RawMessage `json:"primaryKey"`
	Value      json.RawMessage `json:"value"`
}
