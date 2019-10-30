package hlog

import (
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"encoding/hex"
	"net/http"

	"github.com/sirupsen/logrus"
)

var (
	baseTimestamp time.Time
)

func init() {
	baseTimestamp = time.Now()
}

func miniTS() int {
	return int(time.Since(baseTimestamp) / time.Second)
}

type Trace struct {
	TraceId   string `json:"traceId,omitempty"`
	Caller    string `json:"caller,omitempty"`
	SrcMethod string `json:"srcMethod,omitempty"`
}

type HintContent struct {
	Sample HintSampling `json:"sample,omitempty"`
}

type HintSampling struct {
	Rate int   `json:"rate,omitempty"`
	Code int64 `json:"code,omitempty"`
}

type LogFormatter struct {
	WorkerId         int64
	DisableTimestamp bool
	FullTimestamp    bool
	TimestampFormat  string
	DisableSorting   bool
	DisableLog       bool
	Trace
	Fields logrus.Fields
}

func shouldContinueTraceBack(name string)bool{
	if strings.Contains(name, "hlog") ||
		!strings.Contains(name, "logrus") ||
		!strings.Contains(name,"gorm"){
		return true
	}
	return false
}

func (f *LogFormatter) header() string {
	p, file, line, ok := runtime.Caller(9)
	name := runtime.FuncForPC(p).Name()
	start := 10
	if strings.Contains(name,"gorm"){
		start = 13
	}
	if shouldContinueTraceBack(name){
		for i := start; i < start+3; i++ {
			if p, file, line, ok = runtime.Caller(i);ok{
				name = runtime.FuncForPC(p).Name()
				if !shouldContinueTraceBack(name){
					break
				}
			}
		}
	}
	if !ok {
		file = "???"
		line = 1

	} else {
		dirs := strings.Split(file, "/")
		if len(dirs) >= 2 {
			file = dirs[len(dirs)-2] + "/" + dirs[len(dirs)-1]
		} else {
			file = dirs[len(dirs)-1]

		}

	}
	return fmt.Sprintf("%s:%d", file, line)

}
func (f *LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	for fieldK, fieldV := range f.Fields {
		entry.Data[fieldK] = fieldV
	}
	var keys = make([]string, 0, len(entry.Data))
	var tag = LOG_TAG_UNDEF
	for k := range entry.Data {
		if k == LOG_TAG {
			tag = entry.Data[k].(string)
			continue
		}
		keys = append(keys, k)
	}

	if !f.DisableSorting {
		sort.Strings(keys)
	}

	b := &bytes.Buffer{}

	prefixFieldClashes(entry.Data)

	if f.TimestampFormat == "" {
		f.TimestampFormat = time.RFC3339
	}
	f.printLog(b, entry, keys, tag)

	return b.Bytes(), nil
}

func (f *LogFormatter) printLog(b *bytes.Buffer, entry *logrus.Entry, keys []string, tag string) {
	if f.DisableLog && tag != LOGTAG_ACCESS_IN &&
		tag != LOGTAG_ACCESS_OUT && entry.Logger.GetLevel() >= logrus.ErrorLevel {
		return
	}
	defer func() {
		b.WriteByte('\n')
	}()
	levelText := strings.ToUpper(entry.Level.String())
	header := f.header()
	if !f.FullTimestamp {
		fmt.Fprintf(b, "[%s][%d][%s] %s||_msg=%s||logid=%d||traceid=%s",
			levelText,
			miniTS(),
			header,
			tag,
			strings.Trim(entry.Message, " \r\t\v\n"),
			f.WorkerId,
			f.getTraceId())
	} else {
		fmt.Fprintf(b, "[%s][%s][%s] %s||_msg=%s||logid=%d||traceid=%s",
			levelText,
			entry.Time.Format(f.TimestampFormat),
			header,
			tag,
			strings.Trim(entry.Message, " \r\t\v\n"),
			f.WorkerId,
			f.getTraceId())
	}
	for _, k := range keys {
		v := entry.Data[k]
		if k == LOG_BEGIN {
			v = float64(entry.Time.Sub(v.(time.Time)).Nanoseconds()) / (1000 * 1000)
			k = "proc_time"
		}
		switch v.(type) {
		case []byte:
			v = string(v.([]byte))
		}
		t := fmt.Sprintf("%v", v)
		fmt.Fprintf(b, "||%s=%v", k, strings.Trim(t, " \r\t\v\n"))
	}
}

func (f *LogFormatter) getTraceId() string {
	if len(f.TraceId) <= 0 {
		f.TraceId = calculateTraceId(getIp())
	}
	return f.TraceId
}

func (f *LogFormatter) setTraceId(traceId string) {
	f.TraceId = traceId
}

func (f *LogFormatter) clearTrace() {
	f.Trace = Trace{}
}

func (f *LogFormatter) parseTrace(req *http.Request) {
	f.TraceId = req.Header.Get(http.CanonicalHeaderKey(DEFAULT_TRACE_HEADER))
}

func (f *LogFormatter) getTrace() *Trace {
	return &Trace{
		TraceId: f.getTraceId(),
	}
}
func (f *LogFormatter) setTrace(t *Trace) {
	f.TraceId = t.TraceId
	f.SrcMethod = t.SrcMethod
	f.Caller = t.Caller
}

func (f *LogFormatter) addHttpTrace(req *http.Request) string {
	trace := f.getTrace()
	req.Header.Set(DEFAULT_TRACE_HEADER, trace.TraceId)
	return trace.TraceId
}

func (f *LogFormatter) addRspTrace(rsp *http.ResponseWriter) string {
	trace := f.getTrace()
	(*rsp).Header().Set(DEFAULT_TRACE_HEADER, trace.TraceId)
	return trace.TraceId
}
func prefixFieldClashes(data logrus.Fields) {
	_, ok := data["time"]
	if ok {
		data["fields.time"] = data["time"]
	}

	_, ok = data["msg"]
	if ok {
		data["fields.msg"] = data["msg"]
	}

	_, ok = data["level"]
	if ok {
		data["fields.level"] = data["level"]
	}
}
func NewSpanId() string {
	return fmt.Sprintf("%d", rand.Int63())
}

func calculateTraceId(ip string) (traceId string) {
	now := time.Now()
	timestamp := uint32(now.Unix())
	timeNano := now.UnixNano()
	pid := os.Getpid()
	b := bytes.Buffer{}

	b.WriteString(hex.EncodeToString(net.ParseIP(ip).To4()))
	b.WriteString(fmt.Sprintf("%x", timestamp&0xffffffff))
	b.WriteString(fmt.Sprintf("%04x", timeNano&0xffff))
	b.WriteString(fmt.Sprintf("%04x", pid&0xffff))
	b.WriteString(fmt.Sprintf("%06x", rand.Int31n(1<<24)))
	b.WriteString("b0")
	return b.String()
}

func getIp() string {
	var ip string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		ip = "127.0.0.1"
		return ip
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
				break
			}
		}
	}
	return ip
}
