package main

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	"log"
	"os"
	"path/filepath"
	"sync"

	"code.google.com/p/draw2d/draw2d" //FIXME: what the license? it claims to use AGG...
	"code.google.com/p/go.exp/fsnotify"
	//"github.com/howeyc/fsnotify"
	"github.com/andlabs/ui"

	"github.com/akavel/draq/util"
)

func main() {
	err := ui.Go(main2)
	if err != nil {
		log.Println("GUI error:", err.Error())
	}
}

func main2() {
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

type BlitArea struct {
	m   sync.Mutex
	img *image.RGBA
}

func (*BlitArea) Mouse(e ui.MouseEvent) (repaint bool) { return false }
func (*BlitArea) Key(e ui.KeyEvent) (repaint bool)     { return false }
func (a *BlitArea) Paint(region image.Rectangle) *image.RGBA {
	a.m.Lock()
	img := a.img
	a.m.Unlock()

	b := img.Bounds()
	if region.Max.X <= b.Max.X && region.Max.Y <= b.Max.Y {
		return img.SubImage(region).(*image.RGBA)
	}

	fake := image.NewRGBA(region)
	draw.Draw(fake, region, img, image.ZP, draw.Over) //FIXME: ok or not?
	return fake
}

//NOTE: drawing on 'img' is not safe afterwards till next Swap()
func (a *BlitArea) Swap(img *image.RGBA) *image.RGBA {
	a.m.Lock()
	old := a.img
	a.img = img
	a.m.Unlock()

	return old
}

func painter(fn string, signal chan struct{}, exit chan<- struct{}) {
	defer func() { exit <- struct{}{} }()

	RECT := image.Rectangle{image.Pt(0, 0), image.Pt(640, 480)}

	area := BlitArea{
		img: image.NewRGBA(RECT),
	}
	warea := ui.NewArea(640, 480, &area)

	abs, err := filepath.Abs(fn)
	if err != nil {
		abs = fn
	}

	win := ui.NewWindow(abs+" - draq", 640, 480)
	ui.AppQuit = win.Closing // treat quitting the application like closing the main window
	win.Open(warea)

	buf := image.NewRGBA(RECT)
	for {
		select {
		case <-win.Closing:
			return
		case <-signal:
		}
		log.Println("repaint!")

		err := paint(&buf, fn)
		if err != nil {
			log.Println("error:", err.Error())
			continue
		}

		size := buf.Bounds()
		buf = area.Swap(buf)
		warea.SetSize(size.Max.X, size.Max.Y) // calls RepaintAll() undercover
	}
}

// raise signal, unless already raised
func raise(signal chan<- struct{}) {
	select {
	case signal <- struct{}{}:
	default:
	}
}

func paint(img **image.RGBA, fn string) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	p := util.Parser{
		Scanner: util.NewScanner(f),
	}

	bad := func(t string, args ...interface{}) error {
		return fmt.Errorf("%d: %s", p.Scanner.Offset, fmt.Sprintf(t, args...))
	}

	//buf := image.NewRGBA(img.Bounds())
	buf := *img
	g := draw2d.NewGraphicContext(buf)

	//tmp := image.NewRGBA(region)

	var lastpath *draw2d.PathStorage
	for {
		t, eof := p.Cmd()
		if eof {
			break
		}
		//log.Printf(" TOKEN '%s'\n", s.TokenText())
		switch t {
		case "bg", "fg":
			c, err := p.Color()
			if err != nil {
				return bad(t + " COLOR: " + err.Error())
			}
			if t == "bg" {
				g.SetFillColor(c)
			} else {
				g.SetStrokeColor(c)
			}
		case "clear":
			g.Clear()

		case "mv":
			x, y, err := p.Point()
			if err != nil {
				return bad(t + " " + err.Error())
			}
			g.MoveTo(x, y)
		case "ln":
			x, y, err := p.Point()
			if err != nil {
				return bad(t + " " + err.Error())
			}
			g.LineTo(x, y)
		case "qd":
			x1, y1, err := p.Point()
			if err != nil {
				return bad(t + " POINT 1: " + err.Error())
			}
			x2, y2, err := p.Point()
			if err != nil {
				return bad(t + " POINT 2: " + err.Error())
			}
			g.QuadCurveTo(x1, y1, x2, y2)
		case "cb":
			x1, y1, err := p.Point()
			if err != nil {
				return bad(t + " POINT 1: " + err.Error())
			}
			x2, y2, err := p.Point()
			if err != nil {
				return bad(t + " POINT 2: " + err.Error())
			}
			x3, y3, err := p.Point()
			if err != nil {
				return bad(t + " POINT 3: " + err.Error())
			}
			g.CubicCurveTo(x1, y1, x2, y2, x3, y3)

		case "stroke", "fill":
			if g.Current.Path.IsEmpty() {
				g.Current.Path = lastpath
			} else {
				lastpath = g.Current.Path.Copy()
			}
			switch t {
			case "stroke":
				g.Stroke()
			case "fill":
				g.Fill()
			}

		default:
			return bad("unknown command '%s'", t)
		}
	}

	return nil
}
