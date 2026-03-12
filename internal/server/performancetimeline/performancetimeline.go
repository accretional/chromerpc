// Package performancetimeline implements the gRPC PerformanceTimelineService by bridging to CDP.
package performancetimeline

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/performancetimeline"
)

type Server struct {
	pb.UnimplementedPerformanceTimelineServiceServer
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
	params := map[string]interface{}{
		"eventTypes": req.EventTypes,
	}
	if _, err := s.send(ctx, req.SessionId, "PerformanceTimeline.enable", params); err != nil {
		return nil, fmt.Errorf("PerformanceTimeline.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribePerformanceTimelineEventsRequest, stream pb.PerformanceTimelineService_SubscribeEventsServer) error {
	eventCh := make(chan *pb.PerformanceTimelineEvent, 128)
	ctx := stream.Context()

	// Register a wildcard handler for all PerformanceTimeline.* events.
	unregister := s.client.On("PerformanceTimeline.", func(method string, params json.RawMessage, sessionID string) {
		if req.SessionId != "" && sessionID != req.SessionId {
			return
		}
		evt := convertPerformanceTimelineEvent(method, params)
		if evt != nil {
			select {
			case eventCh <- evt:
			default:
			}
		}
	})

	// Also register specific events since the On() matching is exact.
	perfEvents := []string{
		"PerformanceTimeline.timelineEventAdded",
	}
	unregisters := make([]func(), 0, len(perfEvents)+1)
	unregisters = append(unregisters, unregister)
	for _, method := range perfEvents {
		method := method
		unreg := s.client.On(method, func(m string, params json.RawMessage, sessionID string) {
			if req.SessionId != "" && sessionID != req.SessionId {
				return
			}
			evt := convertPerformanceTimelineEvent(method, params)
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

func convertPerformanceTimelineEvent(method string, params json.RawMessage) *pb.PerformanceTimelineEvent {
	switch method {
	case "PerformanceTimeline.timelineEventAdded":
		var d struct {
			Event struct {
				FrameID  string  `json:"frameId"`
				Type     string  `json:"type"`
				Name     string  `json:"name"`
				Time     float64 `json:"time"`
				Duration float64 `json:"duration"`
				LcpDetails *struct {
					RenderTime float64 `json:"renderTime"`
					LoadTime   float64 `json:"loadTime"`
					Size       float64 `json:"size"`
					ElementID  string  `json:"elementId"`
					URL        string  `json:"url"`
					NodeID     int32   `json:"nodeId"`
				} `json:"lcpDetails"`
				LayoutShiftDetails *struct {
					Value          float64 `json:"value"`
					HadRecentInput bool    `json:"hadRecentInput"`
					LastInputTime  float64 `json:"lastInputTime"`
					Sources        []struct {
						NodeID int32 `json:"nodeId"`
					} `json:"sources"`
				} `json:"layoutShiftDetails"`
			} `json:"event"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		te := &pb.TimelineEvent{
			FrameId:  d.Event.FrameID,
			Type:     d.Event.Type,
			Name:     d.Event.Name,
			Time:     d.Event.Time,
			Duration: d.Event.Duration,
		}
		if d.Event.LcpDetails != nil {
			te.LcpDetails = &pb.LargestContentfulPaint{
				RenderTime: d.Event.LcpDetails.RenderTime,
				LoadTime:   d.Event.LcpDetails.LoadTime,
				Size:       d.Event.LcpDetails.Size,
				ElementId:  d.Event.LcpDetails.ElementID,
				Url:        d.Event.LcpDetails.URL,
				NodeId:     d.Event.LcpDetails.NodeID,
			}
		}
		if d.Event.LayoutShiftDetails != nil {
			ls := &pb.LayoutShift{
				Value:          d.Event.LayoutShiftDetails.Value,
				HadRecentInput: d.Event.LayoutShiftDetails.HadRecentInput,
				LastInputTime:  d.Event.LayoutShiftDetails.LastInputTime,
			}
			for _, src := range d.Event.LayoutShiftDetails.Sources {
				ls.Sources = append(ls.Sources, &pb.LayoutShiftAttribution{
					NodeId: src.NodeID,
				})
			}
			te.LayoutShiftDetails = ls
		}
		return &pb.PerformanceTimelineEvent{Event: &pb.PerformanceTimelineEvent_TimelineEventAdded{
			TimelineEventAdded: &pb.TimelineEventAddedEvent{Event: te},
		}}

	default:
		return nil
	}
}
