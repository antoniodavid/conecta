package network

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

// SpeedTest runs a download speed test
type SpeedTest struct {
	client *http.Client
	urls   []string
}

// NewSpeedTest creates a new speed test handler
func NewSpeedTest() *SpeedTest {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
		},
	}

	return &SpeedTest{
		client: client,
		urls: []string{
			"https://speed.cloudflare.com/__down?bytes=10000000", // 10MB
			"http://speedtest.tele2.net/10MB.zip",
			"https://proof.ovh.net/files/10Mb.dat",
		},
	}
}

// NewSpeedTestWithURLs creates a speed test with custom URLs
func NewSpeedTestWithURLs(urls []string) *SpeedTest {
	st := NewSpeedTest()
	st.urls = urls
	return st
}

// Run executes the speed test
func (st *SpeedTest) Run() *SpeedResult {
	result := &SpeedResult{
		BytesDownload: 0,
		Duration:      0,
	}

	start := time.Now()
	totalBytes := 0

	var lastErr error
	for _, u := range st.urls {
		resp, err := st.client.Get(u)
		if err != nil {
			lastErr = err
			continue
		}

		buf := make([]byte, 32*1024)
		for {
			n, err := resp.Body.Read(buf)
			totalBytes += n
			if err != nil {
				break
			}
		}
		resp.Body.Close()
		break // Success, stop trying
	}

	elapsed := time.Since(start)
	result.Duration = elapsed
	result.BytesDownload = uint64(totalBytes)

	if totalBytes == 0 {
		result.Error = fmt.Errorf("no data downloaded: %w", lastErr)
		return result
	}

	// Calculate speed in Mbps (megabits per second)
	seconds := elapsed.Seconds()
	if seconds > 0 {
		result.DownloadMbps = float64(totalBytes) * 8 / seconds / 1_000_000
	}

	return result
}

// FormatSpeed formats speed result for display
func (r *SpeedResult) FormatSpeed() string {
	if r.Error != nil {
		return fmt.Sprintf("Error: %v", r.Error)
	}

	mbDownloaded := float64(r.BytesDownload) / 1_000_000
	return fmt.Sprintf("Download: %.1f Mbps (%.1f MB/s) — %.1f MB en %.1fs",
		r.DownloadMbps,
		r.DownloadMbps/8,
		mbDownloaded,
		r.Duration.Seconds(),
	)
}
