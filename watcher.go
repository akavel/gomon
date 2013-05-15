package main

import (
	"errors"
	"fmt"
	"github.com/howeyc/fsnotify"
	"log"
	"os"
)

type ExcluderFunc func(subpath string) bool

type Watcher struct {
	*Latch
	Quit     <-chan bool
	Excluder ExcluderFunc
}

func processSubdir(watcher *fsnotify.Watcher, ev *fsnotify.FileEvent) {
	if ev.IsModify() {
		return
	}
	if ev.IsDelete() {
		log.Println("remove watch", ev.Name)
		// FIXME: what to do with err?
		watcher.RemoveWatch(ev.Name)
		return
	}
	// FIXME: Lstat or Stat?
	// TODO: what to do with err? can we safely ignore?
	mode, err := os.Lstat(ev.Name)
	if err != nil {
		log.Println("error processing subdir:", err.Error())
		return
	}
	if !mode.IsDir() {
		return
	}

	// FIXME: handle renames
	if ev.IsCreate() {
		log.Println("add watch", ev.Name)
		// FIXME: what to do with err?
		watcher.Watch(ev.Name)
	}
}

func (w *Watcher) Start(path string) (err2 error) {
	if w.Latch != nil {
		return errors.New("already initialized")
	}

	defer func() {
		if x, ok := err2.(*os.SyscallError); ok {
			err2 = x.Err
		}
		if os.IsNotExist(err2) {
			// FIXME: use correct path below, related to actual error
			err2 = fmt.Errorf("not found: %s: %s", path, err2.Error())
		}
	}()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	w.Latch = NewLatch()
	quit := make(chan bool, 1)
	w.Quit = quit
	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				if w.Excluder != nil && w.Excluder(ev.Name) {
					break
				}
				w.Latch.Open()
				processSubdir(watcher, ev)
				if ev.IsDelete() && ev.Name == path {
					//TODO: check if this works and recognizes dir removal
					quit <- true
					w.Latch.Open()
					err = watcher.Close()
					if err != nil {
						log.Println("error:", err)
					}
					return
				}
			case err := <-watcher.Error:
				log.Println("error:", err)
			}
		}
	}()

	subfolders := Subfolders(path)
	for _, f := range subfolders {
		err = watcher.Watch(f)
		if err != nil {
			err3 := watcher.Close()
			if err3 != nil {
				log.Println("watcher closing error:", err3.Error())
			}
			return err
		}
	}
	return nil
}
