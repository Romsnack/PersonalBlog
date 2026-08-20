package serve

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// debounce is how long to wait for the filesystem to settle. Editors write a
// save as several events, and a rebuild per event would be wasteful.
const debounce = 120 * time.Millisecond

// watch calls onChange after activity in any of dirs (recursively) quiets
// down. The returned func stops watching.
func watch(dirs []string, onChange func()) (func(), error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return w.Add(path)
			}
			return nil
		})
		if err != nil {
			w.Close()
			return nil, err
		}
	}

	go func() {
		var timer *time.Timer
		for {
			select {
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				// A new subdirectory has to be watched too, or posts added
				// under it would never trigger a rebuild.
				if ev.Op&fsnotify.Create != 0 {
					if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
						_ = w.Add(ev.Name)
					}
				}
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounce, onChange)
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				log.Printf("watch: %v", err)
			}
		}
	}()

	return func() { w.Close() }, nil
}
