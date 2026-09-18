package pricing

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// AtomicPriceTable holds a PriceTable that can be swapped atomically.
// Readers see a consistent snapshot; Swap is atomic.
type AtomicPriceTable struct {
	v atomic.Pointer[PriceTable]
}

// NewAtomicPriceTable returns an empty AtomicPriceTable.
func NewAtomicPriceTable() *AtomicPriceTable {
	return &AtomicPriceTable{}
}

// Get returns the current PriceTable, or nil if none has been set.
func (a *AtomicPriceTable) Get() *PriceTable {
	if a == nil {
		return nil
	}
	return a.v.Load()
}

// Swap replaces the current table with pt.
func (a *AtomicPriceTable) Swap(pt *PriceTable) {
	if a == nil || pt == nil {
		return
	}
	a.v.Store(pt)
}

// Fetch retrieves pricing JSON from url, parses it, and atomically swaps it
// into the table. On any error (network, parse, validation), the existing
// table is left untouched. Uses client.Timeout for the per-request deadline.
func (a *AtomicPriceTable) Fetch(ctx context.Context, url string, client *http.Client) error {
	if url == "" {
		return fmt.Errorf("empty pricing URL")
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("fetching %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10 MiB cap
	if err != nil {
		return fmt.Errorf("reading body: %w", err)
	}
	pt, err := LoadPrices(strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("parsing fetched prices: %w", err)
	}
	a.Swap(pt)
	return nil
}