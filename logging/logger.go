package logging

import (
	"io"
	"sync"
	"goblog"
	"os"
	"time"
	"encoding/json"
	"fmt"
	"strings"
	"strconv"
	"bytes"
)

var crlfEscaper = strings.NewReplacer("\r", "\\r", "\n", "\\n")

type Log map[string]interface{}

func (l Log) Format() (string, error) {
	res, err := json.Marshal(l)
	if err == nil {
		return string(res), nil
	}
	return "", err
}

func (l Log) GoString() string {
	count := len(l)
	buf := bytes.NewBufferString("Log{")
	for key, value := range l {
		if count--; count == 0 {
			fmt.Fprintf(buf, "%s:%#v}", key, value)
		} else {
			fmt.Fprintf(buf, "%s:%#v, ", key, value)
		}
	}
	return buf.String()
}

func (l Log) String() string {
	return l.GoString()
}

type Level uint8

const (
	// EmergLevel is 0, "Emergency",system is unusable
	EmergLevel Level = iota
	// AlertLevel is 1, "Alert", action must be taken immediately
	AlertLevel
	// CritiLevel is 2, "Critical", critical conditions
	CritiLevel
	// ErrLevel is 3, "Error", error conditions
	ErrLevel
	// WarningLevel is 4, "Warning", warning conditions
	WarningLevel
	// NoticeLevel is 5, "Notice", normal but significant conditions
	NoticeLevel
	// InfoLevel is 6, "Informational", informational message
	InfoLevel
	// DebugLevel is 7, "Debug", debug-level message
	DebugLevel
)

var levels = []string{"EMERG", "ALERT", "CRIT", "ERR", "WARNING", "NOTICE", "INFO", "DEBUG"}
var std = New(os.Stderr)

func Default(devMode ...bool) *Logger {
	if len(devMode) > 0 && devMode[0] {
		std.SetLogConsume(developmentConsume)
	}
	return std
}

func developmentConsume(log Log, ctx *goblog.Context) {
	std.mu.Lock()
	defer std.mu.Unlock()

	end := time.Now().UTC()
	FprintWithColor(std.Out, fmt.Sprintf("%s", log["IP"]), ColorGreen)
	fmt.Fprintf(std.Out, ` - - [%s] "%s %s %s"`, end.Format(std.tf), log["Method"], log["URL"], log["Proto"])
	status := log["Status"].(int)
	FprintWithColor(std.Out, strconv.Itoa(status), colorStatus(status))
	resTime := float64(end.Sub(log["Start"].(time.Time))) / 1e6
	fmt.Fprintln(std.Out, fmt.Sprintf("%s %.3fms", log["Length"], resTime))
}

func New(w io.Writer) *Logger {
	logger := &Logger{Out: w}
	logger.SetLevel(DebugLevel)
	logger.SetTimeFormat("2006-01-02T15:04:05.999Z")
	logger.SetLogFormat("%s %s %s")

	logger.init = func(log Log, ctx *goblog.Context) {
		log["IP"] = ctx.IP()
		log["Method"] = ctx.Method
		log["URL"] = ctx.Req.URL.String()
		log["Proto"] = ctx.Req.Proto
		log["UserAgent"] = ctx.Get(goblog.HeaderUserAgent)
		log["Start"] = time.Now()
	}

	logger.consume = func(log Log, _ *goblog.Context) {
		end := time.Now()
		if t, ok := log["Start"].(time.Time); ok {
			log["Time"] = end.Sub(t) / 1e6
			delete(log, "Start")
		}

		if str, err := log.Format(); err == nil {
			logger.Output(end, InfoLevel, str)
		} else {
			logger.Output(end, WarningLevel, log.String())
		}
	}
	return logger
}

type Logger struct {
	Out 	io.Writer
	l 		Level
	tf, lf 	string
	mu 		sync.Mutex
	init 	func(Log, *goblog.Context)
	consume func(Log, *goblog.Context)
}

func (l *Logger) Output(t time.Time, level Level, s string) (err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// log level checked before
	if level < 4 {
		s = goblog.ErrorWithStack(s, 4).String()
	}
	if l := len(s); l > 0 && s[l-1] == '\n' {
		s = s[0 : l-1]
	}
	_, err = fmt.Fprintf(l.Out, l.lf, t.UTC().Format(l.tf), levels[level], crlfEscaper.Replace(s))
	if err == nil {
		l.Out.Write([]byte{'\n'})
	}
	return
}

func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if level > DebugLevel {
		panic(goblog.Err.WithMsg("invalid logger level"))
	}
	l.l = level
}

func (l *Logger) SetTimeFormat(timeFormat string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tf = timeFormat
}

func (l *Logger) SetLogFormat(logFormat string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lf = logFormat
}

func (l *Logger) SetLogConsume(fn func(Log, *goblog.Context)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.consume = fn
}

func (l *Logger) New(ctx *goblog.Context) (interface{}, error) {
	log := Log{}
	l.init(log, ctx)
	return log, nil
}

func (l *Logger) FromCtx(ctx *goblog.Context) Log {
	any, _ := ctx.Any(l)
	return any.(Log)
}

func (l *Logger) Serve(ctx *goblog.Context) error {
	log := l.FromCtx(ctx)
	// Add a "end book" to flush logs
	ctx.OnEnd(func() {
		//Ignore empty log
		if len(log) == 0 {
			return
		}
		log["Status"] = ctx.Res.Status()
		log["Length"] = len(ctx.Res.Body())
		l.consume(log, ctx)
	})
	return nil
}

func colorStatus(code int) ColorType {
	switch {
	case code >= 200 && code < 300:
		return ColorGreen
	case code >= 300 && code < 400:
		return ColorCyan
	case code >= 400 && code < 500:
		return ColorYellow
	default:
		return ColorRed
	}
}