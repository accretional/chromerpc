package page

import (
	"encoding/base64"
	"encoding/json"

	pb "github.com/anthropics/chromerpc/proto/cdp/page"
)

// --- enum string conversions ---

func transitionTypeToString(t pb.TransitionType) string {
	switch t {
	case pb.TransitionType_TRANSITION_TYPE_LINK:
		return "link"
	case pb.TransitionType_TRANSITION_TYPE_TYPED:
		return "typed"
	case pb.TransitionType_TRANSITION_TYPE_ADDRESS_BAR:
		return "address_bar"
	case pb.TransitionType_TRANSITION_TYPE_AUTO_BOOKMARK:
		return "auto_bookmark"
	case pb.TransitionType_TRANSITION_TYPE_AUTO_SUBFRAME:
		return "auto_subframe"
	case pb.TransitionType_TRANSITION_TYPE_MANUAL_SUBFRAME:
		return "manual_subframe"
	case pb.TransitionType_TRANSITION_TYPE_GENERATED:
		return "generated"
	case pb.TransitionType_TRANSITION_TYPE_AUTO_TOPLEVEL:
		return "auto_toplevel"
	case pb.TransitionType_TRANSITION_TYPE_FORM_SUBMIT:
		return "form_submit"
	case pb.TransitionType_TRANSITION_TYPE_RELOAD:
		return "reload"
	case pb.TransitionType_TRANSITION_TYPE_KEYWORD:
		return "keyword"
	case pb.TransitionType_TRANSITION_TYPE_KEYWORD_GENERATED:
		return "keyword_generated"
	case pb.TransitionType_TRANSITION_TYPE_OTHER:
		return "other"
	default:
		return ""
	}
}

func stringToTransitionType(s string) pb.TransitionType {
	switch s {
	case "link":
		return pb.TransitionType_TRANSITION_TYPE_LINK
	case "typed":
		return pb.TransitionType_TRANSITION_TYPE_TYPED
	case "address_bar":
		return pb.TransitionType_TRANSITION_TYPE_ADDRESS_BAR
	case "auto_bookmark":
		return pb.TransitionType_TRANSITION_TYPE_AUTO_BOOKMARK
	case "auto_subframe":
		return pb.TransitionType_TRANSITION_TYPE_AUTO_SUBFRAME
	case "manual_subframe":
		return pb.TransitionType_TRANSITION_TYPE_MANUAL_SUBFRAME
	case "generated":
		return pb.TransitionType_TRANSITION_TYPE_GENERATED
	case "auto_toplevel":
		return pb.TransitionType_TRANSITION_TYPE_AUTO_TOPLEVEL
	case "form_submit":
		return pb.TransitionType_TRANSITION_TYPE_FORM_SUBMIT
	case "reload":
		return pb.TransitionType_TRANSITION_TYPE_RELOAD
	case "keyword":
		return pb.TransitionType_TRANSITION_TYPE_KEYWORD
	case "keyword_generated":
		return pb.TransitionType_TRANSITION_TYPE_KEYWORD_GENERATED
	case "other":
		return pb.TransitionType_TRANSITION_TYPE_OTHER
	default:
		return pb.TransitionType_TRANSITION_TYPE_UNSPECIFIED
	}
}

func referrerPolicyToString(p pb.ReferrerPolicy) string {
	switch p {
	case pb.ReferrerPolicy_REFERRER_POLICY_NO_REFERRER:
		return "noReferrer"
	case pb.ReferrerPolicy_REFERRER_POLICY_NO_REFERRER_WHEN_DOWNGRADE:
		return "noReferrerWhenDowngrade"
	case pb.ReferrerPolicy_REFERRER_POLICY_ORIGIN:
		return "origin"
	case pb.ReferrerPolicy_REFERRER_POLICY_ORIGIN_WHEN_CROSS_ORIGIN:
		return "originWhenCrossOrigin"
	case pb.ReferrerPolicy_REFERRER_POLICY_SAME_ORIGIN:
		return "sameOrigin"
	case pb.ReferrerPolicy_REFERRER_POLICY_STRICT_ORIGIN:
		return "strictOrigin"
	case pb.ReferrerPolicy_REFERRER_POLICY_STRICT_ORIGIN_WHEN_CROSS_ORIGIN:
		return "strictOriginWhenCrossOrigin"
	case pb.ReferrerPolicy_REFERRER_POLICY_UNSAFE_URL:
		return "unsafeUrl"
	default:
		return ""
	}
}

