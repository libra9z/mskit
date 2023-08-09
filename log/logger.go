package log

import (
	l4g "github.com/libra9z/log4go"
)

type Logger struct {
	log l4g.Logger
}

var Mslog Logger

func init() {
	Mslog = Logger{log: l4g.NewDefaultLogger(l4g.FINEST)}
}

func (l *Logger) Finest(arg0 interface{}, args ...interface{}) {
	l.log.Finest(arg0, args)
}

func (l *Logger) Fine(arg0 interface{}, args ...interface{}) {
	l.log.Fine(arg0, args)
}

func (l *Logger) Debug(arg0 interface{}, args ...interface{}) {
	l.log.Debug(arg0, args)
}

func (l *Logger) Error(arg0 interface{}, args ...interface{}) {
	l.log.Error(arg0, args)
}

func (l *Logger) Trace(arg0 interface{}, args ...interface{}) {
	l.log.Trace(arg0, args)
}

func (l *Logger) Info(arg0 interface{}, args ...interface{}) {
	l.log.Info(arg0, args)
}
func (l *Logger) Warn(arg0 interface{}, args ...interface{}) {
	l.log.Warn(arg0, args)
}

func (l *Logger) Critical(arg0 interface{}, args ...interface{}) {
	l.log.Critical(arg0, args)
}
