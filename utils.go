package slog

import (
	"github.com/sirupsen/logrus"
	"time"
)

func GetLogField(tag string, fields ...logrus.Fields) logrus.Fields {
	f := logrus.Fields{
		"tag":     tag,
		LOG_BEGIN: time.Now(),
	}
	for _, field := range fields {
		for k, v := range (map[string]interface{})(field) {
			f[k] = v
		}
	}
	return f
}
