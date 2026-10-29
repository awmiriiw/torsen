package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

var levelNames = map[Level]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

var levelColors = map[Level]string{
	DEBUG: "\033[37m",
	INFO:  "\033[32m",
	WARN:  "\033[33m",
	ERROR: "\033[31m",
	FATAL: "\033[41;37m",
}

const colorReset = "\033[0m"

type Logger struct {
	name         string
	consoleLevel Level
	fileLevel    Level
	fileLogger   *log.Logger
	logFile      *os.File
}

func New(name string, logDir string) *Logger {
	l := &Logger{
		name:         name,
		consoleLevel: INFO,
		fileLevel:    DEBUG,
	}

	if logDir != "" {
		if err := os.MkdirAll(logDir, 0o755); err == nil {
			logPath := filepath.Join(logDir, "torsen.log")
			f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err == nil {
				l.logFile = f
				l.fileLogger = log.New(f, "", 0)
			}
		}
	}

	return l
}

func (l *Logger) SetLevel(levelStr string) {
	switch strings.ToLower(levelStr) {
	case "debug":
		l.consoleLevel = DEBUG
	case "info":
		l.consoleLevel = INFO
	case "warn", "warning":
		l.consoleLevel = WARN
	case "error":
		l.consoleLevel = ERROR
	case "fatal":
		l.consoleLevel = FATAL
	}
}

func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

func (l *Logger) log(level Level, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	if level >= l.consoleLevel {
		color := levelColors[level]
		name := levelNames[level]
		fmt.Fprintf(os.Stderr, "%s%s [%s] [%s] %s%s\n",
			color, timestamp, name, l.name, msg, colorReset)
	}

	if l.fileLogger != nil && level >= l.fileLevel {
		name := levelNames[level]
		l.fileLogger.Printf("%s [%s] [%s] %s", timestamp, name, l.name, msg)
	}
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
}

var _ io.Closer = (*Logger)(nil)