func screenshotFormatToString(f pb.ScreenshotFormat) string {
	switch f {
	case pb.ScreenshotFormat_SCREENSHOT_FORMAT_JPEG:
		return "jpeg"
	case pb.ScreenshotFormat_SCREENSHOT_FORMAT_PNG:
		return "png"
	case pb.ScreenshotFormat_SCREENSHOT_FORMAT_WEBP:
		return "webp"
	default:
		return ""
	}
}

func stringToDialogType(s string) pb.DialogType {
	switch s {
	case "alert":
		return pb.DialogType_DIALOG_TYPE_ALERT
	case "confirm":
		return pb.DialogType_DIALOG_TYPE_CONFIRM
	case "prompt":
		return pb.DialogType_DIALOG_TYPE_PROMPT
	case "beforeunload":
		return pb.DialogType_DIALOG_TYPE_BEFOREUNLOAD
	default:
		return pb.DialogType_DIALOG_TYPE_UNSPECIFIED
	}
}

// --- CDP JSON types for deserialization ---

type cdpFrame struct {
	ID                              string         `json:"id"`
	ParentID                        string         `json:"parentId"`
	LoaderID                        string         `json:"loaderId"`
	Name                            string         `json:"name"`
	URL                             string         `json:"url"`
	URLFragment                     string         `json:"urlFragment"`
	DomainAndRegistry               string         `json:"domainAndRegistry"`
	SecurityOrigin                  string         `json:"securityOrigin"`
	MimeType                        string         `json:"mimeType"`
	UnreachableURL                  string         `json:"unreachableUrl"`
	AdFrameStatus                   *cdpAdFrame    `json:"adFrameStatus"`
	SecureContextType               string         `json:"secureContextType"`
	CrossOriginIsolatedContextType  string         `json:"crossOriginIsolatedContextType"`
	GatedAPIFeatures                []string       `json:"gatedAPIFeatures"`
}

type cdpAdFrame struct {
	AdFrameType  string   `json:"adFrameType"`
	Explanations []string `json:"explanations"`
}

func (f *cdpFrame) toProto() *pb.Frame {
	frame := &pb.Frame{
		Id:                f.ID,
		ParentId:          f.ParentID,
		LoaderId:          f.LoaderID,
		Name:              f.Name,
		Url:               f.URL,
		UrlFragment:       f.URLFragment,
		DomainAndRegistry: f.DomainAndRegistry,
		SecurityOrigin:    f.SecurityOrigin,
		MimeType:          f.MimeType,
		UnreachableUrl:    f.UnreachableURL,
	}
	if f.AdFrameStatus != nil {
		ads := &pb.AdFrameStatus{}
		switch f.AdFrameStatus.AdFrameType {
		case "none":
			ads.AdFrameType = pb.AdFrameType_AD_FRAME_TYPE_NONE
		case "child":
			ads.AdFrameType = pb.AdFrameType_AD_FRAME_TYPE_CHILD
		case "root":
			ads.AdFrameType = pb.AdFrameType_AD_FRAME_TYPE_ROOT
		}
		for _, exp := range f.AdFrameStatus.Explanations {
			switch exp {
			case "ParentIsAd":
				ads.Explanations = append(ads.Explanations, pb.AdFrameExplanation_AD_FRAME_EXPLANATION_PARENT_IS_AD)
			case "CreatedByAdScript":
				ads.Explanations = append(ads.Explanations, pb.AdFrameExplanation_AD_FRAME_EXPLANATION_CREATED_BY_AD_SCRIPT)
			case "MatchedBlockingRule":
				ads.Explanations = append(ads.Explanations, pb.AdFrameExplanation_AD_FRAME_EXPLANATION_MATCHED_BLOCKING_RULE)
			}
		}
		frame.AdFrameStatus = ads
	}
	switch f.SecureContextType {
	case "Secure":
		frame.SecureContextType = pb.SecureContextType_SECURE_CONTEXT_TYPE_SECURE
	case "SecureLocalhost":
		frame.SecureContextType = pb.SecureContextType_SECURE_CONTEXT_TYPE_SECURE_LOCALHOST
	case "InsecureScheme":
		frame.SecureContextType = pb.SecureContextType_SECURE_CONTEXT_TYPE_INSECURE_SCHEME
	case "InsecureAncestor":
		frame.SecureContextType = pb.SecureContextType_SECURE_CONTEXT_TYPE_INSECURE_ANCESTOR
	}
	switch f.CrossOriginIsolatedContextType {
	case "Isolated":
		frame.CrossOriginIsolatedContextType = pb.CrossOriginIsolatedContextType_CROSS_ORIGIN_ISOLATED_CONTEXT_TYPE_ISOLATED
	case "NotIsolated":
		frame.CrossOriginIsolatedContextType = pb.CrossOriginIsolatedContextType_CROSS_ORIGIN_ISOLATED_CONTEXT_TYPE_NOT_ISOLATED
	case "NotIsolatedFeatureDisabled":
		frame.CrossOriginIsolatedContextType = pb.CrossOriginIsolatedContextType_CROSS_ORIGIN_ISOLATED_CONTEXT_TYPE_NOT_ISOLATED_FEATURE_DISABLED
	}
	for _, feat := range f.GatedAPIFeatures {
		switch feat {
		case "SharedArrayBuffers":
			frame.GatedApiFeatures = append(frame.GatedApiFeatures, pb.GatedAPIFeature_GATED_API_FEATURE_SHARED_ARRAY_BUFFERS)
		case "SharedArrayBuffersTransferAllowed":
			frame.GatedApiFeatures = append(frame.GatedApiFeatures, pb.GatedAPIFeature_GATED_API_FEATURE_SHARED_ARRAY_BUFFERS_TRANSFER_ALLOWED)
		case "PerformanceMeasureMemory":
			frame.GatedApiFeatures = append(frame.GatedApiFeatures, pb.GatedAPIFeature_GATED_API_FEATURE_PERFORMANCE_MEASURE_MEMORY)
		case "PerformanceProfile":
			frame.GatedApiFeatures = append(frame.GatedApiFeatures, pb.GatedAPIFeature_GATED_API_FEATURE_PERFORMANCE_PROFILE)
		}
	}
	return frame
}

