package shellgen

import (
	"strings"
	"testing"

	"github.com/cheatmd-dev/cheatmd/pkg/config"
)

func setTestKeyWidget(t *testing.T, key string) {
	t.Helper()
	oldConfig := *config.Get()
	config.Get().KeyWidget = key
	t.Cleanup(func() {
		*config.Get() = oldConfig
	})
}

func TestBashWidget(t *testing.T) {
	setTestKeyWidget(t, "\\C-g")
	out := BashWidget()

	if !strings.Contains(out, `bind -x '"\C-g": _cheatmd_widget'`) {
		t.Errorf("BashWidget did not contain expected bind command, got:\n%s", out)
	}

	if !strings.Contains(out, `output="$(cheatmd --print)"`) {
		t.Errorf("BashWidget missing core execution command")
	}

	// Test Injection edge case (no escaping implies whatever we inject will just appear)
	// We just want to ensure it inserts it directly.
	setTestKeyWidget(t, `"; rm -rf /; "`)
	out = BashWidget()
	if !strings.Contains(out, `bind -x '""; rm -rf /; "": _cheatmd_widget'`) {
		t.Errorf("BashWidget did not inject keybinding as expected, got:\n%s", out)
	}
}

func TestZshWidget(t *testing.T) {
	setTestKeyWidget(t, "\\C-x")
	out := ZshWidget()

	// should translate \C-x to ^x
	if !strings.Contains(out, `bindkey '^x' _cheatmd_widget`) {
		t.Errorf("ZshWidget did not contain expected bindkey command, got:\n%s", out)
	}

	if !strings.Contains(out, `output="$(cheatmd --print --match "$input")"`) {
		t.Errorf("ZshWidget missing core execution command")
	}
}

func TestFishWidget(t *testing.T) {
	setTestKeyWidget(t, "\\C-f")
	out := FishWidget()

	// should translate \C-f to \cf
	if !strings.Contains(out, `bind \cf _cheatmd_widget`) {
		t.Errorf("FishWidget did not contain expected bind command, got:\n%s", out)
	}
}

func TestConvertToZshKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"\\C-g", "^g"},
		{"\\C-X", "^x"},
		{"^g", "^g"},
		{"alt-c", "alt-c"},
	}

	for _, tt := range tests {
		got := convertToZshKey(tt.input)
		if got != tt.expected {
			t.Errorf("convertToZshKey(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestConvertToFishKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"\\C-g", "\\cg"},
		{"\\C-X", "\\cx"},
		{"\\cg", "\\cg"},
		{"alt-c", "alt-c"},
	}

	for _, tt := range tests {
		got := convertToFishKey(tt.input)
		if got != tt.expected {
			t.Errorf("convertToFishKey(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
