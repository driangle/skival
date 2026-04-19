package color

import (
	"testing"
)

func TestWrapAddsCodesWhenEnabled(t *testing.T) {
	SetEnabled(true)
	defer SetEnabled(false)

	tests := []struct {
		fn   func(string) string
		name string
		code string
	}{
		{Green, "Green", "\033[32m"},
		{Red, "Red", "\033[31m"},
		{Yellow, "Yellow", "\033[33m"},
		{Cyan, "Cyan", "\033[36m"},
		{Dim, "Dim", "\033[2m"},
	}

	for _, tt := range tests {
		got := tt.fn("hello")
		want := tt.code + "hello" + "\033[0m"
		if got != want {
			t.Errorf("%s(\"hello\") = %q, want %q", tt.name, got, want)
		}
	}
}

func TestWrapReturnsPlainWhenDisabled(t *testing.T) {
	SetEnabled(false)

	fns := []func(string) string{Green, Red, Yellow, Cyan, Dim}
	for _, fn := range fns {
		if got := fn("hello"); got != "hello" {
			t.Errorf("expected plain string when disabled, got %q", got)
		}
	}
}

func TestWrapEmptyString(t *testing.T) {
	SetEnabled(true)
	defer SetEnabled(false)

	if got := Green(""); got != "" {
		t.Errorf("Green(\"\") = %q, want empty", got)
	}
}

func TestFormatFunctions(t *testing.T) {
	SetEnabled(true)
	defer SetEnabled(false)

	got := Greenf("count: %d", 42)
	want := "\033[32mcount: 42\033[0m"
	if got != want {
		t.Errorf("Greenf = %q, want %q", got, want)
	}

	got = Dimf("$%.4f", 0.0312)
	want = "\033[2m$0.0312\033[0m"
	if got != want {
		t.Errorf("Dimf = %q, want %q", got, want)
	}
}
