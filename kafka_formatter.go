/**
 * @note
 * kafka_formatter
 *
 * @author	songtianming
 * @date 	2019-12-17
 */
package slog

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"strings"
)

type KafkaLogFormatter struct {
	Formatter *LogFormatter
	*KafkaConfig
}

func NewKafkaLogFormatter(f *LogFormatter, c *KafkaConfig) *KafkaLogFormatter {
	return &KafkaLogFormatter{
		Formatter:   f,
		KafkaConfig: c,
	}
}

const NOT_SET = "NOT_SET"

func (f *KafkaLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	if f.Formatter == nil {
		return nil, nil
	}
	message, err := f.Formatter.Format(entry)
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{
		"level":           strings.ToUpper(entry.Level.String()),
		"app":             f.App,
		"app_name":        f.AppName,
		"env_name":        f.EnvName,
		"profile":         f.EnvName,
		"captain_seq":     NOT_SET,
		"captain_gen":     NOT_SET,
		"build_name":      NOT_SET,
		"build_timestamp": NOT_SET,
		"build_git_hash":  NOT_SET,
		"message":         string(message),
		"trace_id":        f.Formatter.TraceId,
	}
	return json.Marshal(m)
}
