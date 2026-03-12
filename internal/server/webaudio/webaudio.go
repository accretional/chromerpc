// Package webaudio implements the gRPC WebAudioService by bridging to CDP.
package webaudio

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/webaudio"
)

type Server struct {
	pb.UnimplementedWebAudioServiceServer
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
	if _, err := s.send(ctx, req.SessionId, "WebAudio.enable", nil); err != nil {
		return nil, fmt.Errorf("WebAudio.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "WebAudio.disable", nil); err != nil {
		return nil, fmt.Errorf("WebAudio.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) GetRealtimeData(ctx context.Context, req *pb.GetRealtimeDataRequest) (*pb.GetRealtimeDataResponse, error) {
	params := map[string]interface{}{
		"contextId": req.ContextId,
	}
	result, err := s.send(ctx, req.SessionId, "WebAudio.getRealtimeData", params)
	if err != nil {
		return nil, fmt.Errorf("WebAudio.getRealtimeData: %w", err)
	}
	var resp struct {
		RealtimeData struct {
			CurrentTime              float64 `json:"currentTime"`
			RenderCapacity           float64 `json:"renderCapacity"`
			CallbackIntervalMean     float64 `json:"callbackIntervalMean"`
			CallbackIntervalVariance float64 `json:"callbackIntervalVariance"`
		} `json:"realtimeData"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("WebAudio.getRealtimeData: unmarshal: %w", err)
	}
	return &pb.GetRealtimeDataResponse{
		RealtimeData: &pb.ContextRealtimeData{
			CurrentTime:              resp.RealtimeData.CurrentTime,
			RenderCapacity:           resp.RealtimeData.RenderCapacity,
			CallbackIntervalMean:     resp.RealtimeData.CallbackIntervalMean,
			CallbackIntervalVariance: resp.RealtimeData.CallbackIntervalVariance,
		},
	}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeWebAudioEventsRequest, stream pb.WebAudioService_SubscribeEventsServer) error {
	eventCh := make(chan *pb.WebAudioEvent, 128)
	ctx := stream.Context()

	webAudioEvents := []string{
		"WebAudio.contextCreated",
		"WebAudio.contextWillBeDestroyed",
		"WebAudio.contextChanged",
		"WebAudio.audioListenerCreated",
		"WebAudio.audioListenerWillBeDestroyed",
		"WebAudio.audioNodeCreated",
		"WebAudio.audioNodeWillBeDestroyed",
		"WebAudio.audioParamCreated",
		"WebAudio.audioParamWillBeDestroyed",
	}

	var unregisters []func()
	for _, method := range webAudioEvents {
		method := method
		unreg := s.client.On(method, func(m string, params json.RawMessage, sessionID string) {
			if req.SessionId != "" && sessionID != req.SessionId {
				return
			}
			evt := convertWebAudioEvent(method, params)
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

// --- helper types for JSON unmarshalling ---

type cdpBaseAudioContext struct {
	ContextID            string  `json:"contextId"`
	ContextType          string  `json:"contextType"`
	ContextState         string  `json:"contextState"`
	SampleRate           float32 `json:"sampleRate"`
	CallbackBufferSize   float32 `json:"callbackBufferSize"`
	MaxOutputChannelCount float32 `json:"maxOutputChannelCount"`
	RealtimeData         *struct {
		CurrentTime              float64 `json:"currentTime"`
		RenderCapacity           float64 `json:"renderCapacity"`
		CallbackIntervalMean     float64 `json:"callbackIntervalMean"`
		CallbackIntervalVariance float64 `json:"callbackIntervalVariance"`
	} `json:"realtimeData"`
}

func (c *cdpBaseAudioContext) toProto() *pb.BaseAudioContext {
	if c == nil {
		return nil
	}
	ctx := &pb.BaseAudioContext{
		ContextId:             c.ContextID,
		ContextType:           c.ContextType,
		ContextState:          c.ContextState,
		SampleRate:            c.SampleRate,
		CallbackBufferSize:    c.CallbackBufferSize,
		MaxOutputChannelCount: c.MaxOutputChannelCount,
	}
	if c.RealtimeData != nil {
		ctx.RealtimeData = &pb.ContextRealtimeData{
			CurrentTime:              c.RealtimeData.CurrentTime,
			RenderCapacity:           c.RealtimeData.RenderCapacity,
			CallbackIntervalMean:     c.RealtimeData.CallbackIntervalMean,
			CallbackIntervalVariance: c.RealtimeData.CallbackIntervalVariance,
		}
	}
	return ctx
}

type cdpAudioListener struct {
	ListenerID string `json:"listenerId"`
	ContextID  string `json:"contextId"`
}

func (l *cdpAudioListener) toProto() *pb.AudioListener {
	if l == nil {
		return nil
	}
	return &pb.AudioListener{
		ListenerId: l.ListenerID,
		ContextId:  l.ContextID,
	}
}

type cdpAudioNode struct {
	NodeID                string  `json:"nodeId"`
	ContextID             string  `json:"contextId"`
	NodeType              string  `json:"nodeType"`
	NumberOfInputs        float32 `json:"numberOfInputs"`
	NumberOfOutputs       float32 `json:"numberOfOutputs"`
	ChannelCount          float32 `json:"channelCount"`
	ChannelCountMode      string  `json:"channelCountMode"`
	ChannelInterpretation string  `json:"channelInterpretation"`
}

func (n *cdpAudioNode) toProto() *pb.AudioNode {
	if n == nil {
		return nil
	}
	return &pb.AudioNode{
		NodeId:                n.NodeID,
		ContextId:             n.ContextID,
		NodeType:              n.NodeType,
		NumberOfInputs:        n.NumberOfInputs,
		NumberOfOutputs:       n.NumberOfOutputs,
		ChannelCount:          n.ChannelCount,
		ChannelCountMode:      n.ChannelCountMode,
		ChannelInterpretation: n.ChannelInterpretation,
	}
}

type cdpAudioParam struct {
	ParamID        string  `json:"paramId"`
	NodeID         string  `json:"nodeId"`
	ContextID      string  `json:"contextId"`
	ParamType      string  `json:"paramType"`
	AutomationRate string  `json:"automationRate"`
	DefaultValue   float32 `json:"defaultValue"`
	MinValue       float32 `json:"minValue"`
	MaxValue       float32 `json:"maxValue"`
}

func (p *cdpAudioParam) toProto() *pb.AudioParam {
	if p == nil {
		return nil
	}
	return &pb.AudioParam{
		ParamId:        p.ParamID,
		NodeId:         p.NodeID,
		ContextId:      p.ContextID,
		ParamType:      p.ParamType,
		AutomationRate: p.AutomationRate,
		DefaultValue:   p.DefaultValue,
		MinValue:       p.MinValue,
		MaxValue:       p.MaxValue,
	}
}

func convertWebAudioEvent(method string, params json.RawMessage) *pb.WebAudioEvent {
	switch method {
	case "WebAudio.contextCreated":
		var d struct {
			Context cdpBaseAudioContext `json:"context"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.WebAudioEvent{Event: &pb.WebAudioEvent_ContextCreated{
			ContextCreated: &pb.ContextCreatedEvent{Context: d.Context.toProto()},
		}}

	case "WebAudio.contextWillBeDestroyed":
		var d struct {
			ContextID string `json:"contextId"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.WebAudioEvent{Event: &pb.WebAudioEvent_ContextWillBeDestroyed{
			ContextWillBeDestroyed: &pb.ContextWillBeDestroyedEvent{ContextId: d.ContextID},
		}}

	case "WebAudio.contextChanged":
		var d struct {
			Context cdpBaseAudioContext `json:"context"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.WebAudioEvent{Event: &pb.WebAudioEvent_ContextChanged{
			ContextChanged: &pb.ContextChangedEvent{Context: d.Context.toProto()},
		}}

	case "WebAudio.audioListenerCreated":
		var d struct {
			Listener cdpAudioListener `json:"listener"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.WebAudioEvent{Event: &pb.WebAudioEvent_AudioListenerCreated{
			AudioListenerCreated: &pb.AudioListenerCreatedEvent{Listener: d.Listener.toProto()},
		}}

	case "WebAudio.audioListenerWillBeDestroyed":
		var d struct {
			ContextID  string `json:"contextId"`
			ListenerID string `json:"listenerId"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.WebAudioEvent{Event: &pb.WebAudioEvent_AudioListenerWillBeDestroyed{
			AudioListenerWillBeDestroyed: &pb.AudioListenerWillBeDestroyedEvent{
				ContextId:  d.ContextID,
				ListenerId: d.ListenerID,
			},
		}}

	case "WebAudio.audioNodeCreated":
		var d struct {
			Node cdpAudioNode `json:"node"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.WebAudioEvent{Event: &pb.WebAudioEvent_AudioNodeCreated{
			AudioNodeCreated: &pb.AudioNodeCreatedEvent{Node: d.Node.toProto()},
		}}

	case "WebAudio.audioNodeWillBeDestroyed":
		var d struct {
			ContextID string `json:"contextId"`
			NodeID    string `json:"nodeId"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.WebAudioEvent{Event: &pb.WebAudioEvent_AudioNodeWillBeDestroyed{
			AudioNodeWillBeDestroyed: &pb.AudioNodeWillBeDestroyedEvent{
				ContextId: d.ContextID,
				NodeId:    d.NodeID,
			},
		}}

	case "WebAudio.audioParamCreated":
		var d struct {
			Param cdpAudioParam `json:"param"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.WebAudioEvent{Event: &pb.WebAudioEvent_AudioParamCreated{
			AudioParamCreated: &pb.AudioParamCreatedEvent{Param: d.Param.toProto()},
		}}

	case "WebAudio.audioParamWillBeDestroyed":
		var d struct {
			ContextID string `json:"contextId"`
			NodeID    string `json:"nodeId"`
			ParamID   string `json:"paramId"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.WebAudioEvent{Event: &pb.WebAudioEvent_AudioParamWillBeDestroyed{
			AudioParamWillBeDestroyed: &pb.AudioParamWillBeDestroyedEvent{
				ContextId: d.ContextID,
				NodeId:    d.NodeID,
				ParamId:   d.ParamID,
			},
		}}
	}
	return nil
}
