package pricing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAtomicPriceTable_GetSet(t *testing.T) {
	a := NewAtomicPriceTable()
	if got := a.Get(); got != nil {
		t.Errorf("Get on empty: got %v, want nil", got)
	}
	pt, err := LoadPrices(strings.NewReader(`{"m1":{"input_cost_per_token":1}}`))
	if err != nil {
		t.Fatal(err)
	}
	a.Swap(pt)
	if got := a.Get(); got != pt {
		t.Errorf("Get after Swap: got %v, want %v", got, pt)
	}
}

func TestFetch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"claude-sonnet-4-5-20250929":{"input_cost_per_token":0.000003}}`))
	}))
	defer srv.Close()

	a := NewAtomicPriceTable()
	client := srv.Client()
	client.Timeout = 2 * time.Second
	if err := a.Fetch(context.Background(), srv.URL, client); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	got := a.Get()
	if got == nil {
		t.Fatal("expected table after successful fetch")
	}
	if _, ok := got.LookupExact("claude-sonnet-4-5-20250929"); !ok {
		t.Error("expected fetched model to be lookup-able")
	}
}

func TestFetch_InvalidJSONKeepsOldTable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	defer srv.Close()

	a := NewAtomicPriceTable()
	old, _ := LoadPrices(strings.NewReader(`{"old-model":{"input_cost_per_token":1}}`))
	a.Swap(old)

	client := srv.Client()
	client.Timeout = 2 * time.Second
	err := a.Fetch(context.Background(), srv.URL, client)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if got := a.Get(); got != old {
		t.Error("expected old table to remain after failed fetch")
	}
}

func TestFetch_TimeoutKeepsOldTable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	a := NewAtomicPriceTable()
	old, _ := LoadPrices(strings.NewReader(`{"old":{"input_cost_per_token":1}}`))
	a.Swap(old)

	client := srv.Client()
	client.Timeout = 50 * time.Millisecond
	err := a.Fetch(context.Background(), srv.URL, client)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if got := a.Get(); got != old {
		t.Error("expected old table to remain after timeout")
	}
}

func TestAtomicPriceTable_ConcurrentReads(t *testing.T) {
	a := NewAtomicPriceTable()
	pt, _ := LoadPrices(strings.NewReader(`{"m1":{"input_cost_per_token":1}}`))
	a.Swap(pt)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = a.Get()
		}()
	}
	wg.Wait()
}