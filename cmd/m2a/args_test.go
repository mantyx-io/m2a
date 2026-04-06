package main

import (
	"reflect"
	"testing"
)

func TestReservedPositional(t *testing.T) {
	if !isReservedPositional("debug") || !isReservedPositional("DEBUG") || !isReservedPositional(" raw ") {
		t.Fatal("expected debug/raw reserved")
	}
	if isReservedPositional("http://localhost") {
		t.Fatal("URL should not be reserved")
	}
	if got := reservedFlagHint("debug"); got != "-debug" {
		t.Fatalf("got %q", got)
	}
}

func TestReorderFlagsBeforePositionals(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "url then debug",
			in:   []string{"http://localhost:8080", "-debug"},
			want: []string{"-debug", "http://localhost:8080"},
		},
		{
			name: "url then version flag",
			in:   []string{"http://localhost:8080", "-version"},
			want: []string{"-version", "http://localhost:8080"},
		},
		{
			name: "debug then url unchanged",
			in:   []string{"-debug", "http://localhost:8080"},
			want: []string{"-debug", "http://localhost:8080"},
		},
		{
			name: "H with value then url",
			in:   []string{"http://x", "-H", "A: b"},
			want: []string{"-H", "A: b", "http://x"},
		},
		{
			name: "double dash",
			in:   []string{"-debug", "--", "http://x", "-not-a-flag"},
			want: []string{"-debug", "http://x", "-not-a-flag"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reorderFlagsBeforePositionals(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}
