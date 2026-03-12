// Package profiler implements the gRPC ProfilerService by bridging to CDP.
package profiler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/profiler"
)

type Server struct {
	pb.UnimplementedProfilerServiceServer
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
	if _, err := s.send(ctx, req.SessionId, "Profiler.enable", nil); err != nil {
		return nil, fmt.Errorf("Profiler.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Profiler.disable", nil); err != nil {
		return nil, fmt.Errorf("Profiler.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) SetSamplingInterval(ctx context.Context, req *pb.SetSamplingIntervalRequest) (*pb.SetSamplingIntervalResponse, error) {
	params := map[string]interface{}{
		"interval": req.Interval,
	}
	if _, err := s.send(ctx, req.SessionId, "Profiler.setSamplingInterval", params); err != nil {
		return nil, fmt.Errorf("Profiler.setSamplingInterval: %w", err)
	}
	return &pb.SetSamplingIntervalResponse{}, nil
}

func (s *Server) Start(ctx context.Context, req *pb.StartRequest) (*pb.StartResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Profiler.start", nil); err != nil {
		return nil, fmt.Errorf("Profiler.start: %w", err)
	}
	return &pb.StartResponse{}, nil
}

func (s *Server) Stop(ctx context.Context, req *pb.StopRequest) (*pb.StopResponse, error) {
	result, err := s.send(ctx, req.SessionId, "Profiler.stop", nil)
	if err != nil {
		return nil, fmt.Errorf("Profiler.stop: %w", err)
	}
	var resp struct {
		Profile json.RawMessage `json:"profile"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Profiler.stop: unmarshal: %w", err)
	}
	profile, err := convertProfile(resp.Profile)
	if err != nil {
		return nil, fmt.Errorf("Profiler.stop: convert profile: %w", err)
	}
	return &pb.StopResponse{Profile: profile}, nil
}

func (s *Server) StartPreciseCoverage(ctx context.Context, req *pb.StartPreciseCoverageRequest) (*pb.StartPreciseCoverageResponse, error) {
	params := map[string]interface{}{
		"callCount":             req.CallCount,
		"detailed":              req.Detailed,
		"allowTriggeredUpdates": req.AllowTriggeredUpdates,
	}
	result, err := s.send(ctx, req.SessionId, "Profiler.startPreciseCoverage", params)
	if err != nil {
		return nil, fmt.Errorf("Profiler.startPreciseCoverage: %w", err)
	}
	var resp struct {
		Timestamp float64 `json:"timestamp"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Profiler.startPreciseCoverage: unmarshal: %w", err)
	}
	return &pb.StartPreciseCoverageResponse{Timestamp: resp.Timestamp}, nil
}

func (s *Server) StopPreciseCoverage(ctx context.Context, req *pb.StopPreciseCoverageRequest) (*pb.StopPreciseCoverageResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Profiler.stopPreciseCoverage", nil); err != nil {
		return nil, fmt.Errorf("Profiler.stopPreciseCoverage: %w", err)
	}
	return &pb.StopPreciseCoverageResponse{}, nil
}

func (s *Server) TakePreciseCoverage(ctx context.Context, req *pb.TakePreciseCoverageRequest) (*pb.TakePreciseCoverageResponse, error) {
	result, err := s.send(ctx, req.SessionId, "Profiler.takePreciseCoverage", nil)
	if err != nil {
		return nil, fmt.Errorf("Profiler.takePreciseCoverage: %w", err)
	}
	var resp struct {
		Result    []rawScriptCoverage `json:"result"`
		Timestamp float64             `json:"timestamp"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Profiler.takePreciseCoverage: unmarshal: %w", err)
	}
	return &pb.TakePreciseCoverageResponse{
		Result:    convertScriptCoverages(resp.Result),
		Timestamp: resp.Timestamp,
	}, nil
}

func (s *Server) GetBestEffortCoverage(ctx context.Context, req *pb.GetBestEffortCoverageRequest) (*pb.GetBestEffortCoverageResponse, error) {
	result, err := s.send(ctx, req.SessionId, "Profiler.getBestEffortCoverage", nil)
	if err != nil {
		return nil, fmt.Errorf("Profiler.getBestEffortCoverage: %w", err)
	}
	var resp struct {
		Result []rawScriptCoverage `json:"result"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Profiler.getBestEffortCoverage: unmarshal: %w", err)
	}
	return &pb.GetBestEffortCoverageResponse{
		Result: convertScriptCoverages(resp.Result),
	}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeEventsRequest, stream pb.ProfilerService_SubscribeEventsServer) error {
	ch := make(chan *pb.ProfilerEvent, 64)
	defer close(ch)

	unsub1 := s.client.On("Profiler.consoleProfileStarted", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			ID       string       `json:"id"`
			Location rawCallFrame `json:"location"`
			Title    string       `json:"title"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		ch <- &pb.ProfilerEvent{
			Event: &pb.ProfilerEvent_ConsoleProfileStarted{
				ConsoleProfileStarted: &pb.ConsoleProfileStartedEvent{
					Id:       raw.ID,
					Location: convertCallFrame(&raw.Location),
					Title:    raw.Title,
				},
			},
		}
	})
	defer unsub1()

	unsub2 := s.client.On("Profiler.consoleProfileFinished", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			ID       string          `json:"id"`
			Location rawCallFrame    `json:"location"`
			Profile  json.RawMessage `json:"profile"`
			Title    string          `json:"title"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		profile, err := convertProfile(raw.Profile)
		if err != nil {
			return
		}
		ch <- &pb.ProfilerEvent{
			Event: &pb.ProfilerEvent_ConsoleProfileFinished{
				ConsoleProfileFinished: &pb.ConsoleProfileFinishedEvent{
					Id:       raw.ID,
					Location: convertCallFrame(&raw.Location),
					Profile:  profile,
					Title:    raw.Title,
				},
			},
		}
	})
	defer unsub2()

	unsub3 := s.client.On("Profiler.preciseCoverageDeltaUpdate", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			Timestamp float64             `json:"timestamp"`
			Occasion  string              `json:"occasion"`
			Result    []rawScriptCoverage `json:"result"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		ch <- &pb.ProfilerEvent{
			Event: &pb.ProfilerEvent_PreciseCoverageDeltaUpdate{
				PreciseCoverageDeltaUpdate: &pb.PreciseCoverageDeltaUpdateEvent{
					Timestamp: raw.Timestamp,
					Occasion:  raw.Occasion,
					Result:    convertScriptCoverages(raw.Result),
				},
			},
		}
	})
	defer unsub3()

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

type rawPositionTickInfo struct {
	Line  int32 `json:"line"`
	Ticks int32 `json:"ticks"`
}

type rawProfileNode struct {
	ID            int32                 `json:"id"`
	CallFrame     rawCallFrame          `json:"callFrame"`
	HitCount      int32                 `json:"hitCount"`
	Children      []int32               `json:"children"`
	DeoptReason   string                `json:"deoptReason"`
	PositionTicks []rawPositionTickInfo `json:"positionTicks"`
}

type rawProfile struct {
	Nodes      []rawProfileNode `json:"nodes"`
	StartTime  float64          `json:"startTime"`
	EndTime    float64          `json:"endTime"`
	Samples    []int32          `json:"samples"`
	TimeDeltas []int32          `json:"timeDeltas"`
}

type rawCoverageRange struct {
	StartOffset int32 `json:"startOffset"`
	EndOffset   int32 `json:"endOffset"`
	Count       int32 `json:"count"`
}

type rawFunctionCoverage struct {
	FunctionName    string             `json:"functionName"`
	Ranges          []rawCoverageRange `json:"ranges"`
	IsBlockCoverage bool               `json:"isBlockCoverage"`
}

type rawScriptCoverage struct {
	ScriptID  string                `json:"scriptId"`
	URL       string                `json:"url"`
	Functions []rawFunctionCoverage `json:"functions"`
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

func convertProfile(data json.RawMessage) (*pb.Profile, error) {
	var raw rawProfile
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	nodes := make([]*pb.ProfileNode, len(raw.Nodes))
	for i, n := range raw.Nodes {
		ticks := make([]*pb.PositionTickInfo, len(n.PositionTicks))
		for j, t := range n.PositionTicks {
			ticks[j] = &pb.PositionTickInfo{Line: t.Line, Ticks: t.Ticks}
		}
		nodes[i] = &pb.ProfileNode{
			Id:            n.ID,
			CallFrame:     convertCallFrame(&n.CallFrame),
			HitCount:      n.HitCount,
			Children:      n.Children,
			DeoptReason:   n.DeoptReason,
			PositionTicks: ticks,
		}
	}
	return &pb.Profile{
		Nodes:      nodes,
		StartTime:  raw.StartTime,
		EndTime:    raw.EndTime,
		Samples:    raw.Samples,
		TimeDeltas: raw.TimeDeltas,
	}, nil
}

func convertScriptCoverages(raws []rawScriptCoverage) []*pb.ScriptCoverage {
	result := make([]*pb.ScriptCoverage, len(raws))
	for i, sc := range raws {
		funcs := make([]*pb.FunctionCoverage, len(sc.Functions))
		for j, f := range sc.Functions {
			ranges := make([]*pb.CoverageRange, len(f.Ranges))
			for k, r := range f.Ranges {
				ranges[k] = &pb.CoverageRange{
					StartOffset: r.StartOffset,
					EndOffset:   r.EndOffset,
					Count:       r.Count,
				}
			}
			funcs[j] = &pb.FunctionCoverage{
				FunctionName:    f.FunctionName,
				Ranges:          ranges,
				IsBlockCoverage: f.IsBlockCoverage,
			}
		}
		result[i] = &pb.ScriptCoverage{
			ScriptId:  sc.ScriptID,
			Url:       sc.URL,
			Functions: funcs,
		}
	}
	return result
}
