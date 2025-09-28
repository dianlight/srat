package tlog

import (
	"fmt"
	"log/slog"
	"reflect"
	"runtime"
	"strings"

	"github.com/fatih/color"
	slogformatter "github.com/samber/slog-formatter"
	"gitlab.com/tozd/go/errors"
)

func stackTraceFormatter(frames *runtime.Frames) string {
	var stackLines []string

	for {
		frame, more := frames.Next()
		stackLines = append(stackLines, fmt.Sprintf("%s:%s: %s", color.GreenString(frame.File), color.BlueString(fmt.Sprintf("%d", frame.Line)), color.HiWhiteString(frame.Function)))
		if !more {
			break
		}
	}

	if isMultilineStacktraceEnabled() {
		return strings.Join(stackLines, "\n")
	}

	return strings.Join(stackLines, " -> ")
}

func isMultilineStacktraceEnabled() bool {
	formatterConfigMu.RLock()
	defer formatterConfigMu.RUnlock()
	return formatterConfig.MultilineStacktrace
}

// ErrorFormatter transforms a go error into a readable error.
//
// Example:
//
//	err := reader.Close()
//	err = fmt.Errorf("could not close reader: %v", err)
//	logger.With("error", reader.Close()).Log("error")
//
// passed to ErrorFormatter("error"), will be transformed into:
//
//	"error": {
//	  "message": "could not close reader: file already closed",
//	  "type": "*io.ErrClosedPipe"
//	}
func ErrorFormatter(fieldName string) slogformatter.Formatter {
	return slogformatter.FormatByFieldType(fieldName, func(err error) slog.Value {
		var pcs []uintptr = make([]uintptr, 50)
		runtime.Callers(9, pcs[:])
		stack := runtime.CallersFrames(pcs)
		values := []slog.Attr{
			slog.String("message", err.Error()),
			slog.String("type", reflect.TypeOf(err).String()),
			slog.String("stacktrace", stackTraceFormatter(stack)),
			slog.Any("org_error", err),
		}
		return slog.GroupValue(values...)
	})
}

/*
// bearer:disable go_lang_permissive_regex_validation
var reStacktrace = regexp.MustCompile(`log/slog.*\n|tlog/tlog.*\n`)

func stacktrace() string {
    stackInfo := make([]byte, 1024*1024)

    if stackSize := runtime.Stack(stackInfo, false); stackSize > 0 {
        traceLines := reStacktrace.Split(string(stackInfo[:stackSize]), -1)
        if len(traceLines) > 0 {
            return traceLines[len(traceLines)-1]
        }
    }

    return ""
}
*/

// TozdErrorFormatter formats gitlab.com/tozd/go/errors with colored stacktraces
func TozdErrorFormatter() slogformatter.Formatter {
	return slogformatter.FormatByType(func(v errors.E) slog.Value {
		// Create formatted error information
		var attrs []slog.Attr

		// Add error message
		attrs = append(attrs, slog.String("message", v.Error()))

		// Check if error has details
		if details := errors.Details(v); len(details) > 0 {
			var detailAttrs []any
			for k, val := range details {
				detailAttrs = append(detailAttrs, slog.Any(k, val))
			}
			attrs = append(attrs, slog.Group("details", detailAttrs...))
		}

		// Check if error has a stack trace
		if stackTracer, ok := v.(interface{ StackTrace() []uintptr }); ok {
			stackTrace := stackTracer.StackTrace()
			if len(stackTrace) > 0 {
				// Use runtime.CallersFrames to get proper frame information
				frames := runtime.CallersFrames(stackTrace)
				attrs = append(attrs, slog.String("stacktrace", stackTraceFormatter(frames)))
			}
		}

		// Add cause if available (for error chains)
		if cause := errors.Cause(v); cause != nil && cause != v {
			attrs = append(attrs, slog.String("cause", cause.Error()))
		}

		attrs = append(attrs, slog.Any("org_error", v))

		return slog.GroupValue(attrs...)
	})
}
