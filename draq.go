package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"text/scanner"

	"code.google.com/p/draw2d/draw2d" //FIXME: what the license? it claims to use AGG...
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

	exit := make(chan struct{}, 1)
	repaint := make(chan struct{}, 1)
	// FIXME: handle errors in painter()
	go painter(fn, repaint, exit)
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

			raise(repaint)
		case err := <-watcher.Error:
			log.Println("error:", err)
			watcher.Close()
			return
		case <-exit:
			watcher.Close()
			return
		}
	}
}

func painter(fn string, signal chan struct{}, exit chan<- struct{}) {
	defer func() { exit <- struct{}{} }()

	var sizelock sync.Mutex
	w, h := 640, 480

	win, err := wde.NewWindow(w, h) // FIXME: set w&h via commandline flags
	if err != nil {
		panic(err)
	}
	abs, err := filepath.Abs(fn)
	if err != nil {
		abs = fn
	}
	win.SetTitle(abs + " - draq")
	win.Show()

	go func() {
		for {
			e := <-win.EventChan()
			switch e := e.(type) {
			case wde.CloseEvent:
				exit <- struct{}{}
				return
			case wde.ResizeEvent:
				sizelock.Lock()
				w, h = e.Width, e.Height
				sizelock.Unlock()
				raise(signal)
			}
		}
	}()

	for {
		<-signal
		log.Println("repaint!")
		err := paint(win.Screen(), fn)
		if err != nil {
			log.Println("error:", err.Error())
			continue
		}
		win.FlushImage(image.Rectangle{Max: image.Point{w, h}})
	}
}

// raise signal, unless already raised
func raise(signal chan<- struct{}) {
	select {
	case signal <- struct{}{}:
	default:
	}
}

func paint(img wde.Image, fn string) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	s := scanner.Scanner{}
	s.Init(f)

	bad := func(t string, args ...interface{}) error {
		return fmt.Errorf("%d:%d: %s", s.Line, s.Column, fmt.Sprintf(t, args...))
	}

	g := draw2d.NewGraphicContext(img)

	for {
		t := s.Scan()
		if t == scanner.EOF {
			return nil
		}
		//log.Printf(" TOKEN '%s'\n", s.TokenText())
		switch t := s.TokenText(); t {
		case "bg", "fg":
			s.Scan()
			val := s.TokenText()
			if len(val) != 6 && len(val) != 8 {
				return bad(t + " COLOR: COLOR must be RRGGBB or RRGGBBAA")
			}
			c, err := strconv.ParseInt(val, 16, 32)
			if err != nil {
				return bad(t+" COLOR: %s", err.Error())
			}
			uc := uint32(c)
			if len(val) == 6 {
				uc = uc << 8
			}
			cc := color.RGBA{
				R: uint8(uc >> 24),
				G: uint8(uc >> 16),
				B: uint8(uc >> 8),
				A: uint8(uc),
			}
			if t == "bg" {
				g.SetFillColor(cc)
			} else {
				g.SetStrokeColor(cc)
			}
		case "clear":
			g.Clear()
		default:
			return bad("unknown command '%s'", t)
		}
	}
}
