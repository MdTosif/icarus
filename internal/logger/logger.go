package logger

import (
	"io"
	"log"
	"os"
	"sync"
)

// Level type as before
type Level int

const (
    DEBUG Level = iota
    INFO
    WARN
    ERROR
)

func (l Level) String() string {
    switch l {
    case DEBUG:
        return "DEBUG"
    case INFO:
        return "INFO"
    case WARN:
        return "WARN"
    case ERROR:
        return "ERROR"
    default:
        return "UNKNOWN"
    }
}




// Package-level (or you can make it struct-level) minimum level.
var (
    mu       sync.RWMutex
    minLevel = INFO
    // Individual loggers
    debugLogger *log.Logger
    infoLogger  *log.Logger
    warnLogger  *log.Logger
    errorLogger *log.Logger
)

func init() {
    // By default, write DEBUG, INFO, WARN to stdout; ERROR to stderr.
    // You can customize flags here (e.g. log.LstdFlags or log.Lshortfile|log.LstdFlags).
    debugLogger = log.New(os.Stdout, "DEBUG: ", log.LstdFlags)
    infoLogger = log.New(os.Stdout, "INFO: ", log.LstdFlags)
    warnLogger = log.New(os.Stdout, "WARN: ", log.LstdFlags)
    errorLogger = log.New(os.Stderr, "ERROR: ", log.LstdFlags)
}

// SetOutput allows directing all levels to a given writer.
// If you want different outputs per level, you could set them individually.
func SetOutput(w io.Writer) {
    mu.Lock()
    defer mu.Unlock()
    debugLogger.SetOutput(w)
    infoLogger.SetOutput(w)
    warnLogger.SetOutput(w)
    errorLogger.SetOutput(w)
}

// SetMinLevel sets the minimum log level globally.
// Messages below this level will be skipped.
func SetMinLevel(l Level) {
    mu.Lock()
    defer mu.Unlock()
    minLevel = l
}

// internal check
func shouldLog(l Level) bool {
    mu.RLock()
    defer mu.RUnlock()
    return l >= minLevel
}

// Public functions:
func Debug(v ...interface{}) {
    if shouldLog(DEBUG) {
        debugLogger.Println(v...)
    }
}
func Debugf(format string, v ...interface{}) {
    if shouldLog(DEBUG) {
        debugLogger.Printf(format, v...)
    }
}

func Info(v ...interface{}) {
    if shouldLog(INFO) {
        infoLogger.Println(v...)
    }
}
func Infof(format string, v ...interface{}) {
    if shouldLog(INFO) {
        infoLogger.Printf(format, v...)
    }
}

func Warn(v ...interface{}) {
    if shouldLog(WARN) {
        warnLogger.Println(v...)
    }
}
func Warnf(format string, v ...interface{}) {
    if shouldLog(WARN) {
        warnLogger.Printf(format, v...)
    }
}

func Error(v ...interface{}) {
    if shouldLog(ERROR) {
        errorLogger.Println(v...)
    }
}
func Errorf(format string, v ...interface{}) {
    if shouldLog(ERROR) {
        errorLogger.Printf(format, v...)
    }
}
