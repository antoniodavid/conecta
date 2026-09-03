package network

import (
	"fmt"
	"io"
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
	// TLS certificates are verified by default.
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
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

	if len(st.urls) == 0 {
		result.Error = fmt.Errorf("no speed test URLs configured")
		return result
	}

	start := time.Now()

	var lastErr error
	succeeded := false
	totalBytes := 0
	var serverURL string
	for _, u := range st.urls {
		n, urlErr := st.fetchOne(u)
		if urlErr != nil {
			lastErr = urlErr
			continue
		}
		totalBytes = n
		serverURL = u
		succeeded = true
		break
	}

	elapsed := time.Since(start)
	result.Duration = elapsed
	result.BytesDownload = uint64(totalBytes)
	result.ServerURL = serverURL

	if !succeeded {
		if lastErr == nil {
			lastErr = fmt.Errorf("no data downloaded")
		}
		result.Error = lastErr
		return result
	}

	// Calculate speed in Mbps (megabits per second)
	seconds := elapsed.Seconds()
	if seconds > 0 {
		result.DownloadMbps = float64(totalBytes) * 8 / seconds / 1_000_000
	}

	return result
}

// fetchOne downloads a single URL. Only a clean EOF with a 2xx status counts
// as success; bad statuses and truncated bodies are rejected.
func (st *SpeedTest) fetchOne(u string) (int, error) {
	if u == "" {
		return 0, fmt.Errorf("empty speed test URL")
	}
	resp, err := st.client.Get(u)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return 0, fmt.Errorf("speed test bad status: %s", resp.Status)
	}

	buf := make([]byte, 32*1024)
	total := 0
	for {
		n, rerr := resp.Body.Read(buf)
		total += n
		if rerr != nil {
			if rerr == io.EOF {
				break
			}
			// Truncated body or any other read failure is rejected.
			return 0, fmt.Errorf("speed test truncated read: %w", rerr)
		}
	}
	if resp.ContentLength > 0 && int64(total) != resp.ContentLength {
		return 0, fmt.Errorf("speed test partial read: got %d of %d bytes", total, resp.ContentLength)
	}
	if total == 0 {
		return 0, fmt.Errorf("speed test empty body")
	}
	return total, nil
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
