// Package network implements the gRPC NetworkService by bridging to CDP over WebSocket.
package network

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/network"
)

// Server implements the cdp.network.NetworkService gRPC service.
type Server struct {
	pb.UnimplementedNetworkServiceServer
	client *cdpclient.Client
}

// New creates a new Network gRPC server backed by the given CDP client.
func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	params := map[string]interface{}{}
	if req.MaxTotalBufferSize > 0 {
		params["maxTotalBufferSize"] = req.MaxTotalBufferSize
	}
	if req.MaxResourceBufferSize > 0 {
		params["maxResourceBufferSize"] = req.MaxResourceBufferSize
	}
	if req.MaxPostDataSize > 0 {
		params["maxPostDataSize"] = req.MaxPostDataSize
	}
	var err error
	if len(params) > 0 {
		_, err = s.client.Send(ctx, "Network.enable", params)
	} else {
		_, err = s.client.Send(ctx, "Network.enable", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Network.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "Network.disable", nil); err != nil {
		return nil, fmt.Errorf("Network.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) SetCacheDisabled(ctx context.Context, req *pb.SetCacheDisabledRequest) (*pb.SetCacheDisabledResponse, error) {
	params := map[string]interface{}{
		"cacheDisabled": req.CacheDisabled,
	}
	if _, err := s.client.Send(ctx, "Network.setCacheDisabled", params); err != nil {
		return nil, fmt.Errorf("Network.setCacheDisabled: %w", err)
	}
	return &pb.SetCacheDisabledResponse{}, nil
}

func (s *Server) SetExtraHTTPHeaders(ctx context.Context, req *pb.SetExtraHTTPHeadersRequest) (*pb.SetExtraHTTPHeadersResponse, error) {
	headers := map[string]interface{}{}
	for k, v := range req.Headers {
		headers[k] = v
	}
	params := map[string]interface{}{
		"headers": headers,
	}
	if _, err := s.client.Send(ctx, "Network.setExtraHTTPHeaders", params); err != nil {
		return nil, fmt.Errorf("Network.setExtraHTTPHeaders: %w", err)
	}
	return &pb.SetExtraHTTPHeadersResponse{}, nil
}

func (s *Server) GetResponseBody(ctx context.Context, req *pb.GetResponseBodyRequest) (*pb.GetResponseBodyResponse, error) {
	params := map[string]interface{}{
		"requestId": req.RequestId,
	}
	result, err := s.client.Send(ctx, "Network.getResponseBody", params)
	if err != nil {
		return nil, fmt.Errorf("Network.getResponseBody: %w", err)
	}
	var resp struct {
		Body          string `json:"body"`
		Base64Encoded bool   `json:"base64Encoded"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Network.getResponseBody: unmarshal: %w", err)
	}
	return &pb.GetResponseBodyResponse{
		Body:          resp.Body,
		Base64Encoded: resp.Base64Encoded,
	}, nil
}

func (s *Server) GetCookies(ctx context.Context, req *pb.GetCookiesRequest) (*pb.GetCookiesResponse, error) {
	params := map[string]interface{}{}
	if len(req.Urls) > 0 {
		params["urls"] = req.Urls
	}
	var result json.RawMessage
	var err error
	if len(params) > 0 {
		result, err = s.client.Send(ctx, "Network.getCookies", params)
	} else {
		result, err = s.client.Send(ctx, "Network.getCookies", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Network.getCookies: %w", err)
	}
	var resp struct {
		Cookies []cdpCookie `json:"cookies"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Network.getCookies: unmarshal: %w", err)
	}
	cookies := make([]*pb.Cookie, len(resp.Cookies))
	for i, c := range resp.Cookies {
		cookies[i] = c.toProto()
	}
	return &pb.GetCookiesResponse{Cookies: cookies}, nil
}

func (s *Server) SetCookie(ctx context.Context, req *pb.SetCookieRequest) (*pb.SetCookieResponse, error) {
	params := map[string]interface{}{
		"name":  req.Name,
		"value": req.Value,
	}
	if req.Url != "" {
		params["url"] = req.Url
	}
	if req.Domain != "" {
		params["domain"] = req.Domain
	}
	if req.Path != "" {
		params["path"] = req.Path
	}
	if req.Secure {
		params["secure"] = true
	}
	if req.HttpOnly {
		params["httpOnly"] = true
	}
	if req.SameSite != "" {
		params["sameSite"] = req.SameSite
	}
	if req.Expires != 0 {
		params["expires"] = req.Expires
	}
	if req.Priority != "" {
		params["priority"] = req.Priority
	}
	if req.SourceScheme != "" {
		params["sourceScheme"] = req.SourceScheme
	}
	if req.SourcePort != 0 {
		params["sourcePort"] = req.SourcePort
	}
	result, err := s.client.Send(ctx, "Network.setCookie", params)
	if err != nil {
		return nil, fmt.Errorf("Network.setCookie: %w", err)
	}
	var resp struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Network.setCookie: unmarshal: %w", err)
	}
	return &pb.SetCookieResponse{Success: resp.Success}, nil
}

func (s *Server) DeleteCookies(ctx context.Context, req *pb.DeleteCookiesRequest) (*pb.DeleteCookiesResponse, error) {
	params := map[string]interface{}{
		"name": req.Name,
	}
	if req.Url != "" {
		params["url"] = req.Url
	}
	if req.Domain != "" {
		params["domain"] = req.Domain
	}
	if req.Path != "" {
		params["path"] = req.Path
	}
	if _, err := s.client.Send(ctx, "Network.deleteCookies", params); err != nil {
		return nil, fmt.Errorf("Network.deleteCookies: %w", err)
	}
	return &pb.DeleteCookiesResponse{}, nil
}

func (s *Server) ClearBrowserCookies(ctx context.Context, req *pb.ClearBrowserCookiesRequest) (*pb.ClearBrowserCookiesResponse, error) {
	if _, err := s.client.Send(ctx, "Network.clearBrowserCookies", nil); err != nil {
		return nil, fmt.Errorf("Network.clearBrowserCookies: %w", err)
	}
	return &pb.ClearBrowserCookiesResponse{}, nil
}

func (s *Server) ClearBrowserCache(ctx context.Context, req *pb.ClearBrowserCacheRequest) (*pb.ClearBrowserCacheResponse, error) {
	if _, err := s.client.Send(ctx, "Network.clearBrowserCache", nil); err != nil {
		return nil, fmt.Errorf("Network.clearBrowserCache: %w", err)
	}
	return &pb.ClearBrowserCacheResponse{}, nil
}

func (s *Server) EmulateNetworkConditions(ctx context.Context, req *pb.EmulateNetworkConditionsRequest) (*pb.EmulateNetworkConditionsResponse, error) {
	params := map[string]interface{}{
		"offline":            req.Offline,
		"latency":            req.Latency,
		"downloadThroughput": req.DownloadThroughput,
		"uploadThroughput":   req.UploadThroughput,
	}
	if req.ConnectionType != "" {
		params["connectionType"] = req.ConnectionType
	}
	if _, err := s.client.Send(ctx, "Network.emulateNetworkConditions", params); err != nil {
		return nil, fmt.Errorf("Network.emulateNetworkConditions: %w", err)
	}
	return &pb.EmulateNetworkConditionsResponse{}, nil
}

func (s *Server) SetUserAgentOverride(ctx context.Context, req *pb.SetUserAgentOverrideRequest) (*pb.SetUserAgentOverrideResponse, error) {
	params := map[string]interface{}{
		"userAgent": req.UserAgent,
	}
	if req.AcceptLanguage != "" {
		params["acceptLanguage"] = req.AcceptLanguage
	}
	if req.Platform != "" {
		params["platform"] = req.Platform
	}
	if _, err := s.client.Send(ctx, "Network.setUserAgentOverride", params); err != nil {
		return nil, fmt.Errorf("Network.setUserAgentOverride: %w", err)
	}
	return &pb.SetUserAgentOverrideResponse{}, nil
}

func (s *Server) GetCertificate(ctx context.Context, req *pb.GetCertificateRequest) (*pb.GetCertificateResponse, error) {
	params := map[string]interface{}{
		"origin": req.Origin,
	}
	result, err := s.client.Send(ctx, "Network.getCertificate", params)
	if err != nil {
		return nil, fmt.Errorf("Network.getCertificate: %w", err)
	}
	var resp struct {
		TableNames []string `json:"tableNames"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Network.getCertificate: unmarshal: %w", err)
	}
	return &pb.GetCertificateResponse{TableNames: resp.TableNames}, nil
}

func (s *Server) SetRequestInterception(ctx context.Context, req *pb.SetRequestInterceptionRequest) (*pb.SetRequestInterceptionResponse, error) {
	patterns := make([]map[string]interface{}, len(req.Patterns))
	for i, p := range req.Patterns {
		pat := map[string]interface{}{}
		if p.UrlPattern != "" {
			pat["urlPattern"] = p.UrlPattern
		}
		if p.ResourceType != "" {
			pat["resourceType"] = p.ResourceType
		}
		if p.InterceptionStage != "" {
			pat["interceptionStage"] = p.InterceptionStage
		}
		patterns[i] = pat
	}
	params := map[string]interface{}{
		"patterns": patterns,
	}
	if _, err := s.client.Send(ctx, "Network.setRequestInterception", params); err != nil {
		return nil, fmt.Errorf("Network.setRequestInterception: %w", err)
	}
	return &pb.SetRequestInterceptionResponse{}, nil
}

func (s *Server) SearchInResponseBody(ctx context.Context, req *pb.SearchInResponseBodyRequest) (*pb.SearchInResponseBodyResponse, error) {
	params := map[string]interface{}{
		"requestId": req.RequestId,
		"query":     req.Query,
	}
	if req.CaseSensitive {
		params["caseSensitive"] = true
	}
	if req.IsRegex {
		params["isRegex"] = true
	}
	result, err := s.client.Send(ctx, "Network.searchInResponseBody", params)
	if err != nil {
		return nil, fmt.Errorf("Network.searchInResponseBody: %w", err)
	}
	var resp struct {
		Result []struct {
			LineNumber  float64 `json:"lineNumber"`
			LineContent string  `json:"lineContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Network.searchInResponseBody: unmarshal: %w", err)
	}
	matches := make([]*pb.SearchMatch, len(resp.Result))
	for i, m := range resp.Result {
		matches[i] = &pb.SearchMatch{
			LineNumber:  m.LineNumber,
			LineContent: m.LineContent,
		}
	}
	return &pb.SearchInResponseBodyResponse{Result: matches}, nil
}

