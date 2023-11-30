package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/MyChaOS87/automqttion.git/pkg/log/config"
)

type (
	Logger interface {
		InitLogger()
		Debug(args ...interface{})
		Debugf(template string, args ...interface{})
		Info(args ...interface{})
		Infof(template string, args ...interface{})
		Warn(args ...interface{})
		Warnf(template string, args ...interface{})
		Error(args ...interface{})
		Errorf(template string, args ...interface{})
		DPanic(args ...interface{})
		DPanicf(template string, args ...interface{})
		Fatal(args ...interface{})
		Fatalf(template string, args ...interface{})
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
	//nolint:gomnd
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2))

	l.sugarLogger = logger.Sugar()
}

func (l *logger) internalDebug(args ...interface{}) {
	l.sugarLogger.Debug(args...)
}

func (l *logger) internalDebugf(template string, args ...interface{}) {
	l.sugarLogger.Debugf(template, args...)
}

func (l *logger) internalInfo(args ...interface{}) {
	l.sugarLogger.Info(args...)
}

func (l *logger) internalInfof(template string, args ...interface{}) {
	l.sugarLogger.Infof(template, args...)
}

func (l *logger) internalWarn(args ...interface{}) {
	l.sugarLogger.Warn(args...)
}

func (l *logger) internalWarnf(template string, args ...interface{}) {
	l.sugarLogger.Warnf(template, args...)
}

func (l *logger) internalError(args ...interface{}) {
	l.sugarLogger.Error(args...)
}

func (l *logger) internalErrorf(template string, args ...interface{}) {
	l.sugarLogger.Errorf(template, args...)
}

func (l *logger) internalDPanic(args ...interface{}) {
	l.sugarLogger.DPanic(args...)
}

func (l *logger) internalDPanicf(template string, args ...interface{}) {
	l.sugarLogger.DPanicf(template, args...)
}

func (l *logger) internalPanic(args ...interface{}) {
	l.sugarLogger.Panic(args...)
}

func (l *logger) internalPanicf(template string, args ...interface{}) {
	l.sugarLogger.Panicf(template, args...)
}

func (l *logger) internalFatal(args ...interface{}) {
	l.sugarLogger.Fatal(args...)
}

func (l *logger) internalFatalf(template string, args ...interface{}) {
	l.sugarLogger.Fatalf(template, args...)
}

func (l *logger) Debug(args ...interface{}) {
	l.internalDebug(args...)
}

func (l *logger) Debugf(template string, args ...interface{}) {
	l.internalDebugf(template, args...)
}

func (l *logger) Info(args ...interface{}) {
	l.internalInfo(args...)
}

func (l *logger) Infof(template string, args ...interface{}) {
	l.internalInfof(template, args...)
}

func (l *logger) Warn(args ...interface{}) {
	l.internalWarn(args...)
}

func (l *logger) Warnf(template string, args ...interface{}) {
	l.internalWarnf(template, args...)
}

func (l *logger) Error(args ...interface{}) {
	l.internalError(args...)
}

func (l *logger) Errorf(template string, args ...interface{}) {
	l.internalErrorf(template, args...)
}

func (l *logger) DPanic(args ...interface{}) {
	l.internalDPanic(args...)
}

func (l *logger) DPanicf(template string, args ...interface{}) {
	l.internalDPanicf(template, args...)
}

func (l *logger) Panic(args ...interface{}) {
	l.internalPanic(args...)
}

func (l *logger) Panicf(template string, args ...interface{}) {
	l.internalPanicf(template, args...)
}

func (l *logger) Fatal(args ...interface{}) {
	l.internalFatal(args...)
}

func (l *logger) Fatalf(template string, args ...interface{}) {
	l.internalFatalf(template, args...)
}

func (l *logger) getLoggerLevel(cfg *config.Logger) zapcore.Level {
	level, exist := l.loggerLevelMap[cfg.Level]
	if !exist {
		return zapcore.DebugLevel
	}

	return level
}
