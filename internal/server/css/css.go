// Package css implements the gRPC CSSService by bridging to CDP over WebSocket.
package css

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/css"
	"google.golang.org/grpc"
)

// Server implements the cdp.css.CSSService gRPC service.
type Server struct {
	pb.UnimplementedCSSServiceServer
	client *cdpclient.Client
}

// New creates a new CSS gRPC server backed by the given CDP client.
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


// --- internal CDP types ---

type cdpSourceRange struct {
	StartLine   int32 `json:"startLine"`
	StartColumn int32 `json:"startColumn"`
	EndLine     int32 `json:"endLine"`
	EndColumn   int32 `json:"endColumn"`
}

func (r *cdpSourceRange) toProto() *pb.SourceRange {
	if r == nil {
		return nil
	}
	return &pb.SourceRange{
		StartLine:   r.StartLine,
		StartColumn: r.StartColumn,
		EndLine:     r.EndLine,
		EndColumn:   r.EndColumn,
	}
}

type cdpCSSProperty struct {
	Name      string          `json:"name"`
	Value     string          `json:"value"`
	Important bool            `json:"important"`
	Implicit  bool            `json:"implicit"`
	Text      string          `json:"text"`
	ParsedOk  bool            `json:"parsedOk"`
	Disabled  bool            `json:"disabled"`
	Range     *cdpSourceRange `json:"range"`
}

func (p *cdpCSSProperty) toProto() *pb.CSSProperty {
	return &pb.CSSProperty{
		Name:      p.Name,
		Value:     p.Value,
		Important: p.Important,
		Implicit:  p.Implicit,
		Text:      p.Text,
		ParsedOk:  p.ParsedOk,
		Disabled:  p.Disabled,
		Range:     p.Range.toProto(),
	}
}

type cdpShorthandEntry struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	Important bool   `json:"important"`
}

func (e *cdpShorthandEntry) toProto() *pb.ShorthandEntry {
	return &pb.ShorthandEntry{
		Name:      e.Name,
		Value:     e.Value,
		Important: e.Important,
	}
}

type cdpCSSStyle struct {
	StyleSheetID    string              `json:"styleSheetId"`
	CSSProperties   []cdpCSSProperty    `json:"cssProperties"`
	ShorthandEntries []cdpShorthandEntry `json:"shorthandEntries"`
	CSSText         string              `json:"cssText"`
	Range           *cdpSourceRange     `json:"range"`
}

func (s *cdpCSSStyle) toProto() *pb.CSSStyle {
	if s == nil {
		return nil
	}
	props := make([]*pb.CSSProperty, len(s.CSSProperties))
	for i := range s.CSSProperties {
		props[i] = s.CSSProperties[i].toProto()
	}
	shorthands := make([]*pb.ShorthandEntry, len(s.ShorthandEntries))
	for i := range s.ShorthandEntries {
		shorthands[i] = s.ShorthandEntries[i].toProto()
	}
	return &pb.CSSStyle{
		StyleSheetId:     s.StyleSheetID,
		CssProperties:    props,
		ShorthandEntries: shorthands,
		CssText:          s.CSSText,
		Range:            s.Range.toProto(),
	}
}

type cdpSelectorEntry struct {
	Value        string          `json:"value"`
	Range        *cdpSourceRange `json:"range"`
	SpecificityA int32           `json:"specificityA"`
	SpecificityB int32           `json:"specificityB"`
	SpecificityC int32           `json:"specificityC"`
}

func (e *cdpSelectorEntry) toProto() *pb.SelectorEntry {
	return &pb.SelectorEntry{
		Value:        e.Value,
		Range:        e.Range.toProto(),
		SpecificityA: e.SpecificityA,
		SpecificityB: e.SpecificityB,
		SpecificityC: e.SpecificityC,
	}
}

