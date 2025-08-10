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
	//	if multiLine {
	//		return slog.String("stacktrace", frames.String())
	//	}
	// TODO: Support multi-line stack trace formatting if needed in the future.

	var stackLines []string

	for {
		frame, more := frames.Next()
		stackLines = append(stackLines, fmt.Sprintf("%s:%s: %s\n", color.GreenString(frame.File), color.BlueString(fmt.Sprintf("%d", frame.Line)), color.HiWhiteString(frame.Function)))
		if !more {
			break
		}
	}
	return strings.Join(stackLines, " ")
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
		//stack := stacktrace()
		var pcs []uintptr = make([]uintptr, 50)
		runtime.Callers(9, pcs[:])
		stack := runtime.CallersFrames(pcs)
		values := []slog.Attr{
			slog.String("message", err.Error()),
			slog.String("type", reflect.TypeOf(err).String()),
			slog.String("stacktrace", stackTraceFormatter(stack)),
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
				//frameIndex := 0

				/*
						// Determine if we should use tree formatting
						useTreeFormat := supportsUnicode() && IsColorsEnabled()

						// Tree characters for Unicode-supported terminals
						treeChars := struct {
							vertical   string
							branch     string
							lastBranch string
							space      string
						}{
							vertical:   "│ ",
							branch:     "├─ ",
							lastBranch: "└─ ",
							space:      "   ",
						}

						// Fallback ASCII characters for terminals without Unicode support
						if !useTreeFormat {
							treeChars.vertical = "| "
							treeChars.branch = "|- "
							treeChars.lastBranch = "`- "
							treeChars.space = "   "
						}

						var allFrames []string // Collect all frames first to determine which is last

						for {
							frame, more := frames.Next()
							frameInfo := fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function)
							allFrames = append(allFrames, frameInfo)
							frameIndex++

							if !more || frameIndex >= 20 {
								if frameIndex >= 20 && more {
									allFrames = append(allFrames, "... (truncated)")
								}
								break
							}
						}

						// Build the complete stacktrace as a single formatted string
						var stackLines []string

						for i, frameInfo := range allFrames {
							var prefix string
							var coloredFrameInfo string

							// Determine the tree prefix
							if len(allFrames) == 1 {
								prefix = ""
							} else if i == len(allFrames)-1 {
								prefix = treeChars.lastBranch
							} else {
								prefix = treeChars.branch
							}

							// Apply color formatting if colors are enabled
							if IsColorsEnabled() {
								var frameColor *color.Color
								if i == 0 {
									// Highlight the top frame (most recent) in red
									frameColor = color.New(color.FgRed, color.Bold)
								} else if i < 3 {
									// Highlight the next few frames in yellow
									frameColor = color.New(color.FgYellow)
								} else {
									// Use gray for deeper stack frames
									frameColor = color.New(color.FgHiBlack)
								}

								coloredFrameInfo = frameColor.Sprint(frameInfo)

								// Color the tree prefix too
								if prefix != "" {
									coloredPrefix := color.New(color.FgCyan).Sprint(prefix)
									coloredFrameInfo = coloredPrefix + coloredFrameInfo
								} else {
									coloredFrameInfo = prefix + coloredFrameInfo
								}
							} else {
								coloredFrameInfo = prefix + frameInfo
							}

							stackLines = append(stackLines, coloredFrameInfo)
						}

						// Check if multiline stacktrace is enabled
						formatterConfigMu.RLock()
						multilineEnabled := formatterConfig.MultilineStacktrace
						formatterConfigMu.RUnlock()

						if multilineEnabled {
							// For multiline output, add each frame as a separate log message
							// We'll format them as individual slog attributes with proper indentation
							var stackFrames []any
							for i, line := range stackLines {
								stackFrames = append(stackFrames, slog.String(fmt.Sprintf("frame_%d", i), line))
							}
							attrs = append(attrs, slog.Group("stacktrace", stackFrames...))
						} else {
							// Join all lines with newlines to create the single-line tree structure
							stacktraceString := strings.Join(stackLines, "\n")
							attrs = append(attrs, slog.String("stacktrace", stacktraceString))
						}
					}
				*/
				attrs = append(attrs, slog.String("stacktrace", stackTraceFormatter(frames)))
			}

		}

		// Add cause if available (for error chains)
		if cause := errors.Cause(v); cause != nil && cause != v {
			attrs = append(attrs, slog.String("cause", cause.Error()))
		}

		return slog.GroupValue(attrs...)
	})
}
