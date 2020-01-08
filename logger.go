package hlog

import (
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
)

type Logger struct {
	logrus.Logger
	config   *Config
	wg       WaitGroupWrapper
	exitChan chan struct{}
	logid    int64
	fields   logrus.Fields
}

func newLogger(c *Config, w io.Writer, workerId int64) (log *Logger) {
	logger := &Logger{exitChan: make(chan struct{}), config: c, fields: logrus.Fields{}, logid: workerId}
	logger.Out = w
	logger.Formatter = LogFormatter(c, logger.fields, workerId)
	logger.Hooks = make(logrus.LevelHooks)
	logger.Level = logrus.InfoLevel
	if c.Debug {
		logger.Level = logrus.DebugLevel
	}
	return logger
}

//Clone a logger with a exist logger's config and out
func (l *Logger) Clone(workerId int64) (log *Logger) {
	log = newLogger(l.config, l.Out, workerId)
	for level, hooks := range l.Hooks {
		log.Hooks[level] = append(log.Hooks[level], hooks...)
	}
	return log
}

func NewLoggerWithConfig(c *Config, workerId int64) (l *Logger) {
	if c.File == nil {
		c.File = &FileConfig{}
	}
	l = newLogger(c, nil, workerId)
	l.Out = newFileWriter(c.File, l.wg, l.exitChan)
	if c.Kafka != nil {
		if h, err := NewKafkaHookWithFormatter(l.Formatter, c.Kafka, c.Debug); err == nil {
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
	return l.Formatter.(*DefaultLogFormatter).getTraceId()
}

func (l *Logger) SetTraceId(traceId string) {
	l.Formatter.(*DefaultLogFormatter).setTraceId(traceId)
}

func (l *Logger) ClearTrace() {
	l.Formatter.(*DefaultLogFormatter).clearTrace()
}
func (l *Logger) Close() {
	close(l.exitChan)
	l.wg.Wait()
}

func (l *Logger) ParseTrace(req *http.Request) {
	l.Formatter.(*DefaultLogFormatter).parseTrace(req)
}

func (l *Logger) AddHttpTrace(req *http.Request) string {
	return l.Formatter.(*DefaultLogFormatter).addHttpTrace(req)
}

func (l *Logger) AddRspTrace(rsp *http.ResponseWriter) string {
	return l.Formatter.(*DefaultLogFormatter).addRspTrace(rsp)
}

func (l *Logger) GetTrace() *Trace {
	return l.Formatter.(*DefaultLogFormatter).getTrace()
}

func (l *Logger) SetTrace(t *Trace) {
	l.Formatter.(*DefaultLogFormatter).setTrace(t)
}

func (l *Logger) AppendFields(fields logrus.Fields) {
	for fieldK, fieldV := range fields {
		l.fields[fieldK] = fieldV
	}
}
