// +build !windows

package hlog

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"syscall"
	"time"
)

type Logger struct {
	logrus.Logger
	wg       WaitGroupWrapper
	file     string
	writer   *os.File
	iNode    uint64
	exitChan chan int
	logid    int64
	fields   logrus.Fields
}

func NewLogger(debug bool, w io.Writer, workerId int64) (log *Logger) {
	logger := &Logger{exitChan: make(chan int)}
	logger.Out = w
	logger.fields = logrus.Fields{}
	logger.Formatter = &LogFormatter{
		WorkerId:        workerId,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000-0700",
		DisableSorting:  false,
		Fields:          logger.fields,
	}
	logger.Hooks = make(logrus.LevelHooks)
	logger.Level = logrus.InfoLevel
	logger.logid = workerId
	if debug {
		logger.Level = logrus.DebugLevel
	}
	return logger
}

func NewLoggerWithKafka(debug bool, w io.Writer, workerId int64, c *KafkaConfig) (log *Logger) {
	logger := &Logger{exitChan: make(chan int)}
	logger.Out = w
	logger.fields = logrus.Fields{}
	formatter := &LogFormatter{
		WorkerId:        workerId,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000-0700",
		DisableSorting:  false,
		Fields:          logger.fields,
	}
	logger.Formatter = formatter
	logger.Hooks = make(logrus.LevelHooks)
	if h, err := NewKafkaHookWithFormatter(formatter,c,debug);err ==nil{
		logger.Hooks.Add(h)
	}
	logger.Level = logrus.InfoLevel
	logger.logid = workerId
	if debug {
		logger.Level = logrus.DebugLevel
	}
	return logger
}

func NewLoggerFromFile(verbose bool, log_file string, worker_id int64) (l *Logger) {
	if 0 == len(log_file) {
		return nil
	}

	if f, err := os.OpenFile(log_file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); err == nil {
		l = NewLogger(verbose, f, worker_id)
		l.writer = f
		l.Out = NewLogWriter(f, l.wg, l.exitChan)
		l.file = log_file
		stat, err := os.Stat(l.file)
		if err == nil && stat != nil {
			l.iNode = stat.Sys().(*syscall.Stat_t).Ino
		}
	} else {
		fmt.Printf("NewLoggerFromFile OpenFile err: %v \n", err)
		l = NewLogger(verbose, os.Stdout, worker_id)
		l.writer = os.Stdout
		l.Out = NewLogWriter(os.Stdout, l.wg, l.exitChan)
		l.file = log_file
	}
	l.wg.Wrap(func() { l.fileScaner() })
	return l
}

func (l *Logger) fileScaner() {
	logTimer := time.After(time.Second)
	for {
		select {
		case <-l.exitChan:
			if l.writer != nil {
				l.writer.Close()
			}
			return
		case <-logTimer:
			logTimer = time.After(time.Second)
			stat, err := os.Stat(l.file)
			//文件不存在，或者大小变小都重新打开
			if (err != nil && !os.IsExist(err)) || (nil != stat && stat.Sys().(*syscall.Stat_t).Ino != l.iNode) {
				fmt.Printf("fileScaner diff, err:%v inode:%v \n", err, l.iNode)
				if f, err := os.OpenFile(l.file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); err == nil {
					o := l.Out
					l.Out = NewLogWriter(f, l.wg, l.exitChan)
					if o != nil {
						if w, ok := o.(*LogWriter); ok {
							w.Close()
						}
					}
					if l.writer != nil {
						l.writer.Close()
					}
					l.writer = f
					stat, err := os.Stat(l.file)
					if err == nil && nil != stat {
						l.iNode = stat.Sys().(*syscall.Stat_t).Ino
					}
				} else {
					fmt.Printf("fileScanner OpenFile err: %v", err)
					continue
				}
			}
		}
	}
}

func (l *Logger) GetLogid() int64 {
	return l.logid
}

func (l *Logger) SetLogid(logid int64) {
	l.logid = logid
}

func (l *Logger) GetTraceId() string {
	return l.Formatter.(*LogFormatter).getTraceId()
}

func (l *Logger) SetTraceId(traceId string) {
	l.Formatter.(*LogFormatter).setTraceId(traceId)
}

func (l *Logger) ClearTrace() {
	l.Formatter.(*LogFormatter).clearTrace()
}
func (l *Logger) Close() {
	close(l.exitChan)
	l.wg.Wait()
}

func (l *Logger) ParseTrace(req *http.Request) {
	l.Formatter.(*LogFormatter).parseTrace(req)
}

func (l *Logger) AddHttpTrace(req *http.Request) string {
	return l.Formatter.(*LogFormatter).addHttpTrace(req)
}

func (l *Logger) AddRspTrace(rsp *http.ResponseWriter) string {
	return l.Formatter.(*LogFormatter).addRspTrace(rsp)
}

func (l *Logger) GetTrace() *Trace {
	return l.Formatter.(*LogFormatter).getTrace()
}

func (l *Logger) SetTrace(t *Trace) {
	l.Formatter.(*LogFormatter).setTrace(t)
}

func (l *Logger) AppendFields(fields logrus.Fields) {
	for fieldK, fieldV := range fields {
		l.fields[fieldK] = fieldV
	}
}
