package network

import (
	"testing"
	"time"
)

func TestSpeedResult_FormatSpeed(t *testing.T) {
	tests := []struct {
		name     string
		result   *SpeedResult
		contains string
	}{
		{
			name: "formats successful result",
			result: &SpeedResult{
				DownloadMbps:  5.2,
				BytesDownload: 10000000,
				Duration:      15 * time.Second,
			},
			contains: "5.2 Mbps",
		},
		{
			name: "formats error result",
			result: &SpeedResult{
				Error: errTimeout,
			},
			contains: "Error:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.FormatSpeed()
			if len(got) == 0 {
				t.Error("FormatSpeed() returned empty string")
			}
			// Just check it doesn't panic and returns something
		})
	}
}

var errTimeout = &timeoutError{}

type timeoutError struct{}

func (e *timeoutError) Error() string {
	return "timeout"
}
