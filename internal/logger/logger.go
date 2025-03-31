package logger

import (
	"fmt"
	"log"
	"os"
	"sync"
)

var mux sync.RWMutex

var logger Logger = &defaultLogger{
	debug:  true,
	stdlog: log.New(os.Stdout, "debug: ", log.LstdFlags),
	errlog: log.New(os.Stderr, "error: ", log.LstdFlags),
}

func Get() Logger {
	mux.RLock()
	defer mux.RUnlock()
	return logger
}

func Set(l Logger) {
	mux.Lock()
	defer mux.Unlock()
	logger = l
}

type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
}

type defaultLogger struct {
	debug  bool
	stdlog *log.Logger
	errlog *log.Logger
}

func (l *defaultLogger) Debug(msg string, keysAndValues ...any) {
	if l.debug {
		_ = l.stdlog.Output(2, l.format(msg, keysAndValues...))
	}
}

func (l *defaultLogger) Error(msg string, keysAndValues ...any) {
	_ = l.errlog.Output(2, l.format(msg, keysAndValues...))
}

func (l *defaultLogger) format(msg string, keysAndValues ...any) string {
	if len(keysAndValues)%2 != 0 {
		panic("logger: keysAndValues must be even")
	}
	if len(keysAndValues) == 0 {
		return msg
	}
	formatted := msg
	for i := 0; i < len(keysAndValues); i += 2 {
		key, val := keysAndValues[i], keysAndValues[i+1]
		formatted += " " + l.stringify(key) + "=" + l.stringify(val)
	}
	return formatted
}

func (l *defaultLogger) stringify(v any) string {
	if v == nil {
		return "nil"
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
