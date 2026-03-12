// Package bluetoothemulation implements the gRPC BluetoothEmulationService by bridging to CDP.
package bluetoothemulation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/bluetoothemulation"
)

type Server struct {
	pb.UnimplementedBluetoothEmulationServiceServer
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


// buildManufacturerDataParams converts a slice of ManufacturerData protos to CDP params.
func buildManufacturerDataParams(data []*pb.ManufacturerData) []map[string]interface{} {
	result := make([]map[string]interface{}, len(data))
	for i, d := range data {
		result[i] = map[string]interface{}{
			"key":  d.Key,
			"data": d.Data,
		}
	}
	return result
}

// buildScanRecordParams converts a ScanRecord proto to a CDP params map.
func buildScanRecordParams(sr *pb.ScanRecord) map[string]interface{} {
	m := map[string]interface{}{}
	if sr.Name != nil {
		m["name"] = *sr.Name
	}
	if len(sr.Uuids) > 0 {
		m["uuids"] = sr.Uuids
	}
	if sr.Appearance != nil {
		m["appearance"] = *sr.Appearance
	}
	if len(sr.ManufacturerData) > 0 {
		m["manufacturerData"] = buildManufacturerDataParams(sr.ManufacturerData)
	}
	return m
}

// buildScanEntryParams converts a ScanEntry proto to a CDP params map.
func buildScanEntryParams(entry *pb.ScanEntry) map[string]interface{} {
	m := map[string]interface{}{
		"deviceAddress": entry.DeviceAddress,
		"rssi":          entry.Rssi,
	}
	if entry.ScanRecord != nil {
		m["scanRecord"] = buildScanRecordParams(entry.ScanRecord)
	}
	return m
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	params := map[string]interface{}{
		"state": req.State,
	}
	if _, err := s.send(ctx, req.SessionId, "BluetoothEmulation.enable", params); err != nil {
		return nil, fmt.Errorf("BluetoothEmulation.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "BluetoothEmulation.disable", nil); err != nil {
		return nil, fmt.Errorf("BluetoothEmulation.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) SimulatePreconnectedPeripheral(ctx context.Context, req *pb.SimulatePreconnectedPeripheralRequest) (*pb.SimulatePreconnectedPeripheralResponse, error) {
	params := map[string]interface{}{
		"address": req.Address,
		"name":    req.Name,
	}
	if len(req.ManufacturerData) > 0 {
		params["manufacturerData"] = buildManufacturerDataParams(req.ManufacturerData)
	}
	if len(req.KnownServiceUuids) > 0 {
		params["knownServiceUUIDs"] = req.KnownServiceUuids
	}
	if _, err := s.send(ctx, req.SessionId, "BluetoothEmulation.simulatePreconnectedPeripheral", params); err != nil {
		return nil, fmt.Errorf("BluetoothEmulation.simulatePreconnectedPeripheral: %w", err)
	}
	return &pb.SimulatePreconnectedPeripheralResponse{}, nil
}

func (s *Server) SimulateAdvertisement(ctx context.Context, req *pb.SimulateAdvertisementRequest) (*pb.SimulateAdvertisementResponse, error) {
	params := map[string]interface{}{
		"entry": buildScanEntryParams(req.Entry),
	}
	if _, err := s.send(ctx, req.SessionId, "BluetoothEmulation.simulateAdvertisement", params); err != nil {
		return nil, fmt.Errorf("BluetoothEmulation.simulateAdvertisement: %w", err)
	}
	return &pb.SimulateAdvertisementResponse{}, nil
}
