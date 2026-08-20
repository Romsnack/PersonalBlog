// Package serve runs a local preview server that rebuilds on change and
// reloads the browser.
package serve

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

// reloader fans one "rebuilt" signal out to every connected browser tab over
// server-sent events.
type reloader struct {
	mu      sync.Mutex
	clients map[chan struct{}]bool
}

func newReloader() *reloader {
	return &reloader{clients: map[chan struct{}]bool{}}
}

func (r *reloader) subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	r.mu.Lock()
	r.clients[ch] = true
	r.mu.Unlock()
	return ch
}

func (r *reloader) unsubscribe(ch chan struct{}) {
	r.mu.Lock()
	delete(r.clients, ch)
	r.mu.Unlock()
}

func (r *reloader) broadcast() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for ch := range r.clients {
		select {
		case ch <- struct{}{}:
		default: // a reload is already queued for this tab
		}
	}
}

func (r *reloader) handler(w http.ResponseWriter, req *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	ch := r.subscribe()
	defer r.unsubscribe(ch)

	for {
		select {
		case <-req.Context().Done():
			return
		case <-ch:
			fmt.Fprint(w, "data: reload\n\n")
			flusher.Flush()
		}
	}
}

// noCache stops the browser from serving a stale page after a rebuild, which
// would otherwise make live reload look broken.
func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		h.ServeHTTP(w, r)
	})
}

// Run serves outDir on addr and rebuilds via rebuild whenever a file under one
// of watchDirs changes. It blocks until the server exits.
func Run(addr, outDir string, watchDirs []string, rebuild func() error) error {
	r := newReloader()

	stop, err := watch(watchDirs, func() {
		if err := rebuild(); err != nil {
			log.Printf("rebuild failed: %v", err)
			return
		}
		log.Println("rebuilt")
		r.broadcast()
	})
	if err != nil {
		return err
	}
	defer stop()

	mux := http.NewServeMux()
	mux.HandleFunc("/_livereload", r.handler)
	mux.Handle("/", noCache(http.FileServer(http.Dir(outDir))))

	log.Printf("serving %s on http://localhost%s", outDir, addr)
	return http.ListenAndServe(addr, mux)
}
