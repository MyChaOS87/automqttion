package log

import (
	"go.uber.org/zap/zapcore"

	"github.com/MyChaOS87/automqttion.git/pkg/log/config"
)

//nolint:gochecknoglobals
var defaultLogger = &logger{
	cfg: &config.Logger{
		Development: true,
		Encoding:    "console",
		Level:       "info",
	},
	loggerLevelMap: map[string]zapcore.Level{
		"debug":  zapcore.DebugLevel,
		"info":   zapcore.InfoLevel,
		"warn":   zapcore.WarnLevel,
		"error":  zapcore.ErrorLevel,
		"dpanic": zapcore.DPanicLevel,
		"panic":  zapcore.PanicLevel,
		"fatal":  zapcore.FatalLevel,
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

func Debug(args ...interface{}) {
	defaultLogger.internalDebug(args...)
}

func Debugf(template string, args ...interface{}) {
	defaultLogger.internalDebugf(template, args...)
}

func Info(args ...interface{}) {
	defaultLogger.internalInfo(args...)
}

func Infof(template string, args ...interface{}) {
	defaultLogger.internalInfof(template, args...)
}

func Warn(args ...interface{}) {
	defaultLogger.internalWarn(args...)
}

func Warnf(template string, args ...interface{}) {
	defaultLogger.internalWarnf(template, args...)
}

func Error(args ...interface{}) {
	defaultLogger.internalError(args...)
}

func Errorf(template string, args ...interface{}) {
	defaultLogger.internalErrorf(template, args...)
}

func DPanic(args ...interface{}) {
	defaultLogger.internalDPanic(args...)
}

func DPanicf(template string, args ...interface{}) {
	defaultLogger.internalDPanicf(template, args...)
}

func Fatal(args ...interface{}) {
	defaultLogger.internalFatal(args...)
}

func Fatalf(template string, args ...interface{}) {
	defaultLogger.internalFatalf(template, args...)
}
