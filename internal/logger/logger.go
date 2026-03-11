package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Level 日志级别
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// Logger 日志记录器
type Logger struct {
	level      Level
	logDir     string
	consoleLog bool // 是否输出到控制台
	fileLog    bool // 是否输出到文件
	mutex      sync.Mutex
}

// New 创建日志记录器
// level: debug, info, warn, error
// logDir: 日志目录，空字符串表示不写文件
func New(level, logDir string) *Logger {
	l := &Logger{
		level:      parseLevel(level),
		logDir:     logDir,
		consoleLog: true,
		fileLog:    logDir != "",
	}

	if l.fileLog {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Printf("⚠️ 创建日志目录失败: %v", err)
			l.fileLog = false
		}
	}

	return l
}

func parseLevel(level string) Level {
	switch level {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// Debug 调试日志（debug级别才输出到文件）
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.level <= DebugLevel {
		msg := fmt.Sprintf(format, v...)
		l.write("DEBUG", msg, l.fileLog) // debug写文件，不写控制台
	}
}

// Info 信息日志
func (l *Logger) Info(format string, v ...interface{}) {
	if l.level <= InfoLevel {
		msg := fmt.Sprintf(format, v...)
		l.write("INFO", msg, true)
	}
}

// Warn 警告日志
func (l *Logger) Warn(format string, v ...interface{}) {
	if l.level <= WarnLevel {
		msg := fmt.Sprintf(format, v...)
		l.write("WARN", msg, true)
	}
}

// Error 错误日志
func (l *Logger) Error(format string, v ...interface{}) {
	if l.level <= ErrorLevel {
		msg := fmt.Sprintf(format, v...)
		l.write("ERROR", msg, true)
	}
}

// write 写入日志
// console: 是否输出到控制台
func (l *Logger) write(level, msg string, console bool) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] %s: %s", timestamp, level, msg)

	// 输出到控制台
	if console && l.consoleLog {
		log.Println(logLine)
	}

	// 写入文件
	if l.fileLog && l.level == DebugLevel {
		l.mutex.Lock()
		defer l.mutex.Unlock()

		filename := filepath.Join(l.logDir, time.Now().Format("20060102")+".log")
		f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		defer f.Close()

		fmt.Fprintln(f, logLine)
	}
}
