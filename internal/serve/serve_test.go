package serve

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// waitFor polls cond until it holds or the deadline passes. Used instead of a
// fixed sleep so the tests stay fast and do not flake under -race.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func (r *reloader) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.clients)
}

func TestReloaderSubscribeAndUnsubscribe(t *testing.T) {
	r := newReloader()
	if r.count() != 0 {
		t.Fatalf("a new reloader has %d clients, want 0", r.count())
	}

	a, b := r.subscribe(), r.subscribe()
	if r.count() != 2 {
		t.Errorf("got %d clients, want 2", r.count())
	}

	r.unsubscribe(a)
	if r.count() != 1 {
		t.Errorf("got %d clients after unsubscribe, want 1", r.count())
	}
	r.unsubscribe(b)
	if r.count() != 0 {
		t.Errorf("got %d clients, want 0", r.count())
	}
}

func TestBroadcastReachesEverySubscriber(t *testing.T) {
	r := newReloader()
	a, b := r.subscribe(), r.subscribe()

	r.broadcast()

	for name, ch := range map[string]chan struct{}{"a": a, "b": b} {
		select {
		case <-ch:
		default:
			t.Errorf("subscriber %s got no reload signal", name)
		}
	}
}

// The channels are buffered to depth one and broadcast must not block when a
// tab has a reload already queued, or a slow client would stall every rebuild.
func TestBroadcastDoesNotBlockOnAFullChannel(t *testing.T) {
	r := newReloader()
	ch := r.subscribe()

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.broadcast()
		r.broadcast()
		r.broadcast()
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("broadcast blocked when a subscriber already had a signal queued")
	}

	if len(ch) != 1 {
		t.Errorf("queued %d signals, want the channel coalesced to 1", len(ch))
	}
}

func TestBroadcastWithNoSubscribers(t *testing.T) {
	newReloader().broadcast() // must not panic or block
}

func TestLiveReloadHandlerStreamsAReloadEvent(t *testing.T) {
	r := newReloader()
	srv := httptest.NewServer(http.HandlerFunc(r.handler))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", got)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", got)
	}

	// The handler subscribes before it streams; broadcasting earlier would be
	// delivered to nobody.
	waitFor(t, "the handler to subscribe", func() bool { return r.count() == 1 })
	r.broadcast()

	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(line) != "data: reload" {
		t.Errorf("streamed %q, want data: reload", strings.TrimSpace(line))
	}
}

func TestLiveReloadHandlerUnsubscribesWhenTheClientLeaves(t *testing.T) {
	r := newReloader()
	srv := httptest.NewServer(http.HandlerFunc(r.handler))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the handler to subscribe", func() bool { return r.count() == 1 })

	cancel()
	_ = resp.Body.Close()

	waitFor(t, "the handler to unsubscribe", func() bool { return r.count() == 0 })
}

func TestNoCacheSetsNoStore(t *testing.T) {
	h := noCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("page"))
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	if rec.Body.String() != "page" {
		t.Errorf("body = %q, want it passed through", rec.Body.String())
	}
}

func TestWatchFiresOnAChangeAndStops(t *testing.T) {
	dir := t.TempDir()

	changed := make(chan struct{}, 16)
	stop, err := watch([]string{dir}, func() { changed <- struct{}{} })
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "post.md"), []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}

	select {
	case <-changed:
	case <-time.After(10 * time.Second):
		t.Fatal("watch never reported the new file")
	}

	stop()
}

// A watch list naming a directory that does not exist is legal: `serve` passes
// both content/ and static/, and a site may have neither.
func TestWatchSkipsMissingDirectories(t *testing.T) {
	stop, err := watch([]string{filepath.Join(t.TempDir(), "absent")}, func() {})
	if err != nil {
		t.Fatalf("a missing watch dir should not be an error, got %v", err)
	}
	stop()
}
