// Command recipe2json converts an AutomationSequence text proto recipe to the
// JSON accepted by grpcurl for HeadlessBrowserService/RunAutomation.
//
// Usage:
//
//	recipe2json recipes/screenshot_after_load.textproto
package main

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"

	pb "github.com/accretional/chromerpc/proto/cdp/headlessbrowser"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: recipe2json <file.textproto>")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(1)
	}
	var seq pb.AutomationSequence
	if err := prototext.Unmarshal(data, &seq); err != nil {
		fmt.Fprintln(os.Stderr, "parse textproto:", err)
		os.Exit(1)
	}
	out, err := protojson.Marshal(&seq)
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal json:", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}
