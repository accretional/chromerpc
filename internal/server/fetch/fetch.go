// Package fetch implements the gRPC FetchService by bridging to CDP.
package fetch

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/fetch"
	"google.golang.org/grpc"
)

// Server implements the cdp.fetch.FetchService gRPC service.
type Server struct {
	pb.UnimplementedFetchServiceServer
	client *cdpclient.Client
}

// New creates a new Fetch gRPC server backed by the given CDP client.
func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	params := map[string]interface{}{}
	if len(req.Patterns) > 0 {
		patterns := make([]map[string]interface{}, len(req.Patterns))
		for i, p := range req.Patterns {
			pat := map[string]interface{}{}
			if p.UrlPattern != "" {
				pat["urlPattern"] = p.UrlPattern
			}
			if p.ResourceType != "" {
				pat["resourceType"] = p.ResourceType
			}
			if p.RequestStage != "" {
				pat["requestStage"] = p.RequestStage
			}
			patterns[i] = pat
		}
		params["patterns"] = patterns
	}
	if req.HandleAuthRequests {
		params["handleAuthRequests"] = true
	}
	var err error
	if len(params) > 0 {
		_, err = s.client.Send(ctx, "Fetch.enable", params)
	} else {
		_, err = s.client.Send(ctx, "Fetch.enable", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Fetch.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "Fetch.disable", nil); err != nil {
		return nil, fmt.Errorf("Fetch.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) ContinueRequest(ctx context.Context, req *pb.ContinueRequestRequest) (*pb.ContinueRequestResponse, error) {
	params := map[string]interface{}{
		"requestId": req.RequestId,
	}
	if req.Url != "" {
		params["url"] = req.Url
	}
	if req.Method != "" {
		params["method"] = req.Method
	}
	if req.PostData != "" {
		params["postData"] = req.PostData
	}
	if len(req.Headers) > 0 {
		params["headers"] = headerEntriesToCDP(req.Headers)
	}
	if req.InterceptResponse {
		params["interceptResponse"] = true
	}
	if _, err := s.client.Send(ctx, "Fetch.continueRequest", params); err != nil {
		return nil, fmt.Errorf("Fetch.continueRequest: %w", err)
	}
	return &pb.ContinueRequestResponse{}, nil
}

func (s *Server) FulfillRequest(ctx context.Context, req *pb.FulfillRequestRequest) (*pb.FulfillRequestResponse, error) {
	params := map[string]interface{}{
		"requestId":    req.RequestId,
		"responseCode": req.ResponseCode,
	}
	if len(req.ResponseHeaders) > 0 {
		params["responseHeaders"] = headerEntriesToCDP(req.ResponseHeaders)
	}
	if len(req.Body) > 0 {
		params["body"] = base64.StdEncoding.EncodeToString(req.Body)
	}
	if req.ResponsePhrase != "" {
		params["responsePhrase"] = req.ResponsePhrase
	}
	if req.BinaryResponseHeaders != "" {
		params["binaryResponseHeaders"] = req.BinaryResponseHeaders
	}
	if _, err := s.client.Send(ctx, "Fetch.fulfillRequest", params); err != nil {
		return nil, fmt.Errorf("Fetch.fulfillRequest: %w", err)
	}
	return &pb.FulfillRequestResponse{}, nil
}

func (s *Server) FailRequest(ctx context.Context, req *pb.FailRequestRequest) (*pb.FailRequestResponse, error) {
	params := map[string]interface{}{
		"requestId": req.RequestId,
		"reason":    req.Reason,
	}
	if _, err := s.client.Send(ctx, "Fetch.failRequest", params); err != nil {
		return nil, fmt.Errorf("Fetch.failRequest: %w", err)
	}
	return &pb.FailRequestResponse{}, nil
}

func (s *Server) ContinueWithAuth(ctx context.Context, req *pb.ContinueWithAuthRequest) (*pb.ContinueWithAuthResponse, error) {
	params := map[string]interface{}{
		"requestId": req.RequestId,
	}
	if req.AuthChallengeResponse != nil {
		acr := map[string]interface{}{
			"response": req.AuthChallengeResponse.Response,
		}
		if req.AuthChallengeResponse.Username != "" {
			acr["username"] = req.AuthChallengeResponse.Username
		}
		if req.AuthChallengeResponse.Password != "" {
			acr["password"] = req.AuthChallengeResponse.Password
		}
		params["authChallengeResponse"] = acr
	}
	if _, err := s.client.Send(ctx, "Fetch.continueWithAuth", params); err != nil {
		return nil, fmt.Errorf("Fetch.continueWithAuth: %w", err)
	}
	return &pb.ContinueWithAuthResponse{}, nil
}

func (s *Server) GetResponseBody(ctx context.Context, req *pb.GetResponseBodyRequest) (*pb.GetResponseBodyResponse, error) {
	params := map[string]interface{}{
		"requestId": req.RequestId,
	}
	result, err := s.client.Send(ctx, "Fetch.getResponseBody", params)
	if err != nil {
		return nil, fmt.Errorf("Fetch.getResponseBody: %w", err)
	}
	var resp struct {
		Body          string `json:"body"`
		Base64Encoded bool   `json:"base64Encoded"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Fetch.getResponseBody: unmarshal: %w", err)
	}
	return &pb.GetResponseBodyResponse{
		Body:          resp.Body,
		Base64Encoded: resp.Base64Encoded,
	}, nil
}

func (s *Server) TakeResponseBodyAsStream(ctx context.Context, req *pb.TakeResponseBodyAsStreamRequest) (*pb.TakeResponseBodyAsStreamResponse, error) {
	params := map[string]interface{}{
		"requestId": req.RequestId,
	}
	result, err := s.client.Send(ctx, "Fetch.takeResponseBodyAsStream", params)
	if err != nil {
		return nil, fmt.Errorf("Fetch.takeResponseBodyAsStream: %w", err)
	}
	var resp struct {
		Stream string `json:"stream"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Fetch.takeResponseBodyAsStream: unmarshal: %w", err)
	}
	return &pb.TakeResponseBodyAsStreamResponse{
		Stream: resp.Stream,
	}, nil
}

func (s *Server) ContinueResponse(ctx context.Context, req *pb.ContinueResponseRequest) (*pb.ContinueResponseResponse, error) {
	params := map[string]interface{}{
		"requestId": req.RequestId,
	}
	if req.ResponseCode != 0 {
		params["responseCode"] = req.ResponseCode
	}
	if req.ResponsePhrase != "" {
		params["responsePhrase"] = req.ResponsePhrase
	}
	if len(req.ResponseHeaders) > 0 {
		params["responseHeaders"] = headerEntriesToCDP(req.ResponseHeaders)
	}
	if req.BinaryResponseHeaders != "" {
		params["binaryResponseHeaders"] = req.BinaryResponseHeaders
	}
	if _, err := s.client.Send(ctx, "Fetch.continueResponse", params); err != nil {
		return nil, fmt.Errorf("Fetch.continueResponse: %w", err)
	}
	return &pb.ContinueResponseResponse{}, nil
}

// SubscribeEvents streams CDP Fetch events to the gRPC client.
func (s *Server) SubscribeEvents(req *pb.SubscribeEventsRequest, stream grpc.ServerStreamingServer[pb.FetchEvent]) error {
	eventCh := make(chan *pb.FetchEvent, 128)
	ctx := stream.Context()

	fetchEvents := []string{
		"Fetch.requestPaused",
		"Fetch.authRequired",
	}

	unregisters := make([]func(), len(fetchEvents))
	for i, method := range fetchEvents {
		method := method
		unregisters[i] = s.client.On(method, func(m string, params json.RawMessage, sessionID string) {
			evt := convertFetchEvent(method, params)
			if evt != nil {
				select {
				case eventCh <- evt:
				default:
				}
			}
		})
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

// --- internal helpers ---

func headerEntriesToCDP(entries []*pb.HeaderEntry) []map[string]interface{} {
	result := make([]map[string]interface{}, len(entries))
	for i, h := range entries {
		result[i] = map[string]interface{}{
			"name":  h.Name,
			"value": h.Value,
		}
	}
	return result
}

// --- Event conversion ---

func convertFetchEvent(method string, params json.RawMessage) *pb.FetchEvent {
	switch method {
	case "Fetch.requestPaused":
		var d struct {
			RequestID           string `json:"requestId"`
			Request             struct {
				URL      string            `json:"url"`
				Method   string            `json:"method"`
				PostData string            `json:"postData"`
				Headers  map[string]string `json:"headers"`
			} `json:"request"`
			FrameID             string `json:"frameId"`
			ResourceType        string `json:"resourceType"`
			ResponseErrorReason string `json:"responseErrorReason"`
			ResponseStatusCode  int32  `json:"responseStatusCode"`
			ResponseStatusText  string `json:"responseStatusText"`
			ResponseHeaders     []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"responseHeaders"`
			NetworkID          string `json:"networkId"`
			RedirectedRequestID string `json:"redirectedRequestId"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		responseHeaders := make([]*pb.HeaderEntry, len(d.ResponseHeaders))
		for i, h := range d.ResponseHeaders {
			responseHeaders[i] = &pb.HeaderEntry{Name: h.Name, Value: h.Value}
		}
		return &pb.FetchEvent{Event: &pb.FetchEvent_RequestPaused{
			RequestPaused: &pb.RequestPausedEvent{
				RequestId: d.RequestID,
				Request: &pb.RequestInfo{
					Url:      d.Request.URL,
					Method:   d.Request.Method,
					PostData: d.Request.PostData,
					Headers:  d.Request.Headers,
				},
				FrameId:             d.FrameID,
				ResourceType:        d.ResourceType,
				ResponseErrorReason: d.ResponseErrorReason,
				ResponseStatusCode:  d.ResponseStatusCode,
				ResponseStatusText:  d.ResponseStatusText,
				ResponseHeaders:     responseHeaders,
				NetworkId:           d.NetworkID,
				RedirectedRequestId: d.RedirectedRequestID,
			},
		}}

	case "Fetch.authRequired":
		var d struct {
			RequestID    string `json:"requestId"`
			Request      struct {
				URL      string            `json:"url"`
				Method   string            `json:"method"`
				PostData string            `json:"postData"`
				Headers  map[string]string `json:"headers"`
			} `json:"request"`
			FrameID      string `json:"frameId"`
			ResourceType string `json:"resourceType"`
			AuthChallenge struct {
				Source string `json:"source"`
				Origin string `json:"origin"`
				Scheme string `json:"scheme"`
				Realm  string `json:"realm"`
			} `json:"authChallenge"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.FetchEvent{Event: &pb.FetchEvent_AuthRequired{
			AuthRequired: &pb.AuthRequiredEvent{
				RequestId: d.RequestID,
				Request: &pb.RequestInfo{
					Url:      d.Request.URL,
					Method:   d.Request.Method,
					PostData: d.Request.PostData,
					Headers:  d.Request.Headers,
				},
				FrameId:      d.FrameID,
				ResourceType: d.ResourceType,
				AuthChallenge: &pb.AuthChallenge{
					Source: d.AuthChallenge.Source,
					Origin: d.AuthChallenge.Origin,
					Scheme: d.AuthChallenge.Scheme,
					Realm:  d.AuthChallenge.Realm,
				},
			},
		}}
	}
	return nil
}
