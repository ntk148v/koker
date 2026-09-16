package filesystem

import (
	"strings"
	"testing"
)

func TestFormatOverlayFsMountOption(t *testing.T) {
	tests := []struct {
		name     string
		lower    []string
		upper    []string
		work     []string
		expected string
	}{
		{
			name:     "read-write overlay",
			lower:    []string{"/layer1", "/layer2"},
			upper:    []string{"/diff"},
			work:     []string{"/work"},
			expected: "lowerdir=/layer1:/layer2,upperdir=/diff,workdir=/work",
		},
		{
			name:     "read-only overlay without upper and work",
			lower:    []string{"/layer1", "/layer2"},
			upper:    nil,
			work:     nil,
			expected: "lowerdir=/layer1:/layer2",
		},
		{
			name:     "read-only overlay with empty slices",
			lower:    []string{"/layer1"},
			upper:    []string{},
			work:     []string{},
			expected: "lowerdir=/layer1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatOverlayFsMountOption(tt.lower, tt.upper, tt.work)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
			if len(tt.upper) == 0 && strings.Contains(result, "upperdir=") {
				t.Errorf("read-only option should not contain upperdir: %s", result)
			}
			if len(tt.work) == 0 && strings.Contains(result, "workdir=") {
				t.Errorf("read-only option should not contain workdir: %s", result)
			}
		})
	}
}
