package hlog

import (
	"io"
	"os"
	"sync"
	"time"
)

const (
	DefaultQueueSize        = 100000
	DefaultFlushDiskTimeout = 10 * time.Millisecond
)

var (
	singleWritter io.Writer   = os.Stdout
	singleQueue   chan []byte = make(chan []byte, DefaultQueueSize)
)

type LogWriter struct {
	o     io.Writer
	wg    WaitGroupWrapper
	quit  chan int
	close chan int
	lock  sync.Mutex
}

func NewLogWriter(w io.Writer, wg WaitGroupWrapper, quit chan int) *LogWriter {
	writer := &LogWriter{
		o:     w,
		wg:    wg,
		quit:  quit,
		close: make(chan int),
	}
	singleWritter = w
	wg.Wrap(writer.watcher)
	return writer
}

func (this *LogWriter) watcher() {
	for {
		select {
		case msg := <-singleQueue:
			this.flush(msg)
		case <-this.quit:
			for {
				select {
				case msg := <-singleQueue:
					this.flush(msg)
				default:
					return
				}
			}
		case <-this.close:
			for {
				select {
				case msg := <-singleQueue:
					this.flush(msg)
				default:
					return
				}
			}
		}
	}
}

func (this *LogWriter) flush(msg []byte) {
	done := make(chan bool, 1)
	this.wg.Wrap(func() {
		this.lock.Lock()
		defer this.lock.Unlock()
		singleWritter.Write(msg)
		done <- true
	})
	select {
	case <-done:
		return
	case <-time.After(DefaultFlushDiskTimeout):
		return
	}
}

func (this *LogWriter) Write(p []byte) (n int, err error) {
	select {
	case singleQueue <- p:
		return len(p), nil
	default:
		return 0, nil
	}
}

func (this *LogWriter) Close() error {
	close(this.close)
	return nil
}
