package contracttest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v3"
)

type manifest struct {
	Version  int       `yaml:"version"`
	Fixtures []fixture `yaml:"fixtures"`
}

type fixture struct {
	ID        string   `yaml:"id"`
	File      string   `yaml:"file"`
	Textproto string   `yaml:"textproto"`
	Message   string   `yaml:"message"`
	Payload   string   `yaml:"payload"`
	Direction string   `yaml:"direction"`
	Purpose   string   `yaml:"purpose"`
	RoundTrip string   `yaml:"round_trip"`
	Expected  string   `yaml:"expected"`
	Tags      []string `yaml:"tags"`
}

type expectedFile struct {
	Message    string      `yaml:"message"`
	Payload    string      `yaml:"payload"`
	Assertions []assertion `yaml:"assertions"`
}

type assertion struct {
	Path     string        `yaml:"path"`
	Equals   interface{}   `yaml:"equals"`
	Contains []interface{} `yaml:"contains"`
	Length   *int          `yaml:"length"`
	Present  *bool         `yaml:"present"`
	Absent   *bool         `yaml:"absent"`
}

type resolvedValue struct {
	Value   protoreflect.Value
	Field   protoreflect.FieldDescriptor
	Present bool
}

func TestContractGoldenFixtures(t *testing.T) {
	root := fixtureRoot()
	manifest := loadYAML[manifest](t, filepath.Join(root, "manifest.yaml"))
	if manifest.Version != 1 {
		t.Fatalf("manifest version = %d, want 1", manifest.Version)
	}

	for _, fx := range manifest.Fixtures {
		t.Run(fx.ID, func(t *testing.T) {
			msg := newMessage(t, fx.Message)
			data, err := os.ReadFile(filepath.Join(root, fx.File))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			if err := proto.Unmarshal(data, msg); err != nil {
				t.Fatalf("decode fixture: %v", err)
			}

			assertPayload(t, msg, fx.Payload)
			expected := loadYAML[expectedFile](t, filepath.Join(root, fx.Expected))
			if expected.Message != fx.Message || expected.Payload != fx.Payload {
				t.Fatalf("expected metadata mismatch: got %s/%s want %s/%s", expected.Message, expected.Payload, fx.Message, fx.Payload)
			}
			assertMessage(t, msg, expected.Assertions)

			encoded, err := proto.Marshal(msg)
			if err != nil {
				t.Fatalf("marshal round trip: %v", err)
			}
			second := newMessage(t, fx.Message)
			if err := proto.Unmarshal(encoded, second); err != nil {
				t.Fatalf("decode round trip: %v", err)
			}
			assertPayload(t, second, fx.Payload)
			assertMessage(t, second, expected.Assertions)
			if fx.RoundTrip == "byte_exact" && !bytes.Equal(encoded, data) {
				t.Fatalf("byte-exact round trip changed fixture bytes")
			}
		})
	}
}

func fixtureRoot() string {
	if override := os.Getenv("TERMINALS_CONTRACT_FIXTURE_ROOT"); override != "" {
		return override
	}
	return filepath.Join("..", "..", "..", "api", "testdata", "contract")
}

func loadYAML[T any](t *testing.T, path string) T {
	t.Helper()
	var out T
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := yaml.Unmarshal(data, &out); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return out
}

func newMessage(t *testing.T, typeName string) proto.Message {
	t.Helper()
	switch typeName {
	case "terminals.control.v1.ConnectRequest":
		return &controlv1.ConnectRequest{}
	case "terminals.control.v1.ConnectResponse":
		return &controlv1.ConnectResponse{}
	case "terminals.control.v1.WireEnvelope":
		return &controlv1.WireEnvelope{}
	default:
		t.Fatalf("unsupported contract message type %q", typeName)
		return nil
	}
}

func assertPayload(t *testing.T, msg proto.Message, want string) {
	t.Helper()
	oneofs := msg.ProtoReflect().Descriptor().Oneofs()
	payload := oneofs.ByName("payload")
	if payload == nil {
		t.Fatalf("%s has no payload oneof", msg.ProtoReflect().Descriptor().FullName())
	}
	gotField := msg.ProtoReflect().WhichOneof(payload)
	if gotField == nil {
		t.Fatalf("payload oneof is unset, want %s", want)
	}
	if got := string(gotField.Name()); got != want {
		t.Fatalf("payload = %s, want %s", got, want)
	}
}

func assertMessage(t *testing.T, msg proto.Message, assertions []assertion) {
	t.Helper()
	for _, assertion := range assertions {
		got, err := valueAtPath(msg.ProtoReflect(), assertion.Path)
		if err != nil {
			t.Fatalf("%s: %v", assertion.Path, err)
		}
		if assertion.Present != nil {
			if got.Present != *assertion.Present {
				t.Fatalf("%s present = %t, want %t", assertion.Path, got.Present, *assertion.Present)
			}
			continue
		}
		if assertion.Absent != nil {
			if absent := !got.Present; absent != *assertion.Absent {
				t.Fatalf("%s absent = %t, want %t", assertion.Path, absent, *assertion.Absent)
			}
			continue
		}
		if assertion.Length != nil {
			if got.Value.List().Len() != *assertion.Length {
				t.Fatalf("%s length = %d, want %d", assertion.Path, got.Value.List().Len(), *assertion.Length)
			}
			continue
		}
		if len(assertion.Contains) > 0 {
			for _, want := range assertion.Contains {
				if !listContains(got, fmt.Sprint(want)) {
					t.Fatalf("%s does not contain %q; got %v", assertion.Path, want, got.Value.Interface())
				}
			}
			continue
		}
		if !scalarEquals(got, assertion.Equals) {
			t.Fatalf("%s = %v, want %v", assertion.Path, got.Value.Interface(), assertion.Equals)
		}
	}
}