type cdpFrameTree struct {
	Frame       cdpFrame       `json:"frame"`
	ChildFrames []cdpFrameTree `json:"childFrames"`
}

func (t *cdpFrameTree) toProto() *pb.FrameTree {
	ft := &pb.FrameTree{
		Frame: t.Frame.toProto(),
	}
	for _, child := range t.ChildFrames {
		child := child
		ft.ChildFrames = append(ft.ChildFrames, child.toProto())
	}
	return ft
}

type cdpFrameResource struct {
	URL          string  `json:"url"`
	Type         string  `json:"type"`
	MimeType     string  `json:"mimeType"`
	LastModified float64 `json:"lastModified"`
	ContentSize  float64 `json:"contentSize"`
	Failed       bool    `json:"failed"`
	Canceled     bool    `json:"canceled"`
}

type cdpFrameResourceTree struct {
	Frame       cdpFrame               `json:"frame"`
	ChildFrames []cdpFrameResourceTree `json:"childFrames"`
	Resources   []cdpFrameResource     `json:"resources"`
}

func (t *cdpFrameResourceTree) toProto() *pb.FrameResourceTree {
	ft := &pb.FrameResourceTree{
		Frame: t.Frame.toProto(),
	}
	for _, child := range t.ChildFrames {
		child := child
		ft.ChildFrames = append(ft.ChildFrames, child.toProto())
	}
	for _, r := range t.Resources {
		ft.Resources = append(ft.Resources, &pb.FrameResource{
			Url:          r.URL,
			Type:         r.Type,
			MimeType:     r.MimeType,
			LastModified: r.LastModified,
			ContentSize:  r.ContentSize,
			Failed:       r.Failed,
			Canceled:     r.Canceled,
		})
	}
	return ft
}

// --- Event conversion ---

