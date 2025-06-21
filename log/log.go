package log

import (
	"log"
	"os"
)

// Logger is a simple interface for logging.
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// defaultLogger implements the Logger interface using the standard log package.
type defaultLogger struct {
	info  *log.Logger
	err   *log.Logger
	debug *log.Logger
}

// NewDefaultLogger creates a new default logger.
func NewDefaultLogger() Logger {
	return &defaultLogger{
		info:  log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile),
		err:   log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile),
		debug: log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (l *defaultLogger) Info(msg string, args ...interface{}) {
	l.info.Printf(msg, args...)
}

func (l *defaultLogger) Error(msg string, args ...interface{}) {
	l.err.Printf(msg, args...)
}

func (l *defaultLogger) Debug(msg string, args ...interface{}) {
	l.debug.Printf(msg, args...)
}
