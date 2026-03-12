// Package input implements the gRPC InputService by bridging to CDP.
package input

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/input"
)

type Server struct {
	pb.UnimplementedInputServiceServer
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


func (s *Server) DispatchKeyEvent(ctx context.Context, req *pb.DispatchKeyEventRequest) (*pb.DispatchKeyEventResponse, error) {
	params := map[string]interface{}{
		"type": req.Type,
	}
	if req.Modifiers != 0 {
		params["modifiers"] = req.Modifiers
	}
	if req.Timestamp != 0 {
		params["timestamp"] = req.Timestamp
	}
	if req.Text != "" {
		params["text"] = req.Text
	}
	if req.UnmodifiedText != "" {
		params["unmodifiedText"] = req.UnmodifiedText
	}
	if req.KeyIdentifier != "" {
		params["keyIdentifier"] = req.KeyIdentifier
	}
	if req.Code != "" {
		params["code"] = req.Code
	}
	if req.Key != "" {
		params["key"] = req.Key
	}
	if req.WindowsVirtualKeyCode != 0 {
		params["windowsVirtualKeyCode"] = req.WindowsVirtualKeyCode
	}
	if req.NativeVirtualKeyCode != 0 {
		params["nativeVirtualKeyCode"] = req.NativeVirtualKeyCode
	}
	if req.AutoRepeat {
		params["autoRepeat"] = true
	}
	if req.IsKeypad {
		params["isKeypad"] = true
	}
	if req.IsSystemKey {
		params["isSystemKey"] = true
	}
	if len(req.Commands) > 0 {
		params["commands"] = req.Commands
	}
	if req.Location != 0 {
		params["location"] = req.Location
	}
	if _, err := s.send(ctx, req.SessionId, "Input.dispatchKeyEvent", params); err != nil {
		return nil, fmt.Errorf("Input.dispatchKeyEvent: %w", err)
	}
	return &pb.DispatchKeyEventResponse{}, nil
}

func (s *Server) DispatchMouseEvent(ctx context.Context, req *pb.DispatchMouseEventRequest) (*pb.DispatchMouseEventResponse, error) {
	params := map[string]interface{}{
		"type": req.Type,
		"x":    req.X,
		"y":    req.Y,
	}
	if req.Modifiers != 0 {
		params["modifiers"] = req.Modifiers
	}
	if req.Timestamp != 0 {
		params["timestamp"] = req.Timestamp
	}
	if req.Button != "" {
		params["button"] = req.Button
	}
	if req.Buttons != 0 {
		params["buttons"] = req.Buttons
	}
	if req.ClickCount != 0 {
		params["clickCount"] = req.ClickCount
	}
	if req.Force != 0 {
		params["force"] = req.Force
	}
	if req.TangentialPressure != 0 {
		params["tangentialPressure"] = req.TangentialPressure
	}
	if req.TiltX != 0 {
		params["tiltX"] = req.TiltX
	}
	if req.TiltY != 0 {
		params["tiltY"] = req.TiltY
	}
	if req.Twist != 0 {
		params["twist"] = req.Twist
	}
	if req.DeltaX != 0 {
		params["deltaX"] = req.DeltaX
	}
	if req.DeltaY != 0 {
		params["deltaY"] = req.DeltaY
	}
	if req.PointerType != "" {
		params["pointerType"] = req.PointerType
	}
	if _, err := s.send(ctx, req.SessionId, "Input.dispatchMouseEvent", params); err != nil {
		return nil, fmt.Errorf("Input.dispatchMouseEvent: %w", err)
	}
	return &pb.DispatchMouseEventResponse{}, nil
}

func (s *Server) DispatchTouchEvent(ctx context.Context, req *pb.DispatchTouchEventRequest) (*pb.DispatchTouchEventResponse, error) {
	params := map[string]interface{}{
		"type": req.Type,
	}
	touchPoints := make([]map[string]interface{}, len(req.TouchPoints))
	for i, tp := range req.TouchPoints {
		point := map[string]interface{}{
			"x": tp.X,
			"y": tp.Y,
		}
		if tp.RadiusX != 0 {
			point["radiusX"] = tp.RadiusX
		}
		if tp.RadiusY != 0 {
			point["radiusY"] = tp.RadiusY
		}
		if tp.RotationAngle != 0 {
			point["rotationAngle"] = tp.RotationAngle
		}
		if tp.Force != 0 {
			point["force"] = tp.Force
		}
		if tp.Id != 0 {
			point["id"] = tp.Id
		}
		touchPoints[i] = point
	}
	params["touchPoints"] = touchPoints
	if req.Modifiers != 0 {
		params["modifiers"] = req.Modifiers
	}
	if req.Timestamp != 0 {
		params["timestamp"] = req.Timestamp
	}
	if _, err := s.send(ctx, req.SessionId, "Input.dispatchTouchEvent", params); err != nil {
		return nil, fmt.Errorf("Input.dispatchTouchEvent: %w", err)
	}
	return &pb.DispatchTouchEventResponse{}, nil
}

func (s *Server) InsertText(ctx context.Context, req *pb.InsertTextRequest) (*pb.InsertTextResponse, error) {
	params := map[string]interface{}{"text": req.Text}
	if _, err := s.send(ctx, req.SessionId, "Input.insertText", params); err != nil {
		return nil, fmt.Errorf("Input.insertText: %w", err)
	}
	return &pb.InsertTextResponse{}, nil
}

func (s *Server) SetIgnoreInputEvents(ctx context.Context, req *pb.SetIgnoreInputEventsRequest) (*pb.SetIgnoreInputEventsResponse, error) {
	params := map[string]interface{}{"ignore": req.Ignore}
	if _, err := s.send(ctx, req.SessionId, "Input.setIgnoreInputEvents", params); err != nil {
		return nil, fmt.Errorf("Input.setIgnoreInputEvents: %w", err)
	}
	return &pb.SetIgnoreInputEventsResponse{}, nil
}

func (s *Server) DispatchDragEvent(ctx context.Context, req *pb.DispatchDragEventRequest) (*pb.DispatchDragEventResponse, error) {
	params := map[string]interface{}{
		"type": req.Type,
		"x":    req.X,
		"y":    req.Y,
	}
	if req.Data != nil {
		data := map[string]interface{}{
			"dragOperationsMask": req.Data.DragOperationsMask,
		}
		items := make([]map[string]interface{}, len(req.Data.Items))
		for i, item := range req.Data.Items {
			it := map[string]interface{}{"mimeType": item.MimeType, "data": item.Data}
			if item.Title != "" {
				it["title"] = item.Title
			}
			if item.BaseUrl != "" {
				it["baseURL"] = item.BaseUrl
			}
			items[i] = it
		}
		data["items"] = items
		if len(req.Data.Files) > 0 {
			data["files"] = req.Data.Files
		}
		params["data"] = data
	}
	if req.Modifiers != 0 {
		params["modifiers"] = req.Modifiers
	}
	if _, err := s.send(ctx, req.SessionId, "Input.dispatchDragEvent", params); err != nil {
		return nil, fmt.Errorf("Input.dispatchDragEvent: %w", err)
	}
	return &pb.DispatchDragEventResponse{}, nil
}

func (s *Server) SynthesizePinchGesture(ctx context.Context, req *pb.SynthesizePinchGestureRequest) (*pb.SynthesizePinchGestureResponse, error) {
	params := map[string]interface{}{
		"x":           req.X,
		"y":           req.Y,
		"scaleFactor": req.ScaleFactor,
	}
	if req.RelativeSpeed != 0 {
		params["relativeSpeed"] = req.RelativeSpeed
	}
	if req.GestureSourceType != "" {
		params["gestureSourceType"] = req.GestureSourceType
	}
	if _, err := s.send(ctx, req.SessionId, "Input.synthesizePinchGesture", params); err != nil {
		return nil, fmt.Errorf("Input.synthesizePinchGesture: %w", err)
	}
	return &pb.SynthesizePinchGestureResponse{}, nil
}

func (s *Server) SynthesizeScrollGesture(ctx context.Context, req *pb.SynthesizeScrollGestureRequest) (*pb.SynthesizeScrollGestureResponse, error) {
	params := map[string]interface{}{
		"x": req.X,
		"y": req.Y,
	}
	if req.XDistance != 0 {
		params["xDistance"] = req.XDistance
	}
	if req.YDistance != 0 {
		params["yDistance"] = req.YDistance
	}
	if req.XOverscroll != 0 {
		params["xOverscroll"] = req.XOverscroll
	}
	if req.YOverscroll != 0 {
		params["yOverscroll"] = req.YOverscroll
	}
	if req.PreventFling {
		params["preventFling"] = true
	}
	if req.Speed != 0 {
		params["speed"] = req.Speed
	}
	if req.GestureSourceType != "" {
		params["gestureSourceType"] = req.GestureSourceType
	}
	if req.RepeatCount != 0 {
		params["repeatCount"] = req.RepeatCount
	}
	if req.RepeatDelayMs != 0 {
		params["repeatDelayMs"] = req.RepeatDelayMs
	}
	if req.InteractionMarkerName != "" {
		params["interactionMarkerName"] = req.InteractionMarkerName
	}
	if _, err := s.send(ctx, req.SessionId, "Input.synthesizeScrollGesture", params); err != nil {
		return nil, fmt.Errorf("Input.synthesizeScrollGesture: %w", err)
	}
	return &pb.SynthesizeScrollGestureResponse{}, nil
}

func (s *Server) SynthesizeTapGesture(ctx context.Context, req *pb.SynthesizeTapGestureRequest) (*pb.SynthesizeTapGestureResponse, error) {
	params := map[string]interface{}{
		"x": req.X,
		"y": req.Y,
	}
	if req.Duration != 0 {
		params["duration"] = req.Duration
	}
	if req.TapCount != 0 {
		params["tapCount"] = req.TapCount
	}
	if req.GestureSourceType != "" {
		params["gestureSourceType"] = req.GestureSourceType
	}
	if _, err := s.send(ctx, req.SessionId, "Input.synthesizeTapGesture", params); err != nil {
		return nil, fmt.Errorf("Input.synthesizeTapGesture: %w", err)
	}
	return &pb.SynthesizeTapGestureResponse{}, nil
}

func (s *Server) CancelDragging(ctx context.Context, req *pb.CancelDraggingRequest) (*pb.CancelDraggingResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Input.cancelDragging", nil); err != nil {
		return nil, fmt.Errorf("Input.cancelDragging: %w", err)
	}
	return &pb.CancelDraggingResponse{}, nil
}

func (s *Server) ImeSetComposition(ctx context.Context, req *pb.ImeSetCompositionRequest) (*pb.ImeSetCompositionResponse, error) {
	params := map[string]interface{}{
		"text":           req.Text,
		"selectionStart": req.SelectionStart,
		"selectionEnd":   req.SelectionEnd,
	}
	if req.ReplacementStart != 0 {
		params["replacementStart"] = req.ReplacementStart
	}
	if req.ReplacementEnd != 0 {
		params["replacementEnd"] = req.ReplacementEnd
	}
	if _, err := s.send(ctx, req.SessionId, "Input.imeSetComposition", params); err != nil {
		return nil, fmt.Errorf("Input.imeSetComposition: %w", err)
	}
	return &pb.ImeSetCompositionResponse{}, nil
}
