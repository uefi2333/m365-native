package outbound

import (
	"context"
	"fmt"
	"net/http"
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.msftconnecttest.com/connecttest.txt", nil)
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
	resp.Body.Close()
	if resp.StatusCode >= 500 {
		p.mark(raw, http.ErrServerClosed)
		return latency, fmt.Errorf("health check returned %s", resp.Status)
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
