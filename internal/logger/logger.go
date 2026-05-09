package logger

import (
	"log"
)

var traceEnabled = false

func EnableTrace(enabled bool) {
	traceEnabled = enabled
}

func Info(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func Trace(format string, args ...interface{}) {
	if traceEnabled {
		log.Printf("[TRACE] "+format, args...)
	}
}

func Error(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

func Warning(format string, args ...interface{}) {
	log.Printf("[WARNING] "+format, args...)
}
