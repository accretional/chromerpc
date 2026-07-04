// Command cdpgen generates gRPC .proto files from a Chrome DevTools Protocol JSON
// definition (as produced by scripts/cdp-pull.sh from a live Chrome's
// /json/protocol endpoint, or the upstream browser_protocol.json/js_protocol.json).
//
// This is the Phase-3 "auto-generate protos from CDP" tool (see docs/refactor).
// It encodes the conventions the hand-written protos use:
//
//   - one .proto per domain: package cdp.<lower>, go_package .../proto/cdp/<lower>
//   - service <Domain>Service; each command -> rpc <Pascal>(<Cmd>Request) returns (<Cmd>Response)
//   - request/response messages always emitted (even when empty)
//   - CDP camelCase fields -> proto snake_case; a `string session_id = 99;` adapter
//     field is appended to every Request
//   - domain types -> messages (objects) / enums; scalar-alias types resolve to scalars
//   - events -> one server-streaming rpc SubscribeEvents returning a stream of a
//     <Domain>Event oneof (one arm per event)
//   - type mapping: string->string, integer->int32, number->double, boolean->bool,
//     binary->bytes, array->repeated, object/$ref->message
//
// Scope note (prototype): cross-domain $refs and inline object/enum properties are
// emitted as scalar placeholders with a `// cdpgen: ...` comment rather than fully
// resolved/imported. Same-domain $refs and enums are resolved. Extending to full
// cross-domain imports is the next iteration.
//
// Usage:
//
//	cdpgen -proto proto/cdp/_upstream/chrome-protocol.json -domain Page -out /tmp
//	cdpgen -proto ... -domain all -out proto/cdp        # regenerate every domain
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type protocol struct {
	Domains []domain `json:"domains"`
}
type domain struct {
	Domain       string    `json:"domain"`
	Experimental bool      `json:"experimental"`
	Deprecated   bool      `json:"deprecated"`
	Types        []typeDef `json:"types"`
	Commands     []member  `json:"commands"`
	Events       []member  `json:"events"`
}
type typeDef struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Enum         []string `json:"enum"`
	Properties   []prop   `json:"properties"`
	Items        *prop    `json:"items"`
	Experimental bool     `json:"experimental"`
}
type member struct { // command or event
	Name         string `json:"name"`
	Experimental bool   `json:"experimental"`
	Deprecated   bool   `json:"deprecated"`
	Parameters   []prop `json:"parameters"`
	Returns      []prop `json:"returns"`
}
type prop struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Ref          string   `json:"$ref"`
	Enum         []string `json:"enum"`
	Items        *prop    `json:"items"`
	Properties   []prop   `json:"properties"`
	Optional     bool     `json:"optional"`
	Description  string   `json:"description"`
	Experimental bool     `json:"experimental"`
	Deprecated   bool     `json:"deprecated"`
}

func main() {
	protoPath := flag.String("proto", "", "path to CDP protocol JSON (chrome-protocol.json)")
	domName := flag.String("domain", "", "domain to generate, or 'all'")
	outDir := flag.String("out", ".", "output directory (per-domain subdirs when 'all')")
	flag.Parse()
	if *protoPath == "" || *domName == "" {
		fmt.Fprintln(os.Stderr, "usage: cdpgen -proto <protocol.json> -domain <Name|all> -out <dir>")
		os.Exit(2)
	}

	raw, err := os.ReadFile(*protoPath)
	must(err)
	// The live /json/protocol payload wraps the protocol; upstream files are the
	// protocol directly. Support both, and merge multiple files' domains by union.
	var p protocol
	if err := json.Unmarshal(raw, &p); err != nil || len(p.Domains) == 0 {
		// try {"protocol": {...}} or {"domains": [...]} shapes
		var wrap struct {
			Protocol protocol `json:"protocol"`
		}
		must(json.Unmarshal(raw, &wrap))
		p = wrap.Protocol
	}
	if len(p.Domains) == 0 {
		fmt.Fprintln(os.Stderr, "no domains found in protocol JSON")
		os.Exit(1)
	}

	targets := p.Domains
	if !strings.EqualFold(*domName, "all") {
		targets = nil
		for _, d := range p.Domains {
			if strings.EqualFold(d.Domain, *domName) {
				targets = append(targets, d)
			}
		}
		if len(targets) == 0 {
			fmt.Fprintf(os.Stderr, "domain %q not found\n", *domName)
			os.Exit(1)
		}
	}

	for _, d := range targets {
		out := genDomain(d)
		lower := strings.ToLower(d.Domain)
		var dir, file string
		if strings.EqualFold(*domName, "all") {
			dir = filepath.Join(*outDir, lower)
		} else {
			dir = *outDir
		}
		must(os.MkdirAll(dir, 0o755))
		file = filepath.Join(dir, lower+".proto")
		must(os.WriteFile(file, []byte(out), 0o644))
		fmt.Printf("wrote %s (%d commands, %d events, %d types)\n",
			file, len(d.Commands), len(d.Events), len(d.Types))
	}
}

