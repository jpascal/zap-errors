package errors

import (
	"bytes"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"runtime"
)

type Error struct {
	message    string
	payload    interface{}
	code       int
	stacktrace []*runtime.Frame
	err        error
}

func (ee Error) Error() string {
	return ee.message
}

func (ee Error) Unwrap() error {
	return ee.err
}

func (ee Error) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	if len(ee.message) > 0 {
		encoder.AddString("message", ee.err.Error())
	}
	if len(ee.stacktrace) > 0 {
		buffer := bytes.NewBuffer([]byte{})
		for _, frame := range ee.stacktrace {
			_, _ = fmt.Fprintf(buffer, "%s\t\n%s:%d\n", frame.Function, frame.File, frame.Line)
		}
		encoder.AddString("stacktrace", buffer.String())
	}
	if ee.payload != nil {
		if err := encoder.AddReflected("payload", ee.payload); err != nil {
			return err
		}
	}
	return nil
}

func Errorf(format string, a ...interface{}) Error {
	return Error{
		err:        fmt.Errorf(format, a...),
		stacktrace: stackTrace(),
		message:    fmt.Sprintf(format, a...),
	}
}

func WithMessage(err error, format string, a ...interface{}) Error {
	var parentEnhancedError Error
	if errors.As(err, &parentEnhancedError) {
		if parentEnhancedError.stacktrace == nil {
			parentEnhancedError.stacktrace = stackTrace()
		}
		parentEnhancedError.message = fmt.Sprintf(format, a...) + ": " + parentEnhancedError.message
		return parentEnhancedError
	}
	return Error{
		err:        err,
		stacktrace: stackTrace(),
		message:    fmt.Sprintf(format, a...),
	}
}

func (ee Error) WithPayload(payload interface{}) Error {
	ee.payload = payload
	return ee
}

func (ee Error) WithCode(code int) Error {
	ee.code = code
	return ee
}

func (ee Error) WithStacktrace() Error {
	ee.stacktrace = stackTrace()
	return ee
}

func stackTrace() []*runtime.Frame {
	pc := make([]uintptr, 10)
	n := runtime.Callers(0, pc)
	pc = pc[3:n]
	frames := runtime.CallersFrames(pc)
	traceFrames := make([]*runtime.Frame, 0)
	for {
		frame, more := frames.Next()
		if !more {
			break
		}
		traceFrames = append(traceFrames, &frame)
	}
	return traceFrames
}

func Field(err error) zap.Field {
	var ee Error
	if errors.As(err, &ee) {
		return zap.Object("error", ee)
	} else if err != nil {
		return zap.Any("error", map[string]string{"message": err.Error()})
	}
	return zap.Skip()
}

func As(err error, target interface{}) bool {
	return errors.As(err, &target)
}

func Unwrap(err error) error {
	if errors.As(err, &Error{}) {
		return errors.Unwrap(err)
	}
	return err
}

func Log(logger *zap.Logger, err error) {
	logger.With(Field(err)).Error(err.Error())
}
