package worker

import (
	"testing"
	"time"
)

func TestResolveProcessingDelay(t *testing.T) {
	tests := []struct {
		name    string
		seconds int
		want    time.Duration
	}{
		{name: "configured delay", seconds: 3, want: 3 * time.Second},
		{name: "default delay", seconds: 0, want: 10 * time.Second},
		{name: "negative delay uses default", seconds: -1, want: 10 * time.Second},
	}

	for _, tt := range tests {
		if got := ResolveProcessingDelay(tt.seconds); got != tt.want {
			t.Fatalf("%s: expected %s, got %s", tt.name, tt.want, got)
		}
	}
}
