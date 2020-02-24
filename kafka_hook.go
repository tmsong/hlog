package hlog

import (
	"crypto/tls"
	"errors"
	"github.com/Shopify/sarama"
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"time"
)

func NewKafkaHookWithFormatter(f logrus.Formatter, c *KafkaConfig, debug bool) (*KafkaLogrusHook, error) {
	kFormatter := KafkaFormatter(f, c)
	var levels []logrus.Level
	if debug {
		levels = logrus.AllLevels
	} else {
		levels = []logrus.Level{
			logrus.ErrorLevel,
			logrus.WarnLevel,
			logrus.InfoLevel,
		}
	}
	return NewKafkaLogrusHook(levels, kFormatter, c, nil)
}

// KafkaLogrusHook is the primary struct
type KafkaLogrusHook struct {
	config    *KafkaConfig
	hostname  string
	levels    []logrus.Level
	formatter logrus.Formatter
	producer  sarama.AsyncProducer
}

// NewKafkaLogrusHook creates a new KafkaHook
func NewKafkaLogrusHook(
	levels []logrus.Level,
	formatter logrus.Formatter,
	c *KafkaConfig,
	tls *tls.Config) (*KafkaLogrusHook, error) {
	var err error
	var producer sarama.AsyncProducer
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForLocal       // Only wait for the leader to ack
	kafkaConfig.Producer.Compression = sarama.CompressionSnappy   // Compress messages
	kafkaConfig.Producer.Flush.Frequency = 500 * time.Millisecond // Flush batches every 500ms

	// check here if provided *tls.Config is not nil and assign to the sarama config
	// NOTE: we automatically enabled the TLS config because sarama would error out if our
	//       config were non-nil but disabled. To avoid issue further down the stack, we enable.
	if tls != nil {
		kafkaConfig.Net.TLS.Enable = true
		kafkaConfig.Net.TLS.Config = tls
	}

	if producer, err = sarama.NewAsyncProducer(c.Servers, kafkaConfig); err != nil {
		return nil, err
	}

	go func() {
		for err := range producer.Errors() {
			log.Printf("Failed to send log entry to Kafka: %v\n", err)
		}
	}()

	var hostname string
	if hostname, err = os.Hostname(); err != nil {
		hostname = "localhost"
	}

	hook := &KafkaLogrusHook{
		c,
		hostname,
		levels,
		formatter,
		producer,
	}

	return hook, nil
}

func (hook *KafkaLogrusHook) Clone(f logrus.Formatter) *KafkaLogrusHook {
	return &KafkaLogrusHook{
		hook.config,
		hook.hostname,
		hook.levels,
		KafkaFormatter(f, hook.config),
		hook.producer,
	}
}

// Levels is required to implement the hook interface from logrus
func (hook *KafkaLogrusHook) Levels() []logrus.Level {
	return hook.levels
}

// Fire is required to implement the hook interface from logrus
func (hook *KafkaLogrusHook) Fire(entry *logrus.Entry) error {
	var partitionKey sarama.ByteEncoder
	var b []byte
	var err error

	t := entry.Time
	if b, err = t.MarshalBinary(); err != nil {
		return err
	}
	partitionKey = sarama.ByteEncoder(b)

	if hook.config.InjectHostname {
		if _, ok := entry.Data["hostname"]; !ok {
			entry.Data["hostname"] = hook.hostname
		}
	}

	if b, err = hook.formatter.Format(entry); err != nil {
		return err
	}
	value := sarama.ByteEncoder(b)

	topic := hook.config.Topic
	if tsRaw, ok := entry.Data["topic"]; ok {
		if ts, ok := tsRaw.(string); !ok {
			return errors.New("Incorrect topic filed type (should be string)")
		} else {
			topic = ts
		}
	}
	hook.producer.Input() <- &sarama.ProducerMessage{
		Key:   partitionKey,
		Topic: topic,
		Value: value,
	}
	return nil
}