func (s *Server) SetBlockedURLs(ctx context.Context, req *pb.SetBlockedURLsRequest) (*pb.SetBlockedURLsResponse, error) {
	params := map[string]interface{}{
		"urls": req.Urls,
	}
	if _, err := s.client.Send(ctx, "Network.setBlockedURLs", params); err != nil {
		return nil, fmt.Errorf("Network.setBlockedURLs: %w", err)
	}
	return &pb.SetBlockedURLsResponse{}, nil
}

func (s *Server) GetSecurityIsolationStatus(ctx context.Context, req *pb.GetSecurityIsolationStatusRequest) (*pb.GetSecurityIsolationStatusResponse, error) {
	params := map[string]interface{}{}
	if req.FrameId != "" {
		params["frameId"] = req.FrameId
	}
	var result json.RawMessage
	var err error
	if len(params) > 0 {
		result, err = s.client.Send(ctx, "Network.getSecurityIsolationStatus", params)
	} else {
		result, err = s.client.Send(ctx, "Network.getSecurityIsolationStatus", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Network.getSecurityIsolationStatus: %w", err)
	}
	// The response status is complex; return as JSON string.
	return &pb.GetSecurityIsolationStatusResponse{Status: string(result)}, nil
}

func (s *Server) LoadNetworkResource(ctx context.Context, req *pb.LoadNetworkResourceRequest) (*pb.LoadNetworkResourceResponse, error) {
	params := map[string]interface{}{
		"url": req.Url,
	}
	if req.FrameId != "" {
		params["frameId"] = req.FrameId
	}
	if req.Options != nil {
		opts := map[string]interface{}{
			"disableCache":       req.Options.DisableCache,
			"includeCredentials": req.Options.IncludeCredentials,
		}
		params["options"] = opts
	}
	result, err := s.client.Send(ctx, "Network.loadNetworkResource", params)
	if err != nil {
		return nil, fmt.Errorf("Network.loadNetworkResource: %w", err)
	}
	var resp struct {
		Resource struct {
			Success        bool              `json:"success"`
			NetError       float64           `json:"netError"`
			NetErrorName   string            `json:"netErrorName"`
			HTTPStatusCode float64           `json:"httpStatusCode"`
			Stream         string            `json:"stream"`
			Headers        map[string]string `json:"headers"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Network.loadNetworkResource: unmarshal: %w", err)
	}
	return &pb.LoadNetworkResourceResponse{
		Resource: &pb.LoadNetworkResourcePageResult{
			Success:        resp.Resource.Success,
			NetError:       resp.Resource.NetError,
			NetErrorName:   resp.Resource.NetErrorName,
			HttpStatusCode: resp.Resource.HTTPStatusCode,
			Stream:         resp.Resource.Stream,
			Headers:        resp.Resource.Headers,
		},
	}, nil
}

// SubscribeEvents streams CDP Network events to the gRPC client.
func (s *Server) SubscribeEvents(req *pb.SubscribeNetworkEventsRequest, stream pb.NetworkService_SubscribeEventsServer) error {
	eventCh := make(chan *pb.NetworkEvent, 128)
	ctx := stream.Context()

	networkEvents := []string{
		"Network.requestWillBeSent",
		"Network.responseReceived",
		"Network.dataReceived",
		"Network.loadingFinished",
		"Network.loadingFailed",
		"Network.requestServedFromCache",
	}

	unregisters := make([]func(), len(networkEvents))
	for i, method := range networkEvents {
		method := method
		unregisters[i] = s.client.On(method, func(m string, params json.RawMessage, sessionID string) {
			if req.SessionId != "" && sessionID != req.SessionId {
				return
			}
			evt := convertNetworkEvent(method, params)
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

// --- CDP JSON types for deserialization ---

type cdpCookie struct {
	Name         string  `json:"name"`
	Value        string  `json:"value"`
	Domain       string  `json:"domain"`
	Path         string  `json:"path"`
	Expires      float64 `json:"expires"`
	Size         int32   `json:"size"`
	HTTPOnly     bool    `json:"httpOnly"`
	Secure       bool    `json:"secure"`
	Session      bool    `json:"session"`
	SameSite     string  `json:"sameSite"`
	Priority     string  `json:"priority"`
	SourceScheme string  `json:"sourceScheme"`
	SourcePort   int32   `json:"sourcePort"`
}

func (c *cdpCookie) toProto() *pb.Cookie {
	return &pb.Cookie{
		Name:         c.Name,
		Value:        c.Value,
		Domain:       c.Domain,
		Path:         c.Path,
		Expires:      c.Expires,
		Size:         c.Size,
		HttpOnly:     c.HTTPOnly,
		Secure:       c.Secure,
		Session:      c.Session,
		SameSite:     c.SameSite,
		Priority:     c.Priority,
		SourceScheme: c.SourceScheme,
		SourcePort:   c.SourcePort,
	}
}

type cdpRequest struct {
	URL              string            `json:"url"`
	URLFragment      string            `json:"urlFragment"`
	Method           string            `json:"method"`
	Headers          map[string]string `json:"headers"`
	PostData         string            `json:"postData"`
	HasPostData      bool              `json:"hasPostData"`
	MixedContentType string            `json:"mixedContentType"`
	InitialPriority  string            `json:"initialPriority"`
	ReferrerPolicy   string            `json:"referrerPolicy"`
	IsLinkPreload    bool              `json:"isLinkPreload"`
	IsSameSite       bool              `json:"isSameSite"`
}

func (r *cdpRequest) toProto() *pb.Request {
	if r == nil {
		return nil
	}
	return &pb.Request{
		Url:              r.URL,
		UrlFragment:      r.URLFragment,
		Method:           r.Method,
		Headers:          r.Headers,
		PostData:         r.PostData,
		HasPostData:      r.HasPostData,
		MixedContentType: r.MixedContentType,
		InitialPriority:  r.InitialPriority,
		ReferrerPolicy:   r.ReferrerPolicy,
		IsLinkPreload:    r.IsLinkPreload,
		IsSameSite:       r.IsSameSite,
	}
}

type cdpResourceTiming struct {
	RequestTime         float64 `json:"requestTime"`
	ProxyStart          float64 `json:"proxyStart"`
	ProxyEnd            float64 `json:"proxyEnd"`
	DNSStart            float64 `json:"dnsStart"`
	DNSEnd              float64 `json:"dnsEnd"`
	ConnectStart        float64 `json:"connectStart"`
	ConnectEnd          float64 `json:"connectEnd"`
	SSLStart            float64 `json:"sslStart"`
	SSLEnd              float64 `json:"sslEnd"`
	SendStart           float64 `json:"sendStart"`
	SendEnd             float64 `json:"sendEnd"`
	ReceiveHeadersStart float64 `json:"receiveHeadersStart"`
	ReceiveHeadersEnd   float64 `json:"receiveHeadersEnd"`
}

func (t *cdpResourceTiming) toProto() *pb.ResourceTiming {
	if t == nil {
		return nil
	}
	return &pb.ResourceTiming{
		RequestTime:         t.RequestTime,
		ProxyStart:          t.ProxyStart,
		ProxyEnd:            t.ProxyEnd,
		DnsStart:            t.DNSStart,
		DnsEnd:              t.DNSEnd,
		ConnectStart:        t.ConnectStart,
		ConnectEnd:          t.ConnectEnd,
		SslStart:            t.SSLStart,
		SslEnd:              t.SSLEnd,
		SendStart:           t.SendStart,
		SendEnd:             t.SendEnd,
		ReceiveHeadersStart: t.ReceiveHeadersStart,
		ReceiveHeadersEnd:   t.ReceiveHeadersEnd,
	}
}

type cdpSecurityDetails struct {
	Protocol                          string   `json:"protocol"`
	KeyExchange                       string   `json:"keyExchange"`
	KeyExchangeGroup                  string   `json:"keyExchangeGroup"`
	Cipher                            string   `json:"cipher"`
	Mac                               string   `json:"mac"`
	CertificateID                     int32    `json:"certificateId"`
	SubjectName                       string   `json:"subjectName"`
	SanList                           []string `json:"sanList"`
	Issuer                            string   `json:"issuer"`
	ValidFrom                         float64  `json:"validFrom"`
	ValidTo                           float64  `json:"validTo"`
	CertificateTransparencyCompliance string   `json:"certificateTransparencyCompliance"`
	ServerSignatureAlgorithm          string   `json:"serverSignatureAlgorithm"`
	EncryptedClientHello              bool     `json:"encryptedClientHello"`
}

func (sd *cdpSecurityDetails) toProto() *pb.SecurityDetails {
	if sd == nil {
		return nil
	}
	return &pb.SecurityDetails{
		Protocol:                          sd.Protocol,
		KeyExchange:                       sd.KeyExchange,
		KeyExchangeGroup:                  sd.KeyExchangeGroup,
		Cipher:                            sd.Cipher,
		Mac:                               sd.Mac,
		CertificateId:                     sd.CertificateID,
		SubjectName:                       sd.SubjectName,
		SanList:                           sd.SanList,
		Issuer:                            sd.Issuer,
		ValidFrom:                         sd.ValidFrom,
		ValidTo:                           sd.ValidTo,
		CertificateTransparencyCompliance: sd.CertificateTransparencyCompliance,
		ServerSignatureAlgorithm:          sd.ServerSignatureAlgorithm,
		EncryptedClientHello:              sd.EncryptedClientHello,
	}
}

type cdpResponse struct {
	URL               string              `json:"url"`
	Status            int32               `json:"status"`
	StatusText        string              `json:"statusText"`
	Headers           map[string]string   `json:"headers"`
	MimeType          string              `json:"mimeType"`
	Charset           string              `json:"charset"`
	RequestHeaders    map[string]string   `json:"requestHeaders"`
	ConnectionReused  bool                `json:"connectionReused"`
	ConnectionID      float64             `json:"connectionId"`
	RemoteIPAddress   string              `json:"remoteIPAddress"`
	RemotePort        int32               `json:"remotePort"`
	FromDiskCache     bool                `json:"fromDiskCache"`
	FromServiceWorker bool                `json:"fromServiceWorker"`
	FromPrefetchCache bool                `json:"fromPrefetchCache"`
	EncodedDataLength float64             `json:"encodedDataLength"`
	Timing            *cdpResourceTiming  `json:"timing"`
	Protocol          string              `json:"protocol"`
	SecurityState     string              `json:"securityState"`
	SecurityDetails   *cdpSecurityDetails `json:"securityDetails"`
}

func (r *cdpResponse) toProto() *pb.Response {
	if r == nil {
		return nil
	}
	return &pb.Response{
		Url:               r.URL,
		Status:            r.Status,
		StatusText:        r.StatusText,
		Headers:           r.Headers,
		MimeType:          r.MimeType,
		Charset:           r.Charset,
		RequestHeaders:    r.RequestHeaders,
		ConnectionReused:  r.ConnectionReused,
		ConnectionId:      r.ConnectionID,
		RemoteIpAddress:   r.RemoteIPAddress,
		RemotePort:        r.RemotePort,
		FromDiskCache:     r.FromDiskCache,
		FromServiceWorker: r.FromServiceWorker,
		FromPrefetchCache: r.FromPrefetchCache,
		EncodedDataLength: r.EncodedDataLength,
		Timing:            r.Timing.toProto(),
		Protocol:          r.Protocol,
		SecurityState:     r.SecurityState,
		SecurityDetails:   r.SecurityDetails.toProto(),
	}
}

// --- Event conversion ---

func convertNetworkEvent(method string, params json.RawMessage) *pb.NetworkEvent {
	switch method {
	case "Network.requestWillBeSent":
		var d struct {
			RequestID        string       `json:"requestId"`
			LoaderID         string       `json:"loaderId"`
			DocumentURL      string       `json:"documentURL"`
			Request          cdpRequest   `json:"request"`
			Timestamp        float64      `json:"timestamp"`
			WallTime         float64      `json:"wallTime"`
			Initiator        struct {
				Type string `json:"type"`
			} `json:"initiator"`
			RedirectResponse *cdpResponse `json:"redirectResponse"`
			Type             string       `json:"type"`
			FrameID          string       `json:"frameId"`
			HasUserGesture   bool         `json:"hasUserGesture"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.NetworkEvent{Event: &pb.NetworkEvent_RequestWillBeSent{
			RequestWillBeSent: &pb.RequestWillBeSentEvent{
				RequestId:        d.RequestID,
				LoaderId:         d.LoaderID,
				DocumentUrl:      d.DocumentURL,
				Request:          d.Request.toProto(),
				Timestamp:        d.Timestamp,
				WallTime:         d.WallTime,
				InitiatorType:    d.Initiator.Type,
				RedirectResponse: d.RedirectResponse.toProto(),
				Type:             d.Type,
				FrameId:          d.FrameID,
				HasUserGesture:   d.HasUserGesture,
			},
		}}

	case "Network.responseReceived":
		var d struct {
			RequestID    string      `json:"requestId"`
			LoaderID     string      `json:"loaderId"`
			Timestamp    float64     `json:"timestamp"`
			Type         string      `json:"type"`
			Response     cdpResponse `json:"response"`
			HasExtraInfo bool        `json:"hasExtraInfo"`
			FrameID      string      `json:"frameId"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.NetworkEvent{Event: &pb.NetworkEvent_ResponseReceived{
			ResponseReceived: &pb.ResponseReceivedEvent{
				RequestId:    d.RequestID,
				LoaderId:     d.LoaderID,
				Timestamp:    d.Timestamp,
				Type:         d.Type,
				Response:     d.Response.toProto(),
				HasExtraInfo: d.HasExtraInfo,
				FrameId:      d.FrameID,
			},
		}}

	case "Network.dataReceived":
		var d struct {
			RequestID         string  `json:"requestId"`
			Timestamp         float64 `json:"timestamp"`
			DataLength        int32   `json:"dataLength"`
			EncodedDataLength int32   `json:"encodedDataLength"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.NetworkEvent{Event: &pb.NetworkEvent_DataReceived{
			DataReceived: &pb.DataReceivedEvent{
				RequestId:         d.RequestID,
				Timestamp:         d.Timestamp,
				DataLength:        d.DataLength,
				EncodedDataLength: d.EncodedDataLength,
			},
		}}

	case "Network.loadingFinished":
		var d struct {
			RequestID         string  `json:"requestId"`
			Timestamp         float64 `json:"timestamp"`
			EncodedDataLength float64 `json:"encodedDataLength"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.NetworkEvent{Event: &pb.NetworkEvent_LoadingFinished{
			LoadingFinished: &pb.LoadingFinishedEvent{
				RequestId:         d.RequestID,
				Timestamp:         d.Timestamp,
				EncodedDataLength: d.EncodedDataLength,
			},
		}}

	case "Network.loadingFailed":
		var d struct {
			RequestID     string  `json:"requestId"`
			Timestamp     float64 `json:"timestamp"`
			Type          string  `json:"type"`
			ErrorText     string  `json:"errorText"`
			Canceled      bool    `json:"canceled"`
			BlockedReason string  `json:"blockedReason"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.NetworkEvent{Event: &pb.NetworkEvent_LoadingFailed{
			LoadingFailed: &pb.LoadingFailedEvent{
				RequestId:     d.RequestID,
				Timestamp:     d.Timestamp,
				Type:          d.Type,
				ErrorText:     d.ErrorText,
				Canceled:      d.Canceled,
				BlockedReason: d.BlockedReason,
			},
		}}

	case "Network.requestServedFromCache":
		var d struct {
			RequestID string `json:"requestId"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.NetworkEvent{Event: &pb.NetworkEvent_RequestServedFromCache{
			RequestServedFromCache: &pb.RequestServedFromCacheEvent{
				RequestId: d.RequestID,
			},
		}}
	}
	return nil
}
