package cgroups

import (
	"fmt"
	"testing"

	"github.com/ntk148v/koker/pkg/constants"
)

func TestCGModeString(t *testing.T) {
	tests := []struct {
		mode     CGMode
		expected string
	}{
		{Legacy, "Legacy"},
		{Hybrid, "Hybrid"},
		{Unified, "Unified"},
		{Unavailable, "Unavailable"},
		{CGMode(999), "Unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.mode.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.mode.String())
			}
		})
	}
}

func TestCgroupsV2CpuFormat(t *testing.T) {
	cpus := 1.5
	cpuVal := fmt.Sprintf("%d %d", int(cpus*constants.DefaultCfsPeriod), constants.DefaultCfsPeriod)
	expected := "150000 100000"
	if cpuVal != expected {
		t.Errorf("expected %q, got %q", expected, cpuVal)
	}
}
