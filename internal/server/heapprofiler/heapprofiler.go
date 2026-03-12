// Package heapprofiler implements the gRPC HeapProfilerService by bridging to CDP.
package heapprofiler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/heapprofiler"
)

type Server struct {
	pb.UnimplementedHeapProfilerServiceServer
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
	if _, err := s.send(ctx, req.SessionId, "HeapProfiler.enable", nil); err != nil {
		return nil, fmt.Errorf("HeapProfiler.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "HeapProfiler.disable", nil); err != nil {
		return nil, fmt.Errorf("HeapProfiler.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) StartTrackingHeapObjects(ctx context.Context, req *pb.StartTrackingHeapObjectsRequest) (*pb.StartTrackingHeapObjectsResponse, error) {
	params := map[string]interface{}{
		"trackAllocations": req.TrackAllocations,
	}
	if _, err := s.send(ctx, req.SessionId, "HeapProfiler.startTrackingHeapObjects", params); err != nil {
		return nil, fmt.Errorf("HeapProfiler.startTrackingHeapObjects: %w", err)
	}
	return &pb.StartTrackingHeapObjectsResponse{}, nil
}

func (s *Server) StopTrackingHeapObjects(ctx context.Context, req *pb.StopTrackingHeapObjectsRequest) (*pb.StopTrackingHeapObjectsResponse, error) {
	params := map[string]interface{}{
		"reportProgress":            req.ReportProgress,
		"treatGlobalObjectsAsRoots": req.TreatGlobalObjectsAsRoots,
		"captureNumericValue":       req.CaptureNumericValue,
		"exposeInternals":           req.ExposeInternals,
	}
	if _, err := s.send(ctx, req.SessionId, "HeapProfiler.stopTrackingHeapObjects", params); err != nil {
		return nil, fmt.Errorf("HeapProfiler.stopTrackingHeapObjects: %w", err)
	}
	return &pb.StopTrackingHeapObjectsResponse{}, nil
}

func (s *Server) TakeHeapSnapshot(ctx context.Context, req *pb.TakeHeapSnapshotRequest) (*pb.TakeHeapSnapshotResponse, error) {
	params := map[string]interface{}{
		"reportProgress":            req.ReportProgress,
		"treatGlobalObjectsAsRoots": req.TreatGlobalObjectsAsRoots,
		"captureNumericValue":       req.CaptureNumericValue,
		"exposeInternals":           req.ExposeInternals,
	}
	if _, err := s.send(ctx, req.SessionId, "HeapProfiler.takeHeapSnapshot", params); err != nil {
		return nil, fmt.Errorf("HeapProfiler.takeHeapSnapshot: %w", err)
	}
	return &pb.TakeHeapSnapshotResponse{}, nil
}

func (s *Server) CollectGarbage(ctx context.Context, req *pb.CollectGarbageRequest) (*pb.CollectGarbageResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "HeapProfiler.collectGarbage", nil); err != nil {
		return nil, fmt.Errorf("HeapProfiler.collectGarbage: %w", err)
	}
	return &pb.CollectGarbageResponse{}, nil
}

func (s *Server) GetObjectByHeapObjectId(ctx context.Context, req *pb.GetObjectByHeapObjectIdRequest) (*pb.GetObjectByHeapObjectIdResponse, error) {
	params := map[string]interface{}{
		"objectId": req.ObjectId,
	}
	if req.ObjectGroup != "" {
		params["objectGroup"] = req.ObjectGroup
	}
	result, err := s.send(ctx, req.SessionId, "HeapProfiler.getObjectByHeapObjectId", params)
	if err != nil {
		return nil, fmt.Errorf("HeapProfiler.getObjectByHeapObjectId: %w", err)
	}
	var resp struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("HeapProfiler.getObjectByHeapObjectId: unmarshal: %w", err)
	}
	return &pb.GetObjectByHeapObjectIdResponse{Result: string(resp.Result)}, nil
}

func (s *Server) AddInspectedHeapObject(ctx context.Context, req *pb.AddInspectedHeapObjectRequest) (*pb.AddInspectedHeapObjectResponse, error) {
	params := map[string]interface{}{
		"heapObjectId": req.HeapObjectId,
	}
	if _, err := s.send(ctx, req.SessionId, "HeapProfiler.addInspectedHeapObject", params); err != nil {
		return nil, fmt.Errorf("HeapProfiler.addInspectedHeapObject: %w", err)
	}
	return &pb.AddInspectedHeapObjectResponse{}, nil
}

func (s *Server) GetHeapObjectId(ctx context.Context, req *pb.GetHeapObjectIdRequest) (*pb.GetHeapObjectIdResponse, error) {
	params := map[string]interface{}{
		"objectId": req.ObjectId,
	}
	result, err := s.send(ctx, req.SessionId, "HeapProfiler.getHeapObjectId", params)
	if err != nil {
		return nil, fmt.Errorf("HeapProfiler.getHeapObjectId: %w", err)
	}
	var resp struct {
		HeapSnapshotObjectId string `json:"heapSnapshotObjectId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("HeapProfiler.getHeapObjectId: unmarshal: %w", err)
	}
	return &pb.GetHeapObjectIdResponse{HeapSnapshotObjectId: resp.HeapSnapshotObjectId}, nil
}

func (s *Server) StartSampling(ctx context.Context, req *pb.StartSamplingRequest) (*pb.StartSamplingResponse, error) {
	params := map[string]interface{}{}
	if req.SamplingInterval != 0 {
		params["samplingInterval"] = req.SamplingInterval
	}
	if req.IncludeObjectsCollectedByMajorGc {
		params["includeObjectsCollectedByMajorGC"] = req.IncludeObjectsCollectedByMajorGc
	}
	if req.IncludeObjectsCollectedByMinorGc {
		params["includeObjectsCollectedByMinorGC"] = req.IncludeObjectsCollectedByMinorGc
	}
	var p interface{}
	if len(params) > 0 {
		p = params
	}
	if _, err := s.send(ctx, req.SessionId, "HeapProfiler.startSampling", p); err != nil {
		return nil, fmt.Errorf("HeapProfiler.startSampling: %w", err)
	}
	return &pb.StartSamplingResponse{}, nil
}

func (s *Server) StopSampling(ctx context.Context, req *pb.StopSamplingRequest) (*pb.StopSamplingResponse, error) {
	result, err := s.send(ctx, req.SessionId, "HeapProfiler.stopSampling", nil)
	if err != nil {
		return nil, fmt.Errorf("HeapProfiler.stopSampling: %w", err)
	}
	var resp struct {
		Profile json.RawMessage `json:"profile"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("HeapProfiler.stopSampling: unmarshal: %w", err)
	}
	profile, err := convertSamplingHeapProfile(resp.Profile)
	if err != nil {
		return nil, fmt.Errorf("HeapProfiler.stopSampling: convert profile: %w", err)
	}
	return &pb.StopSamplingResponse{Profile: profile}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeEventsRequest, stream pb.HeapProfilerService_SubscribeEventsServer) error {
	ch := make(chan *pb.HeapProfilerEvent, 64)
	defer close(ch)

	unsub1 := s.client.On("HeapProfiler.addHeapSnapshotChunk", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			Chunk string `json:"chunk"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		ch <- &pb.HeapProfilerEvent{
			Event: &pb.HeapProfilerEvent_AddHeapSnapshotChunk{
				AddHeapSnapshotChunk: &pb.AddHeapSnapshotChunkEvent{
					Chunk: raw.Chunk,
				},
			},
		}
	})
	defer unsub1()

	unsub2 := s.client.On("HeapProfiler.reportHeapSnapshotProgress", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			Done     int32 `json:"done"`
			Total    int32 `json:"total"`
			Finished bool  `json:"finished"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		ch <- &pb.HeapProfilerEvent{
			Event: &pb.HeapProfilerEvent_ReportHeapSnapshotProgress{
				ReportHeapSnapshotProgress: &pb.ReportHeapSnapshotProgressEvent{
					Done:     raw.Done,
					Total:    raw.Total,
					Finished: raw.Finished,
				},
			},
		}
	})
	defer unsub2()

	unsub3 := s.client.On("HeapProfiler.lastSeenObjectId", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			LastSeenObjectId int32   `json:"lastSeenObjectId"`
			Timestamp        float64 `json:"timestamp"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		ch <- &pb.HeapProfilerEvent{
			Event: &pb.HeapProfilerEvent_LastSeenObjectId{
				LastSeenObjectId: &pb.LastSeenObjectIdEvent{
					LastSeenObjectId: raw.LastSeenObjectId,
					Timestamp:        raw.Timestamp,
				},
			},
		}
	})
	defer unsub3()

	unsub4 := s.client.On("HeapProfiler.heapStatsUpdate", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			StatsUpdate []int32 `json:"statsUpdate"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		ch <- &pb.HeapProfilerEvent{
			Event: &pb.HeapProfilerEvent_HeapStatsUpdate{
				HeapStatsUpdate: &pb.HeapStatsUpdateEvent{
					StatsUpdate: raw.StatsUpdate,
				},
			},
		}
	})
	defer unsub4()

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

