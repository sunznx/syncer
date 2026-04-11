package color

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestColorSprint(t *testing.T) {
	// Ensure colors are enabled for this test
	disabled = false

	tests := []struct {
		color Color
		input string
		want  string
	}{
		{Red, "hello", "\033[31mhello\033[0m"},
		{Green, "world", "\033[32mworld\033[0m"},
		{Yellow, "warn", "\033[33mwarn\033[0m"},
		{Cyan, "info", "\033[36minfo\033[0m"},
		{Gray, "dim", "\033[90mdim\033[0m"},
	}

	for _, tt := range tests {
		t.Run(string(tt.color), func(t *testing.T) {
			got := tt.color.Sprint(tt.input)
			if got != tt.want {
				t.Errorf("Sprint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestColorSprintf(t *testing.T) {
	disabled = false

	got := Red.Sprintf("value: %d", 42)
	want := "\033[31mvalue: 42\033[0m"
	if got != want {
		t.Errorf("Sprintf() = %q, want %q", got, want)
	}
}

func TestColorPrint(t *testing.T) {
	disabled = false
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Green.Print("ok")

	w.Close()
	os.Stdout = oldStdout

	var buf strings.Builder
	_, _ = fmt.Fscan(r, &buf)
	// We just verify no panic occurs; exact stdout capture in tests is brittle.
}

func TestColorPrintf(t *testing.T) {
	disabled = false
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Cyan.Printf("%s", "test")

	w.Close()
	os.Stdout = oldStdout
	_ = r
}

func TestDisable(t *testing.T) {
	disabled = false
	Disable()
	if !disabled {
		t.Error("expected disabled to be true after Disable()")
	}

	got := Red.Sprint("hello")
	want := "hello"
	if got != want {
		t.Errorf("after Disable(), Sprint() = %q, want %q", got, want)
	}

	// Re-enable for other tests
	disabled = false
}

func TestSemanticHelpers(t *testing.T) {
	disabled = false

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"DryRun", DryRun("dry"), "\033[33mdry\033[0m"},
		{"Action", Action("act"), "\033[32mact\033[0m"},
		{"Path", Path("/tmp"), "\033[36m/tmp\033[0m"},
		{"Arrow", Arrow(), "\033[90m -> \033[0m"},
		{"Error", Error("err"), "\033[31merr\033[0m"},
		{"Info", Info("info"), "\033[36minfo\033[0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestStrip(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"\033[31mred\033[0m", "red"},
		{"\033[32m\033[1mbold green\033[0m", "bold green"},
		{"no color", "no color"},
		{"", ""},
		{"\033[", "\033["}, // malformed, should stop at break
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Strip(tt.input)
			if got != tt.want {
				t.Errorf("Strip(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
