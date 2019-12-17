/**
 * @note
 * kafka_formatter
 *
 * @author	songtianming
 * @date 	2019-12-17
 */
package hlog

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
)

type KafkaConfig struct{
	Servers []string
	Topic string
	App           string
	AppName       string
	EnvName       string
}

type KafkaLogFormatter struct {
	Formatter *LogFormatter
	*KafkaConfig
}

func NewKafkaLogFormatter(f *LogFormatter,c *KafkaConfig) *KafkaLogFormatter {
	return &KafkaLogFormatter{
		Formatter: f,
		KafkaConfig:c,
	}
}

func (f *KafkaLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	if f.Formatter == nil {
		return nil, nil
	}
	message, err := f.Formatter.Format(entry)
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{
		"app":      f.App,
		"app_name": f.AppName,
		"env_name": f.EnvName,
		"message":  string(message),
		"trace_id": f.Formatter.TraceId,
	}
	return json.Marshal(m)
}