type cdpCSSMedia struct {
	Text         string          `json:"text"`
	Source       string          `json:"source"`
	SourceURL    string          `json:"sourceURL"`
	Range        *cdpSourceRange `json:"range"`
	StyleSheetID string          `json:"styleSheetId"`
}

func (m *cdpCSSMedia) toProto() *pb.CSSMedia {
	return &pb.CSSMedia{
		Text:         m.Text,
		Source:       m.Source,
		SourceUrl:    m.SourceURL,
		Range:        m.Range.toProto(),
		StyleSheetId: m.StyleSheetID,
	}
}

type cdpCSSRule struct {
	StyleSheetID string             `json:"styleSheetId"`
	SelectorList *cdpSelectorList   `json:"selectorList"`
	Origin       string             `json:"origin"`
	Style        *cdpCSSStyle       `json:"style"`
	Media        []cdpCSSMedia      `json:"media"`
}

type cdpSelectorList struct {
	Selectors []cdpSelectorEntry `json:"selectors"`
	Text      string             `json:"text"`
}

func (r *cdpCSSRule) toProto() *pb.CSSRule {
	if r == nil {
		return nil
	}
	rule := &pb.CSSRule{
		StyleSheetId: r.StyleSheetID,
		Origin:       r.Origin,
		Style:        r.Style.toProto(),
	}
	if r.SelectorList != nil {
		rule.SelectorText = r.SelectorList.Text
		selectors := make([]*pb.SelectorEntry, len(r.SelectorList.Selectors))
		for i := range r.SelectorList.Selectors {
			selectors[i] = r.SelectorList.Selectors[i].toProto()
		}
		rule.Selectors = selectors
	}
	if len(r.Media) > 0 {
		media := make([]*pb.CSSMedia, len(r.Media))
		for i := range r.Media {
			media[i] = r.Media[i].toProto()
		}
		rule.Media = media
	}
	return rule
}

type cdpRuleMatch struct {
	Rule              cdpCSSRule `json:"rule"`
	MatchingSelectors []int32    `json:"matchingSelectors"`
}

func (m *cdpRuleMatch) toProto() *pb.RuleMatch {
	return &pb.RuleMatch{
		Rule:              m.Rule.toProto(),
		MatchingSelectors: m.MatchingSelectors,
	}
}

type cdpInheritedStyleEntry struct {
	InlineStyle     *cdpCSSStyle   `json:"inlineStyle"`
	MatchedCSSRules []cdpRuleMatch `json:"matchedCSSRules"`
}

func (e *cdpInheritedStyleEntry) toProto() *pb.InheritedStyleEntry {
	entry := &pb.InheritedStyleEntry{
		InlineStyle: e.InlineStyle.toProto(),
	}
	if len(e.MatchedCSSRules) > 0 {
		rules := make([]*pb.RuleMatch, len(e.MatchedCSSRules))
		for i := range e.MatchedCSSRules {
			rules[i] = e.MatchedCSSRules[i].toProto()
		}
		entry.MatchedCssRules = rules
	}
	return entry
}

type cdpStyleSheetHeader struct {
	StyleSheetID  string  `json:"styleSheetId"`
	FrameID       string  `json:"frameId"`
	SourceURL     string  `json:"sourceURL"`
	Origin        string  `json:"origin"`
	Title         string  `json:"title"`
	OwnerNode     string  `json:"ownerNode"`
	Disabled      bool    `json:"disabled"`
	IsInline      bool    `json:"isInline"`
	IsMutable     bool    `json:"isMutable"`
	IsConstructed bool    `json:"isConstructed"`
	StartLine     float64 `json:"startLine"`
	StartColumn   float64 `json:"startColumn"`
	Length        float64 `json:"length"`
	EndLine       float64 `json:"endLine"`
	EndColumn     float64 `json:"endColumn"`
}

