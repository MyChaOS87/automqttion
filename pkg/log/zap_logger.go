package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/MyChaOS87/automqttion/pkg/log/config"
)

type (
	Logger interface {
		InitLogger()
		Debug(args ...any)
		Debugf(template string, args ...any)
		Info(args ...any)
		Infof(template string, args ...any)
		Warn(args ...any)
		Warnf(template string, args ...any)
		Error(args ...any)
		Errorf(template string, args ...any)
		DPanic(args ...any)
		DPanicf(template string, args ...any)
		Fatal(args ...any)
		Fatalf(template string, args ...any)
	}

	logger struct {
		cfg            *config.Logger
		sugarLogger    *zap.SugaredLogger
		loggerLevelMap map[string]zapcore.Level
	}
)

// NewLogger constructor.
func NewLogger(cfg *config.Logger) Logger {
	return &logger{
		cfg: cfg,
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
}

func (l *logger) InitLogger() {
	logLevel := l.getLoggerLevel(l.cfg)

	var encoderCfg zapcore.EncoderConfig
	if l.cfg.Development {
		encoderCfg = zap.NewDevelopmentEncoderConfig()
	} else {
		encoderCfg = zap.NewProductionEncoderConfig()
	}

	var encoder zapcore.Encoder

	encoderCfg.LevelKey = "LEVEL"
	encoderCfg.CallerKey = "CALLER"
	encoderCfg.TimeKey = "TIME"
	encoderCfg.NameKey = "NAME"
	encoderCfg.MessageKey = "MESSAGE"

	if l.cfg.Encoding == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(encoder, os.Stdout, zap.NewAtomicLevelAt(logLevel))

	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2))

	l.sugarLogger = logger.Sugar()
}

func (l *logger) internalDebug(args ...any) {
	l.sugarLogger.Debug(args...)
}

func (l *logger) internalDebugf(template string, args ...any) {
	l.sugarLogger.Debugf(template, args...)
}

func (l *logger) internalInfo(args ...any) {
	l.sugarLogger.Info(args...)
}

func (l *logger) internalInfof(template string, args ...any) {
	l.sugarLogger.Infof(template, args...)
}

func (l *logger) internalWarn(args ...any) {
	l.sugarLogger.Warn(args...)
}

func (l *logger) internalWarnf(template string, args ...any) {
	l.sugarLogger.Warnf(template, args...)
}

func (l *logger) internalError(args ...any) {
	l.sugarLogger.Error(args...)
}

func (l *logger) internalErrorf(template string, args ...any) {
	l.sugarLogger.Errorf(template, args...)
}

func (l *logger) internalDPanic(args ...any) {
	l.sugarLogger.DPanic(args...)
}

func (l *logger) internalDPanicf(template string, args ...any) {
	l.sugarLogger.DPanicf(template, args...)
}

func (l *logger) internalPanic(args ...any) {
	l.sugarLogger.Panic(args...)
}

func (l *logger) internalPanicf(template string, args ...any) {
	l.sugarLogger.Panicf(template, args...)
}

func (l *logger) internalFatal(args ...any) {
	l.sugarLogger.Fatal(args...)
}

func (l *logger) internalFatalf(template string, args ...any) {
	l.sugarLogger.Fatalf(template, args...)
}

func (l *logger) Debug(args ...any) {
	l.internalDebug(args...)
}

func (l *logger) Debugf(template string, args ...any) {
	l.internalDebugf(template, args...)
}

func (l *logger) Info(args ...any) {
	l.internalInfo(args...)
}

func (l *logger) Infof(template string, args ...any) {
	l.internalInfof(template, args...)
}

func (l *logger) Warn(args ...any) {
	l.internalWarn(args...)
}

func (l *logger) Warnf(template string, args ...any) {
	l.internalWarnf(template, args...)
}

func (l *logger) Error(args ...any) {
	l.internalError(args...)
}

func (l *logger) Errorf(template string, args ...any) {
	l.internalErrorf(template, args...)
}

func (l *logger) DPanic(args ...any) {
	l.internalDPanic(args...)
}

func (l *logger) DPanicf(template string, args ...any) {
	l.internalDPanicf(template, args...)
}

func (l *logger) Panic(args ...any) {
	l.internalPanic(args...)
}

func (l *logger) Panicf(template string, args ...any) {
	l.internalPanicf(template, args...)
}

func (l *logger) Fatal(args ...any) {
	l.internalFatal(args...)
}

func (l *logger) Fatalf(template string, args ...any) {
	l.internalFatalf(template, args...)
}

func (l *logger) getLoggerLevel(cfg *config.Logger) zapcore.Level {
	level, exist := l.loggerLevelMap[cfg.Level]
	if !exist {
		return zapcore.DebugLevel
	}

	return level
}
