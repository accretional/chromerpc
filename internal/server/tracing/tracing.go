// Package tracing implements the gRPC TracingService by bridging to CDP.
package tracing

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/tracing"
)

type Server struct {
	pb.UnimplementedTracingServiceServer
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


func (s *Server) Start(ctx context.Context, req *pb.StartRequest) (*pb.StartResponse, error) {
	params := map[string]interface{}{}
	if req.Categories != "" {
		params["categories"] = req.Categories
	}
	if req.Options != "" {
		params["options"] = req.Options
	}
	if req.BufferUsageReportingInterval != 0 {
		params["bufferUsageReportingInterval"] = req.BufferUsageReportingInterval
	}
	if req.TransferMode != "" {
		params["transferMode"] = req.TransferMode
	}
	if req.StreamFormat != "" {
		params["streamFormat"] = req.StreamFormat
	}
	if req.StreamCompression != "" {
		params["streamCompression"] = req.StreamCompression
	}
	if req.TraceConfig != nil {
		tc := map[string]interface{}{}
		if req.TraceConfig.RecordMode != "" {
			tc["recordMode"] = req.TraceConfig.RecordMode
		}
		if req.TraceConfig.TraceBufferSizeInKb != 0 {
			tc["traceBufferSizeInKb"] = req.TraceConfig.TraceBufferSizeInKb
		}
		if req.TraceConfig.EnableSampling {
			tc["enableSampling"] = true
		}
		if req.TraceConfig.EnableSystrace {
			tc["enableSystrace"] = true
		}
		if req.TraceConfig.EnableArgumentFilter {
			tc["enableArgumentFilter"] = true
		}
		if len(req.TraceConfig.IncludedCategories) > 0 {
			tc["includedCategories"] = req.TraceConfig.IncludedCategories
		}
		if len(req.TraceConfig.ExcludedCategories) > 0 {
			tc["excludedCategories"] = req.TraceConfig.ExcludedCategories
		}
		if len(req.TraceConfig.SyntheticDelays) > 0 {
			tc["syntheticDelays"] = req.TraceConfig.SyntheticDelays
		}
		if req.TraceConfig.MemoryDumpConfig != "" {
			tc["memoryDumpConfig"] = json.RawMessage(req.TraceConfig.MemoryDumpConfig)
		}
		params["traceConfig"] = tc
	}
	if len(params) > 0 {
		_, err := s.send(ctx, req.SessionId, "Tracing.start", params)
		if err != nil {
			return nil, fmt.Errorf("Tracing.start: %w", err)
		}
	} else {
		if _, err := s.send(ctx, req.SessionId, "Tracing.start", nil); err != nil {
			return nil, fmt.Errorf("Tracing.start: %w", err)
		}
	}
	return &pb.StartResponse{}, nil
}

func (s *Server) End(ctx context.Context, req *pb.EndRequest) (*pb.EndResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Tracing.end", nil); err != nil {
		return nil, fmt.Errorf("Tracing.end: %w", err)
	}
	return &pb.EndResponse{}, nil
}

func (s *Server) GetCategories(ctx context.Context, req *pb.GetCategoriesRequest) (*pb.GetCategoriesResponse, error) {
	result, err := s.send(ctx, req.SessionId, "Tracing.getCategories", nil)
	if err != nil {
		return nil, fmt.Errorf("Tracing.getCategories: %w", err)
	}
	var resp struct {
		Categories []string `json:"categories"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Tracing.getCategories: unmarshal: %w", err)
	}
	return &pb.GetCategoriesResponse{Categories: resp.Categories}, nil
}

func (s *Server) RecordClockSyncMarker(ctx context.Context, req *pb.RecordClockSyncMarkerRequest) (*pb.RecordClockSyncMarkerResponse, error) {
	params := map[string]interface{}{
		"syncId": req.SyncId,
	}
	if _, err := s.send(ctx, req.SessionId, "Tracing.recordClockSyncMarker", params); err != nil {
		return nil, fmt.Errorf("Tracing.recordClockSyncMarker: %w", err)
	}
	return &pb.RecordClockSyncMarkerResponse{}, nil
}

func (s *Server) RequestMemoryDump(ctx context.Context, req *pb.RequestMemoryDumpRequest) (*pb.RequestMemoryDumpResponse, error) {
	params := map[string]interface{}{}
	if req.Deterministic {
		params["deterministic"] = true
	}
	if req.LevelOfDetail != "" {
		params["levelOfDetail"] = req.LevelOfDetail
	}
	var result json.RawMessage
	var err error
	if len(params) > 0 {
		result, err = s.send(ctx, req.SessionId, "Tracing.requestMemoryDump", params)
	} else {
		result, err = s.send(ctx, req.SessionId, "Tracing.requestMemoryDump", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Tracing.requestMemoryDump: %w", err)
	}
	var resp struct {
		DumpGuid string `json:"dumpGuid"`
		Success  bool   `json:"success"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Tracing.requestMemoryDump: unmarshal: %w", err)
	}
	return &pb.RequestMemoryDumpResponse{
		DumpGuid: resp.DumpGuid,
		Success:  resp.Success,
	}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeEventsRequest, stream pb.TracingService_SubscribeEventsServer) error {
	ch := make(chan *pb.TracingEvent, 64)
	defer close(ch)

	events := []string{
		"Tracing.bufferUsage",
		"Tracing.dataCollected",
		"Tracing.tracingComplete",
	}
	var unsubs []func()
	for _, evt := range events {
		evt := evt
		unsub := s.client.On(evt, func(_ string, params json.RawMessage, _ string) {
			converted := convertTracingEvent(evt, params)
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

func convertTracingEvent(method string, params json.RawMessage) *pb.TracingEvent {
	switch method {
	case "Tracing.bufferUsage":
		var raw struct {
			PercentFull float64 `json:"percentFull"`
			EventCount  float64 `json:"eventCount"`
			Value       float64 `json:"value"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		return &pb.TracingEvent{Event: &pb.TracingEvent_BufferUsage{
			BufferUsage: &pb.BufferUsageEvent{
				PercentFull: raw.PercentFull,
				EventCount:  raw.EventCount,
				Value:       raw.Value,
			},
		}}
	case "Tracing.dataCollected":
		var raw struct {
			Value []json.RawMessage `json:"value"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		// Serialize the array of trace event objects as a single JSON string
		b, err := json.Marshal(raw.Value)
		if err != nil {
			return nil
		}
		return &pb.TracingEvent{Event: &pb.TracingEvent_DataCollected{
			DataCollected: &pb.DataCollectedEvent{
				Value: string(b),
			},
		}}
	case "Tracing.tracingComplete":
		var raw struct {
			DataLossOccurred  bool   `json:"dataLossOccurred"`
			Stream            string `json:"stream"`
			TraceFormat       string `json:"traceFormat"`
			StreamCompression string `json:"streamCompression"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		return &pb.TracingEvent{Event: &pb.TracingEvent_TracingComplete{
			TracingComplete: &pb.TracingCompleteEvent{
				DataLossOccurred:  raw.DataLossOccurred,
				Stream:            raw.Stream,
				TraceFormat:       raw.TraceFormat,
				StreamCompression: raw.StreamCompression,
			},
		}}
	}
	return nil
}
