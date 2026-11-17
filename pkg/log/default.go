package log

import (
	"sync/atomic"

	"go.uber.org/zap/zapcore"

	"github.com/MyChaOS87/automqttion/pkg/log/config"
)

const levelInfo = "info"

//nolint:gochecknoglobals
var defaultLogger atomic.Pointer[logger]

//nolint:gochecknoinits
func init() {
	l := &logger{
		cfg: &config.Logger{
			Development: true,
			Encoding:    "console",
			Level:       levelInfo,
		},
		loggerLevelMap: map[string]zapcore.Level{
			"debug":   zapcore.DebugLevel,
			levelInfo: zapcore.InfoLevel,
			"warn":    zapcore.WarnLevel,
			"error":   zapcore.ErrorLevel,
			"dpanic":  zapcore.DPanicLevel,
			"panic":   zapcore.PanicLevel,
			"fatal":   zapcore.FatalLevel,
		},
	}
	l.InitLogger()
	defaultLogger.Store(l)
}

func SetDefaultLogger(l Logger) {
	if log, ok := l.(*logger); ok {
		defaultLogger.Store(log)
	} else {
		defaultLogger.Load().DPanic("Set default logger needs an *logger as Logger")
	}
}

func Debug(args ...any) {
	defaultLogger.Load().internalDebug(args...)
}

func Debugf(template string, args ...any) {
	defaultLogger.Load().internalDebugf(template, args...)
}

func Info(args ...any) {
	defaultLogger.Load().internalInfo(args...)
}

func Infof(template string, args ...any) {
	defaultLogger.Load().internalInfof(template, args...)
}

func Warn(args ...any) {
	defaultLogger.Load().internalWarn(args...)
}

func Warnf(template string, args ...any) {
	defaultLogger.Load().internalWarnf(template, args...)
}

func Error(args ...any) {
	defaultLogger.Load().internalError(args...)
}

func Errorf(template string, args ...any) {
	defaultLogger.Load().internalErrorf(template, args...)
}

func DPanic(args ...any) {
	defaultLogger.Load().internalDPanic(args...)
}

func DPanicf(template string, args ...any) {
	defaultLogger.Load().internalDPanicf(template, args...)
}

func Fatal(args ...any) {
	defaultLogger.Load().internalFatal(args...)
}

func Fatalf(template string, args ...any) {
	defaultLogger.Load().internalFatalf(template, args...)
}
