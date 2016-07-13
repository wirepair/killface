package main

import (
	"flag"
	"log"
	"time"
)

var mem int
var timeout int

func init() {
	flag.IntVar(&mem, "mem", 3, "how many times to alloc 100mb")
	flag.IntVar(&timeout, "timeout", 15, "how long to wait before exiting")
}

func consume100mb() *[]byte {
	s := make([]byte, 100000000)
	for i := 0; i < 100000000; i++ {
		s[i] = 1
	}
	return &s
}

func main() {
	flag.Parse()
	s := make([]*[]byte, mem)
	for i := 0; i < mem; i++ {
		d := consume100mb()
		if d == nil {
			log.Printf("doh")
		}
		s[i] = d
	}
	t := time.NewTimer(time.Duration(timeout) * time.Second)
	for {
		select {
		case <-t.C:
			log.Printf("alloc'd 100mb: %d\n", len(s))
			return
		}
	}
	return
}