func convertPageEvent(method string, params json.RawMessage) *pb.PageEvent {
	switch method {
	case "Page.domContentEventFired":
		var d struct {
			Timestamp float64 `json:"timestamp"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PageEvent{Event: &pb.PageEvent_DomContentEventFired{
			DomContentEventFired: &pb.DomContentEventFiredEvent{Timestamp: d.Timestamp},
		}}

	case "Page.loadEventFired":
		var d struct {
			Timestamp float64 `json:"timestamp"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PageEvent{Event: &pb.PageEvent_LoadEventFired{
			LoadEventFired: &pb.LoadEventFiredEvent{Timestamp: d.Timestamp},
		}}

	case "Page.frameAttached":
		var d struct {
			FrameID       string `json:"frameId"`
			ParentFrameID string `json:"parentFrameId"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PageEvent{Event: &pb.PageEvent_FrameAttached{
			FrameAttached: &pb.FrameAttachedEvent{
				FrameId:       d.FrameID,
				ParentFrameId: d.ParentFrameID,
			},
		}}

	case "Page.frameDetached":
		var d struct {
			FrameID string `json:"frameId"`
			Reason  string `json:"reason"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PageEvent{Event: &pb.PageEvent_FrameDetached{
			FrameDetached: &pb.FrameDetachedEvent{
				FrameId: d.FrameID,
				Reason:  d.Reason,
			},
		}}

	case "Page.frameNavigated":
		var d struct {
			Frame cdpFrame `json:"frame"`
			Type  string   `json:"type"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PageEvent{Event: &pb.PageEvent_FrameNavigated{
			FrameNavigated: &pb.FrameNavigatedEvent{
				Frame: d.Frame.toProto(),
				Type:  d.Type,
			},
		}}

	case "Page.javascriptDialogOpening":
		var d struct {
			URL               string `json:"url"`
			Message           string `json:"message"`
			Type              string `json:"type"`
			HasBrowserHandler bool   `json:"hasBrowserHandler"`
			DefaultPrompt     string `json:"defaultPrompt"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PageEvent{Event: &pb.PageEvent_JavascriptDialogOpening{
			JavascriptDialogOpening: &pb.JavascriptDialogOpeningEvent{
				Url:               d.URL,
				Message:           d.Message,
				Type:              stringToDialogType(d.Type),
				HasBrowserHandler: d.HasBrowserHandler,
				DefaultPrompt:     d.DefaultPrompt,
			},
		}}

	case "Page.javascriptDialogClosed":
		var d struct {
			Result    bool   `json:"result"`
			UserInput string `json:"userInput"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PageEvent{Event: &pb.PageEvent_JavascriptDialogClosed{
			JavascriptDialogClosed: &pb.JavascriptDialogClosedEvent{
				Result:    d.Result,
				UserInput: d.UserInput,
			},
		}}

	case "Page.lifecycleEvent":
		var d struct {
			FrameID   string  `json:"frameId"`
			LoaderID  string  `json:"loaderId"`
			Name      string  `json:"name"`
			Timestamp float64 `json:"timestamp"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PageEvent{Event: &pb.PageEvent_LifecycleEvent{
			LifecycleEvent: &pb.LifecycleEventEvent{
				FrameId:   d.FrameID,
				LoaderId:  d.LoaderID,
				Name:      d.Name,
				Timestamp: d.Timestamp,
			},
		}}

	case "Page.windowOpen":
		var d struct {
			URL            string   `json:"url"`
			WindowName     string   `json:"windowName"`
			WindowFeatures []string `json:"windowFeatures"`
			UserGesture    bool     `json:"userGesture"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PageEvent{Event: &pb.PageEvent_WindowOpen{
			WindowOpen: &pb.WindowOpenEvent{
				Url:            d.URL,
				WindowName:     d.WindowName,
				WindowFeatures: d.WindowFeatures,
				UserGesture:    d.UserGesture,
			},
		}}

	case "Page.frameStartedLoading":
		var d struct {
			FrameID string `json:"frameId"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PageEvent{Event: &pb.PageEvent_FrameStartedLoading{
			FrameStartedLoading: &pb.FrameStartedLoadingEvent{FrameId: d.FrameID},
		}}

	case "Page.frameStoppedLoading":
		var d struct {
			FrameID string `json:"frameId"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PageEvent{Event: &pb.PageEvent_FrameStoppedLoading{
			FrameStoppedLoading: &pb.FrameStoppedLoadingEvent{FrameId: d.FrameID},
		}}

	case "Page.frameStartedNavigating":
		var d struct {
			FrameID        string `json:"frameId"`
			URL            string `json:"url"`
			LoaderID       string `json:"loaderId"`
			NavigationType string `json:"navigationType"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PageEvent{Event: &pb.PageEvent_FrameStartedNavigating{
			FrameStartedNavigating: &pb.FrameStartedNavigatingEvent{
				FrameId:        d.FrameID,
				Url:            d.URL,
				LoaderId:       d.LoaderID,
				NavigationType: d.NavigationType,
			},
		}}

	case "Page.frameRequestedNavigation":
		var d struct {
			FrameID     string `json:"frameId"`
			Reason      string `json:"reason"`
			URL         string `json:"url"`
			Disposition string `json:"disposition"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PageEvent{Event: &pb.PageEvent_FrameRequestedNavigation{
			FrameRequestedNavigation: &pb.FrameRequestedNavigationEvent{
				FrameId:     d.FrameID,
				Reason:      d.Reason,
				Url:         d.URL,
				Disposition: d.Disposition,
			},
		}}

	case "Page.navigatedWithinDocument":
		var d struct {
			FrameID        string `json:"frameId"`
			URL            string `json:"url"`
			NavigationType string `json:"navigationType"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PageEvent{Event: &pb.PageEvent_NavigatedWithinDocument{
			NavigatedWithinDocument: &pb.NavigatedWithinDocumentEvent{
				FrameId:        d.FrameID,
				Url:            d.URL,
				NavigationType: d.NavigationType,
			},
		}}

	case "Page.interstitialShown":
		return &pb.PageEvent{Event: &pb.PageEvent_InterstitialShown{
			InterstitialShown: &pb.InterstitialShownEvent{},
		}}

	case "Page.interstitialHidden":
		return &pb.PageEvent{Event: &pb.PageEvent_InterstitialHidden{
			InterstitialHidden: &pb.InterstitialHiddenEvent{},
		}}

	case "Page.fileChooserOpened":
		var d struct {
			FrameID       string `json:"frameId"`
			Mode          string `json:"mode"`
			BackendNodeID int32  `json:"backendNodeId"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PageEvent{Event: &pb.PageEvent_FileChooserOpened{
			FileChooserOpened: &pb.FileChooserOpenedEvent{
				FrameId:       d.FrameID,
				Mode:          d.Mode,
				BackendNodeId: d.BackendNodeID,
			},
		}}

	case "Page.screencastFrame":
		var d struct {
			Data     string `json:"data"`
			Metadata struct {
				OffsetTop       float64 `json:"offsetTop"`
				PageScaleFactor float64 `json:"pageScaleFactor"`
				DeviceWidth     float64 `json:"deviceWidth"`
				DeviceHeight    float64 `json:"deviceHeight"`
				ScrollOffsetX   float64 `json:"scrollOffsetX"`
				ScrollOffsetY   float64 `json:"scrollOffsetY"`
				Timestamp       float64 `json:"timestamp"`
			} `json:"metadata"`
			SessionID int32 `json:"sessionId"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		imageData, _ := base64.StdEncoding.DecodeString(d.Data)
		return &pb.PageEvent{Event: &pb.PageEvent_ScreencastFrame{
			ScreencastFrame: &pb.ScreencastFrameEvent{
				Data: imageData,
				Metadata: &pb.ScreencastFrameMetadata{
					OffsetTop:       d.Metadata.OffsetTop,
					PageScaleFactor: d.Metadata.PageScaleFactor,
					DeviceWidth:     d.Metadata.DeviceWidth,
					DeviceHeight:    d.Metadata.DeviceHeight,
					ScrollOffsetX:   d.Metadata.ScrollOffsetX,
					ScrollOffsetY:   d.Metadata.ScrollOffsetY,
					Timestamp:       d.Metadata.Timestamp,
				},
				SessionId: d.SessionID,
			},
		}}

	case "Page.screencastVisibilityChanged":
		var d struct {
			Visible bool `json:"visible"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PageEvent{Event: &pb.PageEvent_ScreencastVisibilityChanged{
			ScreencastVisibilityChanged: &pb.ScreencastVisibilityChangedEvent{Visible: d.Visible},
		}}

	case "Page.documentOpened":
		var d struct {
			Frame cdpFrame `json:"frame"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.PageEvent{Event: &pb.PageEvent_DocumentOpened{
			DocumentOpened: &pb.DocumentOpenedEvent{Frame: d.Frame.toProto()},
		}}

	case "Page.frameResized":
		return &pb.PageEvent{Event: &pb.PageEvent_FrameResized{
			FrameResized: &pb.FrameResizedEvent{},
		}}

	case "Page.backForwardCacheNotUsed":
		var d struct {
			LoaderID                  string `json:"loaderId"`
			FrameID                   string `json:"frameId"`
			NotRestoredExplanations   []struct {
				Type    string `json:"type"`
				Reason  string `json:"reason"`
				Context string `json:"context"`
			} `json:"notRestoredExplanations"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		explanations := make([]*pb.BackForwardCacheNotRestoredExplanation, len(d.NotRestoredExplanations))
		for i, e := range d.NotRestoredExplanations {
			explanations[i] = &pb.BackForwardCacheNotRestoredExplanation{
				Type:    e.Type,
				Reason:  e.Reason,
				Context: e.Context,
			}
		}
		return &pb.PageEvent{Event: &pb.PageEvent_BackForwardCacheNotUsed{
			BackForwardCacheNotUsed: &pb.BackForwardCacheNotUsedEvent{
				LoaderId:                  d.LoaderID,
				FrameId:                   d.FrameID,
				NotRestoredExplanations:   explanations,
			},
		}}

	case "Page.compilationCacheProduced":
		var d struct {
			URL  string `json:"url"`
			Data string `json:"data"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		data, _ := base64.StdEncoding.DecodeString(d.Data)
		return &pb.PageEvent{Event: &pb.PageEvent_CompilationCacheProduced{
			CompilationCacheProduced: &pb.CompilationCacheProducedEvent{
				Url:  d.URL,
				Data: data,
			},
		}}
	}
	return nil
}