var pathSegmentRE = regexp.MustCompile(`^([a-z0-9_]+)(?:\[([A-Za-z0-9_.:-]+)\])?$`)

func valueAtPath(msg protoreflect.Message, path string) (resolvedValue, error) {
	var current protoreflect.Value
	var currentField protoreflect.FieldDescriptor
	present := true
	current = protoreflect.ValueOfMessage(msg)
	for _, segment := range strings.Split(path, ".") {
		match := pathSegmentRE.FindStringSubmatch(segment)
		if match == nil {
			return resolvedValue{}, fmt.Errorf("unsupported path segment %q", segment)
		}
		m := current.Message()
		field := m.Descriptor().Fields().ByName(protoreflect.Name(match[1]))
		if field == nil {
			oneof := m.Descriptor().Oneofs().ByName(protoreflect.Name(match[1]))
			if oneof == nil {
				return resolvedValue{}, fmt.Errorf("unknown field %q on %s", match[1], m.Descriptor().FullName())
			}
			chosen := m.WhichOneof(oneof)
			if chosen == nil {
				return resolvedValue{Present: false}, nil
			}
			current = protoreflect.ValueOfString(string(chosen.Name()))
			currentField = nil
			present = true
			continue
		}
		currentField = field
		current = m.Get(field)
		switch {
		case field.IsMap():
			present = current.Map().Len() > 0
		case field.IsList():
			present = current.List().Len() > 0
		default:
			present = m.Has(field)
		}
		if selector := match[2]; selector != "" {
			switch {
			case field.IsList():
				index, err := strconv.Atoi(selector)
				if err != nil {
					return resolvedValue{}, fmt.Errorf("list index %q for field %q: %w", selector, match[1], err)
				}
				list := current.List()
				if index >= list.Len() {
					return resolvedValue{}, fmt.Errorf("index %d out of range for %q", index, match[1])
				}
				current = list.Get(index)
				present = true
			case field.IsMap():
				key, err := mapKey(field.MapKey(), selector)
				if err != nil {
					return resolvedValue{}, fmt.Errorf("%q map key %q: %w", match[1], selector, err)
				}
				value := current.Map().Get(key)
				if !value.IsValid() {
					return resolvedValue{Field: field, Present: false}, nil
				}
				current = value
				present = true
			default:
				return resolvedValue{}, fmt.Errorf("field %q is not indexable", match[1])
			}
		}
	}
	return resolvedValue{Value: current, Field: currentField, Present: present}, nil
}

func mapKey(field protoreflect.FieldDescriptor, raw string) (protoreflect.MapKey, error) {
	switch field.Kind() {
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(raw).MapKey(), nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		value, err := strconv.ParseInt(raw, 10, 32)
		return protoreflect.ValueOfInt32(int32(value)).MapKey(), err
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		value, err := strconv.ParseInt(raw, 10, 64)
		return protoreflect.ValueOfInt64(value).MapKey(), err
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		value, err := strconv.ParseUint(raw, 10, 32)
		return protoreflect.ValueOfUint32(uint32(value)).MapKey(), err
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		value, err := strconv.ParseUint(raw, 10, 64)
		return protoreflect.ValueOfUint64(value).MapKey(), err
	case protoreflect.BoolKind:
		value, err := strconv.ParseBool(raw)
		return protoreflect.ValueOfBool(value).MapKey(), err
	case protoreflect.EnumKind,
		protoreflect.FloatKind,
		protoreflect.DoubleKind,
		protoreflect.BytesKind,
		protoreflect.MessageKind,
		protoreflect.GroupKind:
		return protoreflect.MapKey{}, fmt.Errorf("unsupported key kind %s", field.Kind())
	default:
		return protoreflect.MapKey{}, fmt.Errorf("unsupported key kind %s", field.Kind())
	}
}

func scalarEquals(got resolvedValue, want interface{}) bool {
	switch v := got.Value.Interface().(type) {
	case protoreflect.EnumNumber:
		desc := got.Field.Enum().Values().ByNumber(v)
		return (desc != nil && string(desc.Name()) == fmt.Sprint(want)) || fmt.Sprint(v) == fmt.Sprint(want)
	default:
		return fmt.Sprint(got.Value.Interface()) == fmt.Sprint(want)
	}
}

func listContains(got resolvedValue, want string) bool {
	list := got.Value.List()
	for i := 0; i < list.Len(); i++ {
		item := list.Get(i)
		if enum, ok := item.Interface().(protoreflect.EnumNumber); ok {
			desc := got.Field.Enum().Values().ByNumber(enum)
			if desc != nil && string(desc.Name()) == want {
				return true
			}
		}
		if fmt.Sprint(item.Interface()) == want {
			return true
		}
	}
	return false
}