func (h *cdpStyleSheetHeader) toProto() *pb.CSSStyleSheetHeader {
	if h == nil {
		return nil
	}
	return &pb.CSSStyleSheetHeader{
		StyleSheetId:  h.StyleSheetID,
		FrameId:       h.FrameID,
		SourceUrl:     h.SourceURL,
		Origin:        h.Origin,
		Title:         h.Title,
		OwnerNode:     h.OwnerNode,
		Disabled:      h.Disabled,
		IsInline:      h.IsInline,
		IsMutable:     h.IsMutable,
		IsConstructed: h.IsConstructed,
		StartLine:     h.StartLine,
		StartColumn:   h.StartColumn,
		Length:        h.Length,
		EndLine:       h.EndLine,
		EndColumn:     h.EndColumn,
	}
}

type cdpRuleUsage struct {
	StyleSheetID string  `json:"styleSheetId"`
	StartOffset  float64 `json:"startOffset"`
	EndOffset    float64 `json:"endOffset"`
	Used         bool    `json:"used"`
}

func (r *cdpRuleUsage) toProto() *pb.RuleUsage {
	return &pb.RuleUsage{
		StyleSheetId: r.StyleSheetID,
		StartOffset:  r.StartOffset,
		EndOffset:    r.EndOffset,
		Used:         r.Used,
	}
}

type cdpPlatformFontUsage struct {
	FamilyName   string  `json:"familyName"`
	IsCustomFont bool    `json:"isCustomFont"`
	GlyphCount   float64 `json:"glyphCount"`
}

func (f *cdpPlatformFontUsage) toProto() *pb.PlatformFontUsage {
	return &pb.PlatformFontUsage{
		FamilyName:   f.FamilyName,
		IsCustomFont: f.IsCustomFont,
		GlyphCount:   f.GlyphCount,
	}
}

