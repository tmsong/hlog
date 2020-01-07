// +build !windows

package hlog

import (
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
)

type Logger struct {
	logrus.Logger
	formatterFunc NewFormatterFunc
	config        *Config
	wg            WaitGroupWrapper
	exitChan      chan struct{}
	logid         int64
	fields        logrus.Fields
}

type NewFormatterFunc = func(c *Config, f logrus.Fields, workerId int64) logrus.Formatter

func newLogger(c *Config, w io.Writer, formatterFunc NewFormatterFunc, workerId int64) (log *Logger) {
	logger := &Logger{exitChan: make(chan struct{}), config: c, formatterFunc: formatterFunc, fields: logrus.Fields{}, logid: workerId}
	logger.Out = w
	logger.Formatter = formatterFunc(c, logger.fields, workerId)
	logger.Hooks = make(logrus.LevelHooks)
	logger.Level = logrus.InfoLevel
	if c.Debug {
		logger.Level = logrus.DebugLevel
	}
	return logger
}

//Clone a logger with a exist logger's config and out
func (l *Logger) Clone(workerId int64) (log *Logger) {
	log = newLogger(l.config, l.Out, l.formatterFunc, workerId)
	for level, hooks := range l.Hooks {
		log.Hooks[level] = append(log.Hooks[level], hooks...)
	}
	return log
}

func NewLoggerWithConfig(c *Config, formatterFunc NewFormatterFunc, workerId int64) (l *Logger) {
	if c.File == nil {
		c.File = &FileConfig{}
	}
	l = newLogger(c, nil, formatterFunc, workerId)
	l.Out = newFileWritter(c.File, l.wg, l.exitChan)
	if c.Kafka != nil {
		if h, err := NewKafkaHookWithFormatter(l.Formatter.(*LogFormatter), c.Kafka, c.Debug); err == nil {
			l.Hooks.Add(h)
		}
	}
	return
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
