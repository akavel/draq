package main

import (
	"log"

	"github.com/skelterjohn/go.wde"
)

func main() {
	go main2()
	wde.Run()
}

func main2() {
	err := run()
	if err != nil {
		log.Println(err.Error())
	}
	wde.Stop()
}

func run() error {
	return nil
}
