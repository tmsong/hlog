package hlog

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"strings"
)

var KafkaFormatter KafkaFormatterFunc //replaceable

func init() {
	KafkaFormatter = NewDefaultKafkaLogFormatter
}

type KafkaFormatterFunc = func(f logrus.Formatter, c *KafkaConfig) logrus.Formatter

type DefaultKafkaLogFormatter struct {
	Formatter logrus.Formatter
	*KafkaConfig
}

func NewDefaultKafkaLogFormatter(f logrus.Formatter, c *KafkaConfig) logrus.Formatter {
	return &DefaultKafkaLogFormatter{
		Formatter:   f,
		KafkaConfig: c,
	}
}

func (f *DefaultKafkaLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	if f.Formatter == nil {
		return nil, nil
	}
	message, err := f.Formatter.Format(entry)
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{
		"level":    strings.ToUpper(entry.Level.String()),
		"app":      f.App,
		"app_name": f.AppName,
		"env_name": f.EnvName,
		"message":  string(message),
	}
	if defaultf, ok := f.Formatter.(*DefaultLogFormatter); ok {
		m["trace_id"] = defaultf.TraceId
	}
	return json.Marshal(m)
}

func (f *DefaultKafkaLogFormatter) Clone(formatter logrus.Formatter) *DefaultKafkaLogFormatter {
	return &DefaultKafkaLogFormatter{
		Formatter:   f,
		KafkaConfig: f.KafkaConfig,
	}
}