// genDomain renders a full .proto for one CDP domain.
func genDomain(d domain) string {
	lower := strings.ToLower(d.Domain)
	// Resolve local type IDs -> proto type (scalar for aliases; message/enum name otherwise).
	local := map[string]string{}
	for _, t := range d.Types {
		switch {
		case len(t.Enum) > 0:
			local[t.ID] = t.ID // proto enum
		case t.Type == "object" && len(t.Properties) > 0:
			local[t.ID] = t.ID // message
		case t.Type == "array":
			local[t.ID] = t.ID // wrapper message
		case t.Type == "object":
			local[t.ID] = "" // open object -> string (see resolve)
		default:
			local[t.ID] = scalar(t.Type) // scalar alias
		}
	}
	g := &gen{dom: d.Domain, local: local}

	// The event oneof wrapper is normally "<Domain>Event", but some domains define
	// a *type* of that exact name (e.g. BackgroundService.BackgroundServiceEvent) —
	// use a distinct wrapper name to avoid a message-name collision.
	evtWrap := d.Domain + "Event"
	if _, clash := local[evtWrap]; clash {
		evtWrap = d.Domain + "Events"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "// Code generated by cdpgen from the Chrome DevTools Protocol. DO NOT EDIT.\n")
	fmt.Fprintf(&b, "// Domain: %s%s\n", d.Domain, flagComment(d.Experimental, d.Deprecated))
	fmt.Fprintf(&b, "syntax = \"proto3\";\n\n")
	fmt.Fprintf(&b, "package cdp.%s;\n\n", lower)
	fmt.Fprintf(&b, "option go_package = \"github.com/accretional/chromerpc/proto/cdp/%s\";\n\n", lower)

	// Service
	fmt.Fprintf(&b, "service %sService {\n", d.Domain)
	for _, c := range d.Commands {
		fmt.Fprintf(&b, "  rpc %s(%sRequest) returns (%sResponse);%s\n",
			pascal(c.Name), pascal(c.Name), pascal(c.Name), flagComment(c.Experimental, c.Deprecated))
	}
	if len(d.Events) > 0 {
		fmt.Fprintf(&b, "  rpc SubscribeEvents(Subscribe%sEventsRequest) returns (stream %s);\n",
			d.Domain, evtWrap)
	}
	fmt.Fprintf(&b, "}\n")

	// Domain types
	if len(d.Types) > 0 {
		fmt.Fprintf(&b, "\n// ==================== Types ====================\n")
		for _, t := range d.Types {
			g.renderType(&b, t)
		}
	}

	// Command request/response messages
	fmt.Fprintf(&b, "\n// ==================== Commands ====================\n")
	for _, c := range d.Commands {
		g.renderMessage(&b, pascal(c.Name)+"Request", c.Parameters, true)
		g.renderMessage(&b, pascal(c.Name)+"Response", c.Returns, false)
	}

	// Events
	if len(d.Events) > 0 {
		fmt.Fprintf(&b, "\n// ==================== Events ====================\n")
		fmt.Fprintf(&b, "message Subscribe%sEventsRequest {\n  string session_id = 1;\n}\n", d.Domain)
		// per-event payload messages
		for _, e := range d.Events {
			g.renderMessage(&b, pascal(e.Name)+"Event", e.Parameters, false)
		}
		// oneof wrapper
		fmt.Fprintf(&b, "message %s {\n  oneof event {\n", evtWrap)
		for i, e := range d.Events {
			fmt.Fprintf(&b, "    %sEvent %s = %d;\n", pascal(e.Name), snake(e.Name), i+1)
		}
		fmt.Fprintf(&b, "  }\n}\n")
	}
	return b.String()
}

type gen struct {
	dom   string
	local map[string]string
}

