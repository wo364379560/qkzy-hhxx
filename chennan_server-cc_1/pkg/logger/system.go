package logger

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Keep package-level logging safe in unit tests and during early startup.
// Init/SystemLog replaces this no-op logger with the configured sinks.
var system = zap.NewNop()

func SystemLog() {
	core := zapcore.NewTee(
		*Console(),
		*LocalFile("system.log"),
	)
	system = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.Development())
}

func Debug(message string, fields ...zap.Field) {
	system.Debug(message, fields...)
}

func Debugf(format string, v ...any) {
	msg := validPrintf(format, v...)
	system.Debug(msg)
}

func Info(message string, fields ...zap.Field) {
	system.Info(message, fields...)
}

func Infof(format string, v ...any) {
	msg := validPrintf(format, v...)
	system.Info(msg)
}

func Warn(message string, fields ...zap.Field) {
	system.Warn(message, fields...)
}

func Warnf(format string, v ...any) {
	msg := validPrintf(format, v...)
	system.Warn(msg)
}

func Error(message string, fields ...zap.Field) {
	system.Error(message, fields...)
}

func Errorf(format string, v ...any) {
	msg := validPrintf(format, v...)
	system.Error(msg)
}

func Panic(message string, fields ...zap.Field) {
	system.Panic(message, fields...)
}

func Panicf(format string, v ...any) {
	msg := validPrintf(format, v...)
	system.Panic(msg)
}

func Printf(format string, v ...any) {
	msg := validPrintf(format, v...)
	system.Info(msg)
}

func Println(v ...string) {
	system.Info(strings.Join(v, " "))
}