// ==================== Raw types and converters ====================

type rawCallFrame struct {
	FunctionName string `json:"functionName"`
	ScriptID     string `json:"scriptId"`
	URL          string `json:"url"`
	LineNumber   int32  `json:"lineNumber"`
	ColumnNumber int32  `json:"columnNumber"`
}

type rawSamplingHeapProfileNode struct {
	CallFrame rawCallFrame                 `json:"callFrame"`
	SelfSize  float64                      `json:"selfSize"`
	ID        int32                        `json:"id"`
	Children  []rawSamplingHeapProfileNode `json:"children"`
}

type rawSamplingHeapProfileSample struct {
	Size    float64 `json:"size"`
	NodeId  int32   `json:"nodeId"`
	Ordinal float64 `json:"ordinal"`
}

type rawSamplingHeapProfile struct {
	Head    rawSamplingHeapProfileNode     `json:"head"`
	Samples []rawSamplingHeapProfileSample `json:"samples"`
}

func convertCallFrame(raw *rawCallFrame) *pb.CallFrame {
	return &pb.CallFrame{
		FunctionName: raw.FunctionName,
		ScriptId:     raw.ScriptID,
		Url:          raw.URL,
		LineNumber:   raw.LineNumber,
		ColumnNumber: raw.ColumnNumber,
	}
}

func convertSamplingHeapProfileNode(raw *rawSamplingHeapProfileNode) *pb.SamplingHeapProfileNode {
	children := make([]*pb.SamplingHeapProfileNode, len(raw.Children))
	for i := range raw.Children {
		children[i] = convertSamplingHeapProfileNode(&raw.Children[i])
	}
	return &pb.SamplingHeapProfileNode{
		CallFrame: convertCallFrame(&raw.CallFrame),
		SelfSize:  raw.SelfSize,
		Id:        raw.ID,
		Children:  children,
	}
}

func convertSamplingHeapProfile(data json.RawMessage) (*pb.SamplingHeapProfile, error) {
	var raw rawSamplingHeapProfile
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	samples := make([]*pb.SamplingHeapProfileSample, len(raw.Samples))
	for i, s := range raw.Samples {
		samples[i] = &pb.SamplingHeapProfileSample{
			Size:    s.Size,
			NodeId:  s.NodeId,
			Ordinal: s.Ordinal,
		}
	}
	return &pb.SamplingHeapProfile{
		Head:    convertSamplingHeapProfileNode(&raw.Head),
		Samples: samples,
	}, nil
}
