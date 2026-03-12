// Package webauthn implements the gRPC WebAuthnService by bridging to CDP.
package webauthn

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/webauthn"
)

type Server struct {
	pb.UnimplementedWebAuthnServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

// protocolToString converts the proto enum to the CDP string value.
func protocolToString(p pb.AuthenticatorProtocol) string {
	switch p {
	case pb.AuthenticatorProtocol_U2F:
		return "u2f"
	case pb.AuthenticatorProtocol_CTAP2:
		return "ctap2"
	default:
		return ""
	}
}

// transportToString converts the proto enum to the CDP string value.
func transportToString(t pb.AuthenticatorTransport) string {
	switch t {
	case pb.AuthenticatorTransport_USB:
		return "usb"
	case pb.AuthenticatorTransport_NFC:
		return "nfc"
	case pb.AuthenticatorTransport_BLE:
		return "ble"
	case pb.AuthenticatorTransport_CABLE:
		return "cable"
	case pb.AuthenticatorTransport_INTERNAL:
		return "internal"
	default:
		return ""
	}
}

// ctap2VersionToString converts the proto enum to the CDP string value.
func ctap2VersionToString(v pb.Ctap2Version) string {
	switch v {
	case pb.Ctap2Version_CTAP2_0:
		return "ctap2_0"
	case pb.Ctap2Version_CTAP2_1:
		return "ctap2_1"
	default:
		return ""
	}
}

// buildOptionsParams converts VirtualAuthenticatorOptions to a CDP params map.
func buildOptionsParams(opts *pb.VirtualAuthenticatorOptions) map[string]interface{} {
	m := map[string]interface{}{
		"protocol":  protocolToString(opts.Protocol),
		"transport": transportToString(opts.Transport),
	}
	if opts.Ctap2Version != pb.Ctap2Version_CTAP2_VERSION_UNSPECIFIED {
		m["ctap2Version"] = ctap2VersionToString(opts.Ctap2Version)
	}
	if opts.HasResidentKey {
		m["hasResidentKey"] = true
	}
	if opts.HasUserVerification {
		m["hasUserVerification"] = true
	}
	if opts.HasLargeBlob {
		m["hasLargeBlob"] = true
	}
	if opts.HasCredBlob {
		m["hasCredBlob"] = true
	}
	if opts.HasMinPinLength {
		m["hasMinPinLength"] = true
	}
	if opts.HasPrf {
		m["hasPrf"] = true
	}
	if opts.AutomaticPresenceSimulation {
		m["automaticPresenceSimulation"] = true
	}
	if opts.IsUserVerified {
		m["isUserVerified"] = true
	}
	if opts.DefaultBackupEligibility {
		m["defaultBackupEligibility"] = true
	}
	if opts.DefaultBackupState {
		m["defaultBackupState"] = true
	}
	return m
}

// buildCredentialParams converts a Credential to a CDP params map.
func buildCredentialParams(c *pb.Credential) map[string]interface{} {
	m := map[string]interface{}{
		"credentialId":        c.CredentialId,
		"isResidentCredential": c.IsResidentCredential,
		"privateKey":          c.PrivateKey,
		"signCount":           c.SignCount,
	}
	if c.RpId != "" {
		m["rpId"] = c.RpId
	}
	if c.UserHandle != "" {
		m["userHandle"] = c.UserHandle
	}
	if c.LargeBlob != "" {
		m["largeBlob"] = c.LargeBlob
	}
	if c.BackupEligibility {
		m["backupEligibility"] = true
	}
	if c.BackupState {
		m["backupState"] = true
	}
	return m
}

// parseCredential converts a CDP JSON credential to a proto Credential.
func parseCredential(raw json.RawMessage) (*pb.Credential, error) {
	var c struct {
		CredentialId        string `json:"credentialId"`
		IsResidentCredential bool  `json:"isResidentCredential"`
		RpId                string `json:"rpId"`
		PrivateKey          string `json:"privateKey"`
		UserHandle          string `json:"userHandle"`
		SignCount           int32  `json:"signCount"`
		LargeBlob           string `json:"largeBlob"`
		BackupEligibility   bool   `json:"backupEligibility"`
		BackupState         bool   `json:"backupState"`
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	return &pb.Credential{
		CredentialId:        c.CredentialId,
		IsResidentCredential: c.IsResidentCredential,
		RpId:                c.RpId,
		PrivateKey:          c.PrivateKey,
		UserHandle:          c.UserHandle,
		SignCount:           c.SignCount,
		LargeBlob:           c.LargeBlob,
		BackupEligibility:   c.BackupEligibility,
		BackupState:         c.BackupState,
	}, nil
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	var params map[string]interface{}
	if req.EnableUi {
		params = map[string]interface{}{
			"enableUI": req.EnableUi,
		}
	}
	if _, err := s.client.Send(ctx, "WebAuthn.enable", params); err != nil {
		return nil, fmt.Errorf("WebAuthn.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "WebAuthn.disable", nil); err != nil {
		return nil, fmt.Errorf("WebAuthn.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) AddVirtualAuthenticator(ctx context.Context, req *pb.AddVirtualAuthenticatorRequest) (*pb.AddVirtualAuthenticatorResponse, error) {
	params := map[string]interface{}{
		"options": buildOptionsParams(req.Options),
	}
	result, err := s.client.Send(ctx, "WebAuthn.addVirtualAuthenticator", params)
	if err != nil {
		return nil, fmt.Errorf("WebAuthn.addVirtualAuthenticator: %w", err)
	}
	var resp struct {
		AuthenticatorId string `json:"authenticatorId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("WebAuthn.addVirtualAuthenticator: unmarshal: %w", err)
	}
	return &pb.AddVirtualAuthenticatorResponse{AuthenticatorId: resp.AuthenticatorId}, nil
}

func (s *Server) RemoveVirtualAuthenticator(ctx context.Context, req *pb.RemoveVirtualAuthenticatorRequest) (*pb.RemoveVirtualAuthenticatorResponse, error) {
	params := map[string]interface{}{
		"authenticatorId": req.AuthenticatorId,
	}
	if _, err := s.client.Send(ctx, "WebAuthn.removeVirtualAuthenticator", params); err != nil {
		return nil, fmt.Errorf("WebAuthn.removeVirtualAuthenticator: %w", err)
	}
	return &pb.RemoveVirtualAuthenticatorResponse{}, nil
}

func (s *Server) AddCredential(ctx context.Context, req *pb.AddCredentialRequest) (*pb.AddCredentialResponse, error) {
	params := map[string]interface{}{
		"authenticatorId": req.AuthenticatorId,
		"credential":      buildCredentialParams(req.Credential),
	}
	if _, err := s.client.Send(ctx, "WebAuthn.addCredential", params); err != nil {
		return nil, fmt.Errorf("WebAuthn.addCredential: %w", err)
	}
	return &pb.AddCredentialResponse{}, nil
}

func (s *Server) GetCredential(ctx context.Context, req *pb.GetCredentialRequest) (*pb.GetCredentialResponse, error) {
	params := map[string]interface{}{
		"authenticatorId": req.AuthenticatorId,
		"credentialId":    req.CredentialId,
	}
	result, err := s.client.Send(ctx, "WebAuthn.getCredential", params)
	if err != nil {
		return nil, fmt.Errorf("WebAuthn.getCredential: %w", err)
	}
	var resp struct {
		Credential json.RawMessage `json:"credential"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("WebAuthn.getCredential: unmarshal: %w", err)
	}
	cred, err := parseCredential(resp.Credential)
	if err != nil {
		return nil, fmt.Errorf("WebAuthn.getCredential: parse credential: %w", err)
	}
	return &pb.GetCredentialResponse{Credential: cred}, nil
}

func (s *Server) GetCredentials(ctx context.Context, req *pb.GetCredentialsRequest) (*pb.GetCredentialsResponse, error) {
	params := map[string]interface{}{
		"authenticatorId": req.AuthenticatorId,
	}
	result, err := s.client.Send(ctx, "WebAuthn.getCredentials", params)
	if err != nil {
		return nil, fmt.Errorf("WebAuthn.getCredentials: %w", err)
	}
	var resp struct {
		Credentials []json.RawMessage `json:"credentials"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("WebAuthn.getCredentials: unmarshal: %w", err)
	}
	creds := make([]*pb.Credential, len(resp.Credentials))
	for i, raw := range resp.Credentials {
		cred, err := parseCredential(raw)
		if err != nil {
			return nil, fmt.Errorf("WebAuthn.getCredentials: parse credential %d: %w", i, err)
		}
		creds[i] = cred
	}
	return &pb.GetCredentialsResponse{Credentials: creds}, nil
}

func (s *Server) RemoveCredential(ctx context.Context, req *pb.RemoveCredentialRequest) (*pb.RemoveCredentialResponse, error) {
	params := map[string]interface{}{
		"authenticatorId": req.AuthenticatorId,
		"credentialId":    req.CredentialId,
	}
	if _, err := s.client.Send(ctx, "WebAuthn.removeCredential", params); err != nil {
		return nil, fmt.Errorf("WebAuthn.removeCredential: %w", err)
	}
	return &pb.RemoveCredentialResponse{}, nil
}

func (s *Server) ClearCredentials(ctx context.Context, req *pb.ClearCredentialsRequest) (*pb.ClearCredentialsResponse, error) {
	params := map[string]interface{}{
		"authenticatorId": req.AuthenticatorId,
	}
	if _, err := s.client.Send(ctx, "WebAuthn.clearCredentials", params); err != nil {
		return nil, fmt.Errorf("WebAuthn.clearCredentials: %w", err)
	}
	return &pb.ClearCredentialsResponse{}, nil
}

func (s *Server) SetUserVerified(ctx context.Context, req *pb.SetUserVerifiedRequest) (*pb.SetUserVerifiedResponse, error) {
	params := map[string]interface{}{
		"authenticatorId": req.AuthenticatorId,
		"isUserVerified":  req.IsUserVerified,
	}
	if _, err := s.client.Send(ctx, "WebAuthn.setUserVerified", params); err != nil {
		return nil, fmt.Errorf("WebAuthn.setUserVerified: %w", err)
	}
	return &pb.SetUserVerifiedResponse{}, nil
}

func (s *Server) SetAutomaticPresenceSimulation(ctx context.Context, req *pb.SetAutomaticPresenceSimulationRequest) (*pb.SetAutomaticPresenceSimulationResponse, error) {
	params := map[string]interface{}{
		"authenticatorId": req.AuthenticatorId,
		"enabled":         req.Enabled,
	}
	if _, err := s.client.Send(ctx, "WebAuthn.setAutomaticPresenceSimulation", params); err != nil {
		return nil, fmt.Errorf("WebAuthn.setAutomaticPresenceSimulation: %w", err)
	}
	return &pb.SetAutomaticPresenceSimulationResponse{}, nil
}

func (s *Server) SetResponseOverrideBits(ctx context.Context, req *pb.SetResponseOverrideBitsRequest) (*pb.SetResponseOverrideBitsResponse, error) {
	params := map[string]interface{}{
		"authenticatorId": req.AuthenticatorId,
	}
	if req.IsBogusSignature {
		params["isBogusSignature"] = true
	}
	if req.IsBadUv {
		params["isBadUV"] = true
	}
	if req.IsBadUp {
		params["isBadUP"] = true
	}
	if _, err := s.client.Send(ctx, "WebAuthn.setResponseOverrideBits", params); err != nil {
		return nil, fmt.Errorf("WebAuthn.setResponseOverrideBits: %w", err)
	}
	return &pb.SetResponseOverrideBitsResponse{}, nil
}
