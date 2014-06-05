package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"code.google.com/p/go.exp/fsnotify"
	//"github.com/howeyc/fsnotify"
	"github.com/skelterjohn/go.wde" //FIXME: use a custom fork, based on non-cgo w32 package fork
)

func main() {
	go main2()
	wde.Run()
}

func main2() {
	defer wde.Stop()
	err := run()
	if err != nil {
		log.Println("error:", err.Error())
	}
}

//FIXME: describe the format of FILENAME contents, and file watching behavior
var usage = errors.New(`USAGE: draq FILENAME`)

func run() error {
	if len(os.Args) < 2 {
		return usage
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	fn := os.Args[1]
	log.Printf("Watching '%s' for changes...\n", fn)
	q := make(chan struct{})
	go watch(w, fn, q)
	err = w.Watch(filepath.Dir(fn))
	<-q
	log.Println("Stopped watching.")
	return err
}

func watch(watcher *fsnotify.Watcher, fn string, q chan<- struct{}) {
	defer func() { q <- struct{}{} }()

	repaint := make(chan struct{}, 1)
	// FIXME: handle errors in painter()
	go painter(fn, repaint)
	repaint <- struct{}{}

	for {
		select {
		case ev := <-watcher.Event:
			log.Println("event:", ev)
			if ev.IsDelete() || ev.IsRename() {
				break
			}

			// try to check if that's the watched file
			f1, err := os.Stat(fn)
			if err != nil {
				break
			}
			f2, err := os.Stat(ev.Name)
			if err != nil {
				break
			}
			if !os.SameFile(f1, f2) {
				break
			}

			select { // raise signal that repaint is needed, unless already queued
			case repaint <- struct{}{}:
			default:
			}
		case err := <-watcher.Error:
			log.Println("error:", err)
			watcher.Close()
			return
		}
	}
}

func painter(fn string, signal <-chan struct{}) {
	for {
		<-signal
		log.Println("repaint!")
	}
}
