package hlog

import (
	"log"
	"sync"
)

type WaitGroupWrapper struct {
	sync.WaitGroup
}

func (w *WaitGroupWrapper) Wrap(cb func()) {
	w.Add(1)
	defer w.Done()
	defer func() {
		if err := recover(); err != nil {
			log.Printf("go routine run error: %+v", err)
		}
	}()
	go func() {
		cb()
	}()
}
