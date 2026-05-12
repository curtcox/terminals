package transport

import (
	"os"
	"path/filepath"
	"testing"

	uiv1 "github.com/curtcox/terminals/terminal_server/gen/go/ui/v1"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
	"google.golang.org/protobuf/proto"
)

func TestIdleMainLayerPlaceholderGoldenWire(t *testing.T) {
	gotNode := descriptorToUINode(ui.IdleMainLayerPlaceholder())
	got, err := proto.Marshal(gotNode)
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "idle_main_layer_placeholder_root.pb")
	want, err := os.ReadFile(golden)
	if err != nil {
		if os.Getenv("UPDATE_IDLE_MAIN_PLACEHOLDER_GOLDEN") == "1" {
			if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(golden, got, 0o644); err != nil {
				t.Fatal(err)
			}
			t.Logf("wrote golden %s (%d bytes)", golden, len(got))
			return
		}
		t.Fatalf("read golden: %v (set UPDATE_IDLE_MAIN_PLACEHOLDER_GOLDEN=1 to write)", err)
	}
	wantNode := &uiv1.Node{}
	if err := proto.Unmarshal(want, wantNode); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(gotNode, wantNode) {
		t.Fatalf("idle placeholder semantic mismatch vs golden; regenerate with UPDATE_IDLE_MAIN_PLACEHOLDER_GOLDEN=1")
	}
}
