package slog

const (
	DEFAULT_TIMESTAMP_FORMAT = "2006-01-02 15:04:05.000-0700"
	DEFAULT_TRACE_HEADER     = "default-header-rid"
)

const (
	LOG_OK        string = "ok"
	LOG_TAG       string = "tag"
	LOG_BEGIN     string = "___TIME___"
	LOG_TAG_UNDEF string = "_undef"
)

const (
	LOGTAG_REQUEST_OK string = "_com_http_success"
	LOGTAG_THRIFT_OK  string = "_com_thrift_success"
	LOGTAG_REDIS_OK   string = "_com_redis_success"
	LOGTAG_ACCESS_IN  string = "_com_request_in"
	LOGTAG_ACCESS_OUT string = "_com_request_out"
	LOGTAG_MYSQL_OK   string = "_com_mysql_success"
)

const (
	LOGTAG_REQUEST_ERR string = "_com_http_failure"
	LOGTAG_THRIFT_ERR  string = "_com_thrift_failure"
	LOGTAG_REDIS_ERR   string = "_com_redis_failure"
	LOGTAG_MYSQL_ERR   string = "_com_mysql_failure"
)

var TagDescSuccMapErr = map[string]string{
	LOGTAG_REQUEST_OK: LOGTAG_REQUEST_ERR,
	LOGTAG_THRIFT_OK:  LOGTAG_THRIFT_ERR,
	LOGTAG_REDIS_OK:   LOGTAG_REDIS_ERR,
	LOGTAG_MYSQL_OK:   LOGTAG_MYSQL_ERR,
}

const (
	LOGCODE_NAME string = "code"
)
