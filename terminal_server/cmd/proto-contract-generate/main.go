package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

type manifest struct {
	Fixtures []fixture `yaml:"fixtures"`
}

type fixture struct {
	File      string `yaml:"file"`
	Textproto string `yaml:"textproto"`
	Message   string `yaml:"message"`
}

func main() {
	manifestPath := flag.String("manifest", "", "path to contract manifest.yaml")
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	if *manifestPath == "" {
		fatalf("--manifest is required")
	}

	data, err := os.ReadFile(*manifestPath)
	if err != nil {
		fatalf("read manifest: %v", err)
	}
	var manifest manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		fatalf("parse manifest: %v", err)
	}
	fixtureRoot := filepath.Join(*root, "api", "testdata", "contract")
	for _, fx := range manifest.Fixtures {
		msg, err := newMessage(fx.Message)
		if err != nil {
			fatalf("%s: %v", fx.Textproto, err)
		}
		text, err := os.ReadFile(filepath.Join(fixtureRoot, fx.Textproto))
		if err != nil {
			fatalf("read %s: %v", fx.Textproto, err)
		}
		if err := prototext.Unmarshal(text, msg); err != nil {
			fatalf("parse %s: %v", fx.Textproto, err)
		}
		bin, err := proto.MarshalOptions{Deterministic: true}.Marshal(msg)
		if err != nil {
			fatalf("marshal %s: %v", fx.Textproto, err)
		}
		if err := os.WriteFile(filepath.Join(fixtureRoot, fx.File), bin, 0o644); err != nil {
			fatalf("write %s: %v", fx.File, err)
		}
	}
}

func newMessage(typeName string) (proto.Message, error) {
	switch typeName {
	case "terminals.control.v1.ConnectRequest":
		return &controlv1.ConnectRequest{}, nil
	case "terminals.control.v1.ConnectResponse":
		return &controlv1.ConnectResponse{}, nil
	case "terminals.control.v1.WireEnvelope":
		return &controlv1.WireEnvelope{}, nil
	default:
		return nil, fmt.Errorf("unsupported contract message type %q", typeName)
	}
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "proto-contract-generate: "+format+"\n", args...)
	os.Exit(1)
}
