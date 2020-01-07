package hlog

type Config struct {
	Debug       bool
	TraceHeader string
	Kafka       *KafkaConfig
	File        *FileConfig
	Format      *FormatterConfig
}

type KafkaConfig struct {
	Servers []string
	Topic   string
	App     string
	AppName string
	EnvName string
}

type FileConfig struct {
	FileName string //加后缀之前的文件命名
	//⤵以下均为rotate配置，没设interval没用
	Interval  int64 //每多少小时切分一次日志，不大于24
	MaxAge    int64 //最多保存多少天的日志文件
	MaxSize   int64 //最多保存多大的日志文件，默认为MB
	LocalTime bool  //是否使用UTC时间来命名日志文件
}

type FormatterConfig struct {
	FullTimestamp   bool
	TimestampFormat string
	DisableSorting  bool
	DisableLog      bool
}