func (g *gen) renderType(b *strings.Builder, t typeDef) {
	switch {
	case len(t.Enum) > 0:
		fmt.Fprintf(b, "enum %s {\n", t.ID)
		fmt.Fprintf(b, "  %s_UNSPECIFIED = 0;\n", screamingSnake(t.ID))
		for i, v := range t.Enum {
			fmt.Fprintf(b, "  %s_%s = %d;\n", screamingSnake(t.ID), screamingSnake(v), i+1)
		}
		fmt.Fprintf(b, "}\n")
	case t.Type == "object" && len(t.Properties) > 0:
		g.renderMessage(b, t.ID, t.Properties, false)
	case t.Type == "array":
		item := "string"
		if t.Items != nil {
			item = g.resolve(*t.Items)
		}
		fmt.Fprintf(b, "message %s {\n  repeated %s items = 1;\n}\n", t.ID, item)
	case t.Type == "object":
		fmt.Fprintf(b, "message %s {\n  string json = 1; // cdpgen: open object (arbitrary JSON)\n}\n", t.ID)
	default:
		// scalar alias — no message; documented for readers.
		fmt.Fprintf(b, "// type %s = %s (CDP scalar alias)\n", t.ID, scalar(t.Type))
	}
}

// renderMessage emits a message from a list of props. If isRequest, append the
// session_id = 99 adapter field.
func (g *gen) renderMessage(b *strings.Builder, name string, props []prop, isRequest bool) {
	fmt.Fprintf(b, "message %s {\n", name)
	n := 1
	hasSession := false
	for _, p := range props {
		if snake(p.Name) == "session_id" {
			hasSession = true
		}
		typ := g.resolve(p)
		rep := ""
		if p.Type == "array" {
			rep = "repeated "
		}
		fmt.Fprintf(b, "  %s%s %s = %d;%s\n", rep, typ, snake(p.Name), n, flagComment(p.Experimental, p.Deprecated))
		n++
	}
	// Append the adapter session field only when the CDP command doesn't already
	// carry its own sessionId param (Target/Page have a few that do).
	if isRequest && !hasSession {
		fmt.Fprintf(b, "  string session_id = 99; // cdpgen: adapter session routing (not CDP)\n")
	}
	fmt.Fprintf(b, "}\n")
}

// resolve maps a CDP property to a proto scalar/message type name.
func (g *gen) resolve(p prop) string {
	if p.Type == "array" {
		if p.Items == nil {
			return "string"
		}
		return g.resolve(*p.Items)
	}
	if p.Ref != "" {
		if strings.Contains(p.Ref, ".") {
			return fmt.Sprintf("string /* cdpgen: cross-domain ref %s */", p.Ref)
		}
		if t, ok := g.local[p.Ref]; ok {
			if t == "" {
				return "string" // open-object alias
			}
			return t
		}
		return fmt.Sprintf("string /* cdpgen: unknown ref %s */", p.Ref)
	}
	if len(p.Enum) > 0 {
		return "string" // cdpgen: inline enum (values in CDP docs)
	}
	if p.Type == "object" {
		return "string" // cdpgen: inline object (JSON)
	}
	return scalar(p.Type)
}

func scalar(t string) string {
	switch t {
	case "string":
		return "string"
	case "integer":
		return "int32"
	case "number":
		return "double"
	case "boolean":
		return "bool"
	case "binary":
		return "bytes"
	case "any":
		return "string"
	default:
		return "string"
	}
}

func flagComment(exp, dep bool) string {
	var s []string
	if exp {
		s = append(s, "experimental")
	}
	if dep {
		s = append(s, "deprecated")
	}
	if len(s) == 0 {
		return ""
	}
	return "  // " + strings.Join(s, ", ")
}

// pascal upper-cases the first rune (captureScreenshot -> CaptureScreenshot,
// printToPDF -> PrintToPDF).
func pascal(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// snake converts camelCase to snake_case (frameId -> frame_id).
func snake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r - 'A' + 'a')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// screamingSnake produces a valid SCREAMING_SNAKE proto identifier: uppercase,
// non-alphanumerics -> '_', and a leading '_' if it would start with a digit.
func screamingSnake(s string) string {
	up := strings.ToUpper(snake(s))
	var b strings.Builder
	for _, r := range up {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := b.String()
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	out = strings.Trim(out, "_")
	if out == "" {
		out = "VALUE"
	}
	if out[0] >= '0' && out[0] <= '9' {
		out = "_" + out
	}
	return out
}

var _ = sort.Strings

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "cdpgen:", err)
		os.Exit(1)
	}
}
