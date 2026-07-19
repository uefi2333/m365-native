package outbound

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
)

func (p *Pool) Check(ctx context.Context, raw string) (time.Duration, error) {
	p.mu.Lock()
	var e *poolEntry
	for _, item := range p.entries {
		if item.raw == raw {
			e = item
			break
		}
	}
	p.mu.Unlock()
	if e == nil {
		return 0, http.ErrNoLocation
	}
	target := os.Getenv("M365_PROXY_HEALTH_URL")
	if target == "" {
		target = "https://www.msftconnecttest.com/connecttest.txt"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return 0, err
	}
	start := time.Now()
	resp, err := e.clients.HTTP.Do(req)
	latency := time.Since(start)
	if err != nil {
		p.mark(raw, err)
		return latency, err
	}
	status := resp.Status
	resp.Body.Close()
	// Any HTTP response proves the proxy connection and HTTP exchange worked.
	// 5xx is reported separately as an upstream/gateway response, not a dial failure.
	if resp.StatusCode >= 500 {
		p.mark(raw, nil)
		return latency, fmt.Errorf("proxy reachable; upstream returned %s", status)
	}
	p.mark(raw, nil)
	return latency, nil
}
func (p *Pool) CheckAll(ctx context.Context) []map[string]any {
	p.mu.Lock()
	raws := make([]string, 0, len(p.entries))
	for _, e := range p.entries {
		raws = append(raws, e.raw)
	}
	p.mu.Unlock()
	for _, raw := range raws {
		c, cancel := context.WithTimeout(ctx, 10*time.Second)
		latency, err := p.Check(c, raw)
		cancel()
		p.mu.Lock()
		for _, e := range p.entries {
			if e.raw == raw {
				e.lastCheck = time.Now()
				e.latency = latency
				e.lastError = ""
				if err != nil {
					e.lastError = err.Error()
				}
				break
			}
		}
		p.mu.Unlock()
	}
	return p.List()
}
