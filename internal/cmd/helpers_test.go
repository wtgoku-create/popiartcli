package cmd

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestIntervalDurationParsesMilliseconds(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("interval", "2000", "")

	interval, err := intervalDuration(cmd, "interval")
	if err != nil {
		t.Fatalf("intervalDuration returned error: %v", err)
	}
	if interval != 2*time.Second {
		t.Fatalf("expected 2s, got %v", interval)
	}
}

func TestIntervalDurationRejectsInvalidValue(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("interval", "abc", "")

	if _, err := intervalDuration(cmd, "interval"); err == nil {
		t.Fatal("expected validation error for invalid interval")
	}
}