// --- RPC implementations ---

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "CSS.enable", nil); err != nil {
		return nil, fmt.Errorf("CSS.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "CSS.disable", nil); err != nil {
		return nil, fmt.Errorf("CSS.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) GetMatchedStylesForNode(ctx context.Context, req *pb.GetMatchedStylesForNodeRequest) (*pb.GetMatchedStylesForNodeResponse, error) {
	params := map[string]interface{}{
		"nodeId": req.NodeId,
	}
	result, err := s.send(ctx, req.SessionId, "CSS.getMatchedStylesForNode", params)
	if err != nil {
		return nil, fmt.Errorf("CSS.getMatchedStylesForNode: %w", err)
	}
	var resp struct {
		InlineStyle     *cdpCSSStyle             `json:"inlineStyle"`
		AttributesStyle *cdpCSSStyle             `json:"attributesStyle"`
		MatchedCSSRules []cdpRuleMatch           `json:"matchedCSSRules"`
		Inherited       []cdpInheritedStyleEntry `json:"inherited"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("CSS.getMatchedStylesForNode: unmarshal: %w", err)
	}
	out := &pb.GetMatchedStylesForNodeResponse{
		InlineStyle:     resp.InlineStyle.toProto(),
		AttributesStyle: resp.AttributesStyle.toProto(),
	}
	if len(resp.MatchedCSSRules) > 0 {
		rules := make([]*pb.RuleMatch, len(resp.MatchedCSSRules))
		for i := range resp.MatchedCSSRules {
			rules[i] = resp.MatchedCSSRules[i].toProto()
		}
		out.MatchedCssRules = rules
	}
	if len(resp.Inherited) > 0 {
		inherited := make([]*pb.InheritedStyleEntry, len(resp.Inherited))
		for i := range resp.Inherited {
			inherited[i] = resp.Inherited[i].toProto()
		}
		out.Inherited = inherited
	}
	return out, nil
}

func (s *Server) GetComputedStyleForNode(ctx context.Context, req *pb.GetComputedStyleForNodeRequest) (*pb.GetComputedStyleForNodeResponse, error) {
	params := map[string]interface{}{
		"nodeId": req.NodeId,
	}
	result, err := s.send(ctx, req.SessionId, "CSS.getComputedStyleForNode", params)
	if err != nil {
		return nil, fmt.Errorf("CSS.getComputedStyleForNode: %w", err)
	}
	var resp struct {
		ComputedStyle []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"computedStyle"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("CSS.getComputedStyleForNode: unmarshal: %w", err)
	}
	props := make([]*pb.CSSComputedStyleProperty, len(resp.ComputedStyle))
	for i, p := range resp.ComputedStyle {
		props[i] = &pb.CSSComputedStyleProperty{
			Name:  p.Name,
			Value: p.Value,
		}
	}
	return &pb.GetComputedStyleForNodeResponse{ComputedStyle: props}, nil
}

func (s *Server) GetInlineStylesForNode(ctx context.Context, req *pb.GetInlineStylesForNodeRequest) (*pb.GetInlineStylesForNodeResponse, error) {
	params := map[string]interface{}{
		"nodeId": req.NodeId,
	}
	result, err := s.send(ctx, req.SessionId, "CSS.getInlineStylesForNode", params)
	if err != nil {
		return nil, fmt.Errorf("CSS.getInlineStylesForNode: %w", err)
	}
	var resp struct {
		InlineStyle     *cdpCSSStyle `json:"inlineStyle"`
		AttributesStyle *cdpCSSStyle `json:"attributesStyle"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("CSS.getInlineStylesForNode: unmarshal: %w", err)
	}
	return &pb.GetInlineStylesForNodeResponse{
		InlineStyle:     resp.InlineStyle.toProto(),
		AttributesStyle: resp.AttributesStyle.toProto(),
	}, nil
}

func (s *Server) GetStyleSheetText(ctx context.Context, req *pb.GetStyleSheetTextRequest) (*pb.GetStyleSheetTextResponse, error) {
	params := map[string]interface{}{
		"styleSheetId": req.StyleSheetId,
	}
	result, err := s.send(ctx, req.SessionId, "CSS.getStyleSheetText", params)
	if err != nil {
		return nil, fmt.Errorf("CSS.getStyleSheetText: %w", err)
	}
	var resp struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("CSS.getStyleSheetText: unmarshal: %w", err)
	}
	return &pb.GetStyleSheetTextResponse{Text: resp.Text}, nil
}

func (s *Server) SetStyleSheetText(ctx context.Context, req *pb.SetStyleSheetTextRequest) (*pb.SetStyleSheetTextResponse, error) {
	params := map[string]interface{}{
		"styleSheetId": req.StyleSheetId,
		"text":         req.Text,
	}
	result, err := s.send(ctx, req.SessionId, "CSS.setStyleSheetText", params)
	if err != nil {
		return nil, fmt.Errorf("CSS.setStyleSheetText: %w", err)
	}
	var resp struct {
		SourceMapURL string `json:"sourceMapURL"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("CSS.setStyleSheetText: unmarshal: %w", err)
	}
	return &pb.SetStyleSheetTextResponse{SourceMapUrl: resp.SourceMapURL}, nil
}

func sourceRangeToMap(r *pb.SourceRange) map[string]interface{} {
	if r == nil {
		return nil
	}
	return map[string]interface{}{
		"startLine":   r.StartLine,
		"startColumn": r.StartColumn,
		"endLine":     r.EndLine,
		"endColumn":   r.EndColumn,
	}
}

func (s *Server) SetStyleTexts(ctx context.Context, req *pb.SetStyleTextsRequest) (*pb.SetStyleTextsResponse, error) {
	edits := make([]map[string]interface{}, len(req.Edits))
	for i, e := range req.Edits {
		edit := map[string]interface{}{
			"styleSheetId": e.StyleSheetId,
			"text":         e.Text,
		}
		if e.Range != nil {
			edit["range"] = sourceRangeToMap(e.Range)
		}
		edits[i] = edit
	}
	params := map[string]interface{}{
		"edits": edits,
	}
	result, err := s.send(ctx, req.SessionId, "CSS.setStyleTexts", params)
	if err != nil {
		return nil, fmt.Errorf("CSS.setStyleTexts: %w", err)
	}
	var resp struct {
		Styles []cdpCSSStyle `json:"styles"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("CSS.setStyleTexts: unmarshal: %w", err)
	}
	styles := make([]*pb.CSSStyle, len(resp.Styles))
	for i := range resp.Styles {
		styles[i] = resp.Styles[i].toProto()
	}
	return &pb.SetStyleTextsResponse{Styles: styles}, nil
}

func (s *Server) AddRule(ctx context.Context, req *pb.AddRuleRequest) (*pb.AddRuleResponse, error) {
	params := map[string]interface{}{
		"styleSheetId": req.StyleSheetId,
		"ruleText":     req.RuleText,
	}
	if req.Location != nil {
		params["location"] = sourceRangeToMap(req.Location)
	}
	result, err := s.send(ctx, req.SessionId, "CSS.addRule", params)
	if err != nil {
		return nil, fmt.Errorf("CSS.addRule: %w", err)
	}
	var resp struct {
		Rule cdpCSSRule `json:"rule"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("CSS.addRule: unmarshal: %w", err)
	}
	return &pb.AddRuleResponse{Rule: resp.Rule.toProto()}, nil
}

func (s *Server) CreateStyleSheet(ctx context.Context, req *pb.CreateStyleSheetRequest) (*pb.CreateStyleSheetResponse, error) {
	params := map[string]interface{}{
		"frameId": req.FrameId,
	}
	result, err := s.send(ctx, req.SessionId, "CSS.createStyleSheet", params)
	if err != nil {
		return nil, fmt.Errorf("CSS.createStyleSheet: %w", err)
	}
	var resp struct {
		StyleSheetID string `json:"styleSheetId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("CSS.createStyleSheet: unmarshal: %w", err)
	}
	return &pb.CreateStyleSheetResponse{StyleSheetId: resp.StyleSheetID}, nil
}

func (s *Server) GetBackgroundColors(ctx context.Context, req *pb.GetBackgroundColorsRequest) (*pb.GetBackgroundColorsResponse, error) {
	params := map[string]interface{}{
		"nodeId": req.NodeId,
	}
	result, err := s.send(ctx, req.SessionId, "CSS.getBackgroundColors", params)
	if err != nil {
		return nil, fmt.Errorf("CSS.getBackgroundColors: %w", err)
	}
	var resp struct {
		BackgroundColors   []string `json:"backgroundColors"`
		ComputedFontSize   string   `json:"computedFontSize"`
		ComputedFontWeight string   `json:"computedFontWeight"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("CSS.getBackgroundColors: unmarshal: %w", err)
	}
	return &pb.GetBackgroundColorsResponse{
		BackgroundColors:   resp.BackgroundColors,
		ComputedFontSize:   resp.ComputedFontSize,
		ComputedFontWeight: resp.ComputedFontWeight,
	}, nil
}

func (s *Server) GetPlatformFontsForNode(ctx context.Context, req *pb.GetPlatformFontsForNodeRequest) (*pb.GetPlatformFontsForNodeResponse, error) {
	params := map[string]interface{}{
		"nodeId": req.NodeId,
	}
	result, err := s.send(ctx, req.SessionId, "CSS.getPlatformFontsForNode", params)
	if err != nil {
		return nil, fmt.Errorf("CSS.getPlatformFontsForNode: %w", err)
	}
	var resp struct {
		Fonts []cdpPlatformFontUsage `json:"fonts"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("CSS.getPlatformFontsForNode: unmarshal: %w", err)
	}
	fonts := make([]*pb.PlatformFontUsage, len(resp.Fonts))
	for i := range resp.Fonts {
		fonts[i] = resp.Fonts[i].toProto()
	}
	return &pb.GetPlatformFontsForNodeResponse{Fonts: fonts}, nil
}

func (s *Server) GetMediaQueries(ctx context.Context, req *pb.GetMediaQueriesRequest) (*pb.GetMediaQueriesResponse, error) {
	result, err := s.send(ctx, req.SessionId, "CSS.getMediaQueries", nil)
	if err != nil {
		return nil, fmt.Errorf("CSS.getMediaQueries: %w", err)
	}
	var resp struct {
		Medias []cdpCSSMedia `json:"medias"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("CSS.getMediaQueries: unmarshal: %w", err)
	}
	medias := make([]*pb.CSSMedia, len(resp.Medias))
	for i := range resp.Medias {
		medias[i] = resp.Medias[i].toProto()
	}
	return &pb.GetMediaQueriesResponse{Medias: medias}, nil
}

func (s *Server) StartRuleUsageTracking(ctx context.Context, req *pb.StartRuleUsageTrackingRequest) (*pb.StartRuleUsageTrackingResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "CSS.startRuleUsageTracking", nil); err != nil {
		return nil, fmt.Errorf("CSS.startRuleUsageTracking: %w", err)
	}
	return &pb.StartRuleUsageTrackingResponse{}, nil
}

func (s *Server) StopRuleUsageTracking(ctx context.Context, req *pb.StopRuleUsageTrackingRequest) (*pb.StopRuleUsageTrackingResponse, error) {
	result, err := s.send(ctx, req.SessionId, "CSS.stopRuleUsageTracking", nil)
	if err != nil {
		return nil, fmt.Errorf("CSS.stopRuleUsageTracking: %w", err)
	}
	var resp struct {
		RuleUsage []cdpRuleUsage `json:"ruleUsage"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("CSS.stopRuleUsageTracking: unmarshal: %w", err)
	}
	usage := make([]*pb.RuleUsage, len(resp.RuleUsage))
	for i := range resp.RuleUsage {
		usage[i] = resp.RuleUsage[i].toProto()
	}
	return &pb.StopRuleUsageTrackingResponse{RuleUsage: usage}, nil
}

func (s *Server) TakeCoverageDelta(ctx context.Context, req *pb.TakeCoverageDeltaRequest) (*pb.TakeCoverageDeltaResponse, error) {
	result, err := s.send(ctx, req.SessionId, "CSS.takeCoverageDelta", nil)
	if err != nil {
		return nil, fmt.Errorf("CSS.takeCoverageDelta: %w", err)
	}
	var resp struct {
		Coverage  []cdpRuleUsage `json:"coverage"`
		Timestamp float64        `json:"timestamp"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("CSS.takeCoverageDelta: unmarshal: %w", err)
	}
	coverage := make([]*pb.RuleUsage, len(resp.Coverage))
	for i := range resp.Coverage {
		coverage[i] = resp.Coverage[i].toProto()
	}
	return &pb.TakeCoverageDeltaResponse{
		Coverage:  coverage,
		Timestamp: resp.Timestamp,
	}, nil
}

func (s *Server) CollectClassNames(ctx context.Context, req *pb.CollectClassNamesRequest) (*pb.CollectClassNamesResponse, error) {
	params := map[string]interface{}{
		"styleSheetId": req.StyleSheetId,
	}
	result, err := s.send(ctx, req.SessionId, "CSS.collectClassNames", params)
	if err != nil {
		return nil, fmt.Errorf("CSS.collectClassNames: %w", err)
	}
	var resp struct {
		ClassNames []string `json:"classNames"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("CSS.collectClassNames: unmarshal: %w", err)
	}
	return &pb.CollectClassNamesResponse{ClassNames: resp.ClassNames}, nil
}

func (s *Server) SetRuleSelector(ctx context.Context, req *pb.SetRuleSelectorRequest) (*pb.SetRuleSelectorResponse, error) {
	params := map[string]interface{}{
		"styleSheetId": req.StyleSheetId,
		"selector":     req.Selector,
	}
	if req.Range != nil {
		params["range"] = sourceRangeToMap(req.Range)
	}
	result, err := s.send(ctx, req.SessionId, "CSS.setRuleSelector", params)
	if err != nil {
		return nil, fmt.Errorf("CSS.setRuleSelector: %w", err)
	}
	var resp struct {
		SelectorList *cdpSelectorList `json:"selectorList"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("CSS.setRuleSelector: unmarshal: %w", err)
	}
	selectorText := ""
	if resp.SelectorList != nil {
		selectorText = resp.SelectorList.Text
	}
	return &pb.SetRuleSelectorResponse{SelectorText: selectorText}, nil
}

func (s *Server) ForcePseudoState(ctx context.Context, req *pb.ForcePseudoStateRequest) (*pb.ForcePseudoStateResponse, error) {
	params := map[string]interface{}{
		"nodeId":              req.NodeId,
		"forcedPseudoClasses": req.ForcedPseudoClasses,
	}
	if _, err := s.send(ctx, req.SessionId, "CSS.forcePseudoState", params); err != nil {
		return nil, fmt.Errorf("CSS.forcePseudoState: %w", err)
	}
	return &pb.ForcePseudoStateResponse{}, nil
}

// SubscribeEvents streams CDP CSS events to the gRPC client.
func (s *Server) SubscribeEvents(req *pb.SubscribeEventsRequest, stream grpc.ServerStreamingServer[pb.CSSEvent]) error {
	eventCh := make(chan *pb.CSSEvent, 128)
	ctx := stream.Context()

	cssEvents := []string{
		"CSS.styleSheetAdded",
		"CSS.styleSheetRemoved",
		"CSS.styleSheetChanged",
		"CSS.mediaQueryResultChanged",
		"CSS.fontsUpdated",
	}

	unregisters := make([]func(), 0, len(cssEvents))
	for _, method := range cssEvents {
		method := method
		unreg := s.client.On(method, func(m string, params json.RawMessage, sessionID string) {
			evt := convertCSSEvent(method, params)
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

func convertCSSEvent(method string, params json.RawMessage) *pb.CSSEvent {
	switch method {
	case "CSS.styleSheetAdded":
		var d struct {
			Header cdpStyleSheetHeader `json:"header"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.CSSEvent{Event: &pb.CSSEvent_StyleSheetAdded{
			StyleSheetAdded: &pb.StyleSheetAddedEvent{
				Header: d.Header.toProto(),
			},
		}}

	case "CSS.styleSheetRemoved":
		var d struct {
			StyleSheetID string `json:"styleSheetId"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.CSSEvent{Event: &pb.CSSEvent_StyleSheetRemoved{
			StyleSheetRemoved: &pb.StyleSheetRemovedEvent{
				StyleSheetId: d.StyleSheetID,
			},
		}}

	case "CSS.styleSheetChanged":
		var d struct {
			StyleSheetID string `json:"styleSheetId"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.CSSEvent{Event: &pb.CSSEvent_StyleSheetChanged{
			StyleSheetChanged: &pb.StyleSheetChangedEvent{
				StyleSheetId: d.StyleSheetID,
			},
		}}

	case "CSS.mediaQueryResultChanged":
		return &pb.CSSEvent{Event: &pb.CSSEvent_MediaQueryResultChanged{
			MediaQueryResultChanged: &pb.MediaQueryResultChangedEvent{},
		}}

	case "CSS.fontsUpdated":
		var d struct {
			Font json.RawMessage `json:"font"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		fontStr := ""
		if d.Font != nil {
			fontStr = string(d.Font)
		}
		return &pb.CSSEvent{Event: &pb.CSSEvent_FontsUpdated{
			FontsUpdated: &pb.FontsUpdatedEvent{
				Font: fontStr,
			},
		}}

	default:
		return nil
	}
}
