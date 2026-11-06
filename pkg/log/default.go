package log

import (
	"go.uber.org/zap/zapcore"

	"github.com/MyChaOS87/automqttion/pkg/log/config"
)

const levelInfo = "info"

//nolint:gochecknoglobals
var defaultLogger = &logger{
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

//nolint:gochecknoinits
func init() {
	defaultLogger.InitLogger()
}

func SetDefaultLogger(l Logger) {
	if log, ok := l.(*logger); ok {
		defaultLogger = log
	} else {
		defaultLogger.DPanic("Set default logger needs an *logger as Logger")
	}
}

func Debug(args ...any) {
	defaultLogger.internalDebug(args...)
}

func Debugf(template string, args ...any) {
	defaultLogger.internalDebugf(template, args...)
}

func Info(args ...any) {
	defaultLogger.internalInfo(args...)
}

func Infof(template string, args ...any) {
	defaultLogger.internalInfof(template, args...)
}

func Warn(args ...any) {
	defaultLogger.internalWarn(args...)
}

func Warnf(template string, args ...any) {
	defaultLogger.internalWarnf(template, args...)
}

func Error(args ...any) {
	defaultLogger.internalError(args...)
}

func Errorf(template string, args ...any) {
	defaultLogger.internalErrorf(template, args...)
}

func DPanic(args ...any) {
	defaultLogger.internalDPanic(args...)
}

func DPanicf(template string, args ...any) {
	defaultLogger.internalDPanicf(template, args...)
}

func Fatal(args ...any) {
	defaultLogger.internalFatal(args...)
}

func Fatalf(template string, args ...any) {
	defaultLogger.internalFatalf(template, args...)
}
