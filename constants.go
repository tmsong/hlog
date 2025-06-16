package hlog

const (
	DefaultTimestampFormat      = "2006-01-02 15:04:05.000-0700"
	DefaultKafkaTimestampFormat = "2006-01-02T15:04:05.999Z07:00"
	DefaultTraceHeader          = "default-header-rid"
)

const (
	LogOK       string = "ok"
	LogTag      string = "tag"
	LogBegin    string = "___TIME___"
	LogTagUndef string = "_undef"
)

const (
	LogTagRequestOk string = "_com_http_success"
	LogTagThriftOk  string = "_com_thrift_success"
	LogTagRedisOk   string = "_com_redis_success"
	LogTagAccessIn  string = "_com_request_in"
	LogTagAccessOut string = "_com_request_out"
	LogTagMysqlOk   string = "_com_mysql_success"
)

const (
	LogTagRequestErr string = "_com_http_failure"
	LogTagThriftErr  string = "_com_thrift_failure"
	LogTagRedisErr   string = "_com_redis_failure"
	LogTagMysqlErr   string = "_com_mysql_failure"
)

var TagDescSuccMapErr = map[string]string{
	LogTagRequestOk: LogTagRequestErr,
	LogTagThriftOk:  LogTagThriftErr,
	LogTagRedisOk:   LogTagRedisErr,
	LogTagMysqlOk:   LogTagMysqlErr,
}

const (
	LogCodeName string = "code"
)
