// Package audits implements the gRPC AuditsService by bridging to CDP.
package audits

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/audits"
)

type Server struct {
	pb.UnimplementedAuditsServiceServer
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
	if _, err := s.send(ctx, req.SessionId, "Audits.enable", nil); err != nil {
		return nil, fmt.Errorf("Audits.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Audits.disable", nil); err != nil {
		return nil, fmt.Errorf("Audits.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) GetEncodedResponse(ctx context.Context, req *pb.GetEncodedResponseRequest) (*pb.GetEncodedResponseResponse, error) {
	params := map[string]interface{}{
		"requestId": req.RequestId,
		"encoding":  req.Encoding,
	}
	if req.Quality != nil {
		params["quality"] = *req.Quality
	}
	if req.SizeOnly != nil {
		params["sizeOnly"] = *req.SizeOnly
	}
	result, err := s.send(ctx, req.SessionId, "Audits.getEncodedResponse", params)
	if err != nil {
		return nil, fmt.Errorf("Audits.getEncodedResponse: %w", err)
	}
	var raw struct {
		Body         *string `json:"body"`
		OriginalSize int32   `json:"originalSize"`
		EncodedSize  int32   `json:"encodedSize"`
	}
	if err := json.Unmarshal(result, &raw); err != nil {
		return nil, fmt.Errorf("Audits.getEncodedResponse: unmarshal: %w", err)
	}
	resp := &pb.GetEncodedResponseResponse{
		OriginalSize: raw.OriginalSize,
		EncodedSize:  raw.EncodedSize,
	}
	if raw.Body != nil {
		resp.Body = raw.Body
	}
	return resp, nil
}

func (s *Server) CheckContrast(ctx context.Context, req *pb.CheckContrastRequest) (*pb.CheckContrastResponse, error) {
	var params map[string]interface{}
	if req.ReportAaa != nil {
		params = map[string]interface{}{
			"reportAAA": *req.ReportAaa,
		}
	}
	if _, err := s.send(ctx, req.SessionId, "Audits.checkContrast", params); err != nil {
		return nil, fmt.Errorf("Audits.checkContrast: %w", err)
	}
	return &pb.CheckContrastResponse{}, nil
}

func (s *Server) CheckFormsIssues(ctx context.Context, req *pb.CheckFormsIssuesRequest) (*pb.CheckFormsIssuesResponse, error) {
	result, err := s.send(ctx, req.SessionId, "Audits.checkFormsIssues", nil)
	if err != nil {
		return nil, fmt.Errorf("Audits.checkFormsIssues: %w", err)
	}
	var raw struct {
		FormIssues []json.RawMessage `json:"formIssues"`
	}
	if err := json.Unmarshal(result, &raw); err != nil {
		return nil, fmt.Errorf("Audits.checkFormsIssues: unmarshal: %w", err)
	}
	issues := make([]string, len(raw.FormIssues))
	for i, fi := range raw.FormIssues {
		issues[i] = string(fi)
	}
	return &pb.CheckFormsIssuesResponse{FormIssues: issues}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeEventsRequest, stream pb.AuditsService_SubscribeEventsServer) error {
	ch := make(chan *pb.AuditsEvent, 64)
	defer close(ch)

	unsubscribe := s.client.On("Audits.issueAdded", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			Issue struct {
				Code    string          `json:"code"`
				Details json.RawMessage `json:"details"`
			} `json:"issue"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		ch <- &pb.AuditsEvent{
			Event: &pb.AuditsEvent_IssueAdded{
				IssueAdded: &pb.IssueAddedEvent{
					Issue: &pb.InspectorIssue{
						Code:    raw.Issue.Code,
						Details: string(raw.Issue.Details),
					},
				},
			},
		}
	})
	defer unsubscribe()

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
