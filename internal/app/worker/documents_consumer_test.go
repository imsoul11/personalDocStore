package worker

import "testing"

func TestResolveProcessedDir(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "configured path", in: "/tmp/processed", want: "/tmp/processed"},
		{name: "trim whitespace", in: "  ./custom/processed  ", want: "./custom/processed"},
		{name: "default path", in: "", want: "./storage/processed"},
	}

	for _, tt := range tests {
		if got := resolveProcessedDir(tt.in); got != tt.want {
			t.Fatalf("%s: expected %q, got %q", tt.name, tt.want, got)
		}
	}
}
