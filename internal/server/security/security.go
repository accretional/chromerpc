// Package security implements the gRPC SecurityService by bridging to CDP.
package security

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/security"
)

type Server struct {
	pb.UnimplementedSecurityServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.client.Send(ctx, "Security.enable", nil); err != nil {
		return nil, fmt.Errorf("Security.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "Security.disable", nil); err != nil {
		return nil, fmt.Errorf("Security.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) SetIgnoreCertificateErrors(ctx context.Context, req *pb.SetIgnoreCertificateErrorsRequest) (*pb.SetIgnoreCertificateErrorsResponse, error) {
	params := map[string]interface{}{"ignore": req.Ignore}
	if _, err := s.client.Send(ctx, "Security.setIgnoreCertificateErrors", params); err != nil {
		return nil, fmt.Errorf("Security.setIgnoreCertificateErrors: %w", err)
	}
	return &pb.SetIgnoreCertificateErrorsResponse{}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeEventsRequest, stream pb.SecurityService_SubscribeEventsServer) error {
	ch := make(chan *pb.SecurityEvent, 64)
	defer close(ch)

	unsubscribe := s.client.On("Security.visibleSecurityStateChanged", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			VisibleSecurityState cdpVisibleSecurityState `json:"visibleSecurityState"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		ch <- &pb.SecurityEvent{
			Event: &pb.SecurityEvent_VisibleSecurityStateChanged{
				VisibleSecurityStateChanged: &pb.VisibleSecurityStateChangedEvent{
					VisibleSecurityState: raw.VisibleSecurityState.toProto(),
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

// --- internal helpers ---

type cdpVisibleSecurityState struct {
	SecurityState         string                      `json:"securityState"`
	CertificateSecurityState *cdpCertificateSecurityState `json:"certificateSecurityState"`
	SafetyTipInfo         *cdpSafetyTipInfo            `json:"safetyTipInfo"`
	SecurityStateIssueIDs []string                    `json:"securityStateIssueIds"`
}

func (s *cdpVisibleSecurityState) toProto() *pb.VisibleSecurityState {
	state := &pb.VisibleSecurityState{
		SecurityState:         s.SecurityState,
		SecurityStateIssueIds: s.SecurityStateIssueIDs,
	}
	if s.CertificateSecurityState != nil {
		state.CertificateSecurityState = s.CertificateSecurityState.toProto()
	}
	if s.SafetyTipInfo != nil {
		state.SafetyTipInfo = &pb.SafetyTipInfo{
			SafetyTipStatus: s.SafetyTipInfo.SafetyTipStatus,
			SafeUrl:         s.SafetyTipInfo.SafeURL,
		}
	}
	return state
}

type cdpCertificateSecurityState struct {
	Protocol                     string   `json:"protocol"`
	KeyExchange                  string   `json:"keyExchange"`
	KeyExchangeGroup             string   `json:"keyExchangeGroup"`
	Cipher                       string   `json:"cipher"`
	Mac                          string   `json:"mac"`
	Certificate                  []string `json:"certificate"`
	SubjectName                  string   `json:"subjectName"`
	Issuer                       string   `json:"issuer"`
	ValidFrom                    float64  `json:"validFrom"`
	ValidTo                      float64  `json:"validTo"`
	CertificateNetworkError      string   `json:"certificateNetworkError"`
	CertificateHasWeakSignature  bool     `json:"certificateHasWeakSignature"`
	CertificateHasSHA1Signature  bool     `json:"certificateHasSha1Signature"`
	ModernSSL                    bool     `json:"modernSSL"`
	ObsoleteSSLProtocol          bool     `json:"obsoleteSslProtocol"`
	ObsoleteSSLKeyExchange       bool     `json:"obsoleteSslKeyExchange"`
	ObsoleteSSLCipher            bool     `json:"obsoleteSslCipher"`
	ObsoleteSSLSignature         bool     `json:"obsoleteSslSignature"`
}

func (c *cdpCertificateSecurityState) toProto() *pb.CertificateSecurityState {
	return &pb.CertificateSecurityState{
		Protocol:                     c.Protocol,
		KeyExchange:                  c.KeyExchange,
		KeyExchangeGroup:             c.KeyExchangeGroup,
		Cipher:                       c.Cipher,
		Mac:                          c.Mac,
		Certificate:                  c.Certificate,
		SubjectName:                  c.SubjectName,
		Issuer:                       c.Issuer,
		ValidFrom:                    c.ValidFrom,
		ValidTo:                      c.ValidTo,
		CertificateNetworkError:      c.CertificateNetworkError,
		CertificateHasWeakSignature:  c.CertificateHasWeakSignature,
		CertificateHasSha1Signature:  c.CertificateHasSHA1Signature,
		ModernSsl:                    c.ModernSSL,
		ObsoleteSslProtocol:          c.ObsoleteSSLProtocol,
		ObsoleteSslKeyExchange:       c.ObsoleteSSLKeyExchange,
		ObsoleteSslCipher:            c.ObsoleteSSLCipher,
		ObsoleteSslSignature:         c.ObsoleteSSLSignature,
	}
}

type cdpSafetyTipInfo struct {
	SafetyTipStatus string `json:"safetyTipStatus"`
	SafeURL         string `json:"safeUrl"`
}
