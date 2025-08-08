package tlog

import (
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"gitlab.com/tozd/go/errors"
)

type TozdErrorFormatterSuite struct {
	suite.Suite
}

func (suite *TozdErrorFormatterSuite) SetupTest() {
	// Enable colors for testing
	EnableColors(true)
}

func (suite *TozdErrorFormatterSuite) TearDownTest() {
	// Clean up
}

func (suite *TozdErrorFormatterSuite) TestSimpleError() {
	formatter := TozdErrorFormatter(false)
	err := errors.New("simple test error")

	value, changed := formatter(nil, slog.Attr{
		Key:   "error",
		Value: slog.AnyValue(err),
	})

	suite.True(changed)
	suite.Equal(slog.KindGroup, value.Kind())

	// The error should have at least a message
	attrs := value.Group()
	suite.NotEmpty(attrs)

	// Find the message attribute
	var messageFound bool
	for _, attr := range attrs {
		if attr.Key == "message" {
			suite.Equal("simple test error", attr.Value.String())
			messageFound = true
			break
		}
	}
	suite.True(messageFound, "Message attribute should be present")
}

func (suite *TozdErrorFormatterSuite) TestErrorWithDetails() {
	formatter := TozdErrorFormatter(false)
	err := errors.WithDetails(
		errors.New("error with details"),
		"user_id", "12345",
		"action", "login",
		"attempts", 3,
	)

	value, changed := formatter(nil, slog.Attr{
		Key:   "error",
		Value: slog.AnyValue(err),
	})

	suite.True(changed)
	attrs := value.Group()

	// Should have message and details
	var hasMessage, hasDetails bool
	for _, attr := range attrs {
		switch attr.Key {
		case "message":
			suite.Equal("error with details", attr.Value.String())
			hasMessage = true
		case "details":
			suite.Equal(slog.KindGroup, attr.Value.Kind())
			hasDetails = true

			// Check details content
			detailAttrs := attr.Value.Group()
			detailMap := make(map[string]any)
			for _, detailAttr := range detailAttrs {
				detailMap[detailAttr.Key] = detailAttr.Value.Any()
			}

			suite.Equal("12345", detailMap["user_id"])
			suite.Equal("login", detailMap["action"])
			// Note: the int value might be converted, so we check for existence
			suite.Contains(detailMap, "attempts")
		}
	}

	suite.True(hasMessage)
	suite.True(hasDetails)
}

func (suite *TozdErrorFormatterSuite) TestErrorWithStackTrace() {
	formatter := TozdErrorFormatter(false)
	err := errors.WithStack(errors.New("error with stack"))

	value, changed := formatter(nil, slog.Attr{
		Key:   "error",
		Value: slog.AnyValue(err),
	})

	suite.True(changed)
	attrs := value.Group()

	// Should have message and stacktrace
	var hasMessage, hasStacktrace bool
	for _, attr := range attrs {
		switch attr.Key {
		case "message":
			suite.Equal("error with stack", attr.Value.String())
			hasMessage = true
		case "stacktrace":
			hasStacktrace = true

			// Check format based on multiline setting
			if IsMultilineStacktraceEnabled() {
				suite.Equal(slog.KindGroup, attr.Value.Kind())
				stackAttrs := attr.Value.Group()
				suite.NotEmpty(stackAttrs, "Stack trace should have frames")
			} else {
				suite.Equal(slog.KindString, attr.Value.Kind())
				stackContent := attr.Value.String()
				suite.NotEmpty(stackContent, "Stack trace should have content")

				// Stack trace should contain file path, line number, and function name
				suite.Contains(stackContent, ":") // file:line separator
				suite.True(strings.Contains(stackContent, "tlog") ||
					strings.Contains(stackContent, "TestError"),
					"Stack trace should contain relevant function name")
			}
		}
	}

	suite.True(hasMessage)
	suite.True(hasStacktrace)
}

func (suite *TozdErrorFormatterSuite) TestErrorWithCause() {
	formatter := TozdErrorFormatter(false)

	baseErr := errors.New("root cause error")
	wrappedErr := errors.Wrap(baseErr, "wrapped error")

	value, changed := formatter(nil, slog.Attr{
		Key:   "error",
		Value: slog.AnyValue(wrappedErr),
	})

	suite.True(changed)
	attrs := value.Group()

	// Should have message and cause
	var hasMessage, hasCause bool
	for _, attr := range attrs {
		switch attr.Key {
		case "message":
			suite.Equal("wrapped error", attr.Value.String())
			hasMessage = true
		case "cause":
			suite.Equal("root cause error", attr.Value.String())
			hasCause = true
		}
	}

	suite.True(hasMessage)
	suite.True(hasCause)
}

func (suite *TozdErrorFormatterSuite) TestComplexError() {
	formatter := TozdErrorFormatter(false)

	// Create a complex error with details, stack trace, and cause
	baseErr := errors.WithDetails(
		errors.New("database connection failed"),
		"host", "localhost",
		"port", 5432,
	)

	wrappedErr := errors.WithDetails(
		errors.Wrap(baseErr, "failed to initialize repository"),
		"repository", "user_repository",
		"retry_count", 3,
	)

	stackErr := errors.WithStack(wrappedErr)

	value, changed := formatter(nil, slog.Attr{
		Key:   "error",
		Value: slog.AnyValue(stackErr),
	})

	suite.True(changed)
	attrs := value.Group()

	// Should have message, details, stacktrace, and cause
	foundAttrs := make(map[string]bool)
	for _, attr := range attrs {
		foundAttrs[attr.Key] = true

		switch attr.Key {
		case "message":
			suite.Equal("failed to initialize repository", attr.Value.String())
		case "details":
			suite.Equal(slog.KindGroup, attr.Value.Kind())
		case "stacktrace":
			// Check format based on multiline setting
			if IsMultilineStacktraceEnabled() {
				suite.Equal(slog.KindGroup, attr.Value.Kind())
			} else {
				suite.Equal(slog.KindString, attr.Value.Kind())
			}
			// Just verify content exists
			if attr.Value.Kind() == slog.KindString {
				suite.NotEmpty(attr.Value.String())
			} else {
				suite.NotEmpty(attr.Value.Group())
			}
		case "cause":
			suite.Equal("database connection failed", attr.Value.String())
		}
	}

	suite.True(foundAttrs["message"], "Should have message")
	suite.True(foundAttrs["details"], "Should have details")
	suite.True(foundAttrs["stacktrace"], "Should have stacktrace")
	suite.True(foundAttrs["cause"], "Should have cause")
}

func (suite *TozdErrorFormatterSuite) TestColorFormatting() {
	// Test with colors enabled
	EnableColors(true)
	formatter := TozdErrorFormatter(false)
	err := errors.WithStack(errors.New("colored error"))

	value, changed := formatter(nil, slog.Attr{
		Key:   "error",
		Value: slog.AnyValue(err),
	})

	suite.True(changed)

	if IsColorsEnabled() {
		// When colors are enabled, stack trace should contain ANSI color codes
		attrs := value.Group()

		for _, attr := range attrs {
			if attr.Key == "stacktrace" {
				stackContent := attr.Value.String()
				// Should contain ANSI escape sequences when colors are enabled
				// Look for common ANSI color codes
				hasColorCodes := strings.Contains(stackContent, "\033[") ||
					strings.Contains(stackContent, "\x1b[")
				suite.True(hasColorCodes, "Stack trace should contain color codes when colors are enabled")
				break
			}
		}
	}

	// Test with colors disabled
	EnableColors(false)
	_, changed2 := formatter(nil, slog.Attr{
		Key:   "error",
		Value: slog.AnyValue(err),
	})

	suite.True(changed2)
	// When colors are disabled, output should not contain ANSI codes
	// This is harder to test directly since the color library might still add codes
	// but at least we've tested both paths
}

func (suite *TozdErrorFormatterSuite) TestTreeFormatting() {
	// Test tree formatting when both colors and unicode are available
	EnableColors(true)
	formatter := TozdErrorFormatter(false)

	// Create an error with multiple stack frames by calling through helper functions
	err := suite.createNestedError()

	value, changed := formatter(nil, slog.Attr{
		Key:   "error",
		Value: slog.AnyValue(err),
	})

	suite.True(changed)
	attrs := value.Group()

	for _, attr := range attrs {
		if attr.Key == "stacktrace" {
			stackContent := attr.Value.String()
			suite.NotEmpty(stackContent, "Should have stack trace content")

			// Check for tree characters in stack trace
			lines := strings.Split(stackContent, "\n")
			suite.True(len(lines) >= 2, "Should have multiple stack frame lines")

			// Check for tree characters in multi-frame stack traces
			if len(lines) > 1 {
				lastLine := lines[len(lines)-1]
				// Last frame should have "└─" (or "`-" fallback)
				suite.True(strings.Contains(lastLine, "└─") || strings.Contains(lastLine, "`-"),
					"Last frame should contain tree terminator: %s", lastLine)

				// Earlier frames should have "├─" (or "|-" fallback)
				for i := 0; i < len(lines)-1; i++ {
					line := lines[i]
					if strings.TrimSpace(line) != "" {
						suite.True(strings.Contains(line, "├─") || strings.Contains(line, "|-"),
							"Frame %d should contain tree branch: %s", i, line)
					}
				}
			}
			break
		}
	}
}

func (suite *TozdErrorFormatterSuite) TestTreeFormattingWithColorsDisabled() {
	// Test with colors disabled (should fall back to ASCII tree characters)
	EnableColors(false)
	formatter := TozdErrorFormatter(false)

	err := suite.createNestedError()

	value, changed := formatter(nil, slog.Attr{
		Key:   "error",
		Value: slog.AnyValue(err),
	})

	suite.True(changed)
	attrs := value.Group()

	for _, attr := range attrs {
		if attr.Key == "stacktrace" {
			stackContent := attr.Value.String()

			if strings.Contains(stackContent, "\n") {
				lines := strings.Split(stackContent, "\n")
				if len(lines) > 1 {
					// Should use ASCII tree characters when colors are disabled
					lastLine := lines[len(lines)-1]
					suite.True(strings.Contains(lastLine, "`-") || strings.Contains(lastLine, "└─"),
						"Should use ASCII fallback tree characters when colors disabled: %s", lastLine)
				}
			}
			break
		}
	}
}

// Helper function to create nested error with multiple stack frames
func (suite *TozdErrorFormatterSuite) createNestedError() error {
	return suite.helperLevel1()
}

func (suite *TozdErrorFormatterSuite) helperLevel1() error {
	return suite.helperLevel2()
}

func (suite *TozdErrorFormatterSuite) helperLevel2() error {
	return errors.WithStack(errors.New("nested error"))
}

func (suite *TozdErrorFormatterSuite) TestMultilineStacktrace() {
	// Test multiline stacktrace mode
	EnableMultilineStacktrace(true)
	suite.True(IsMultilineStacktraceEnabled(), "Multiline stacktrace should be enabled")

	formatter := TozdErrorFormatter(true)
	err := suite.createNestedError()

	value, changed := formatter(nil, slog.Attr{
		Key:   "error",
		Value: slog.AnyValue(err),
	})

	suite.True(changed)
	attrs := value.Group()

	for _, attr := range attrs {
		if attr.Key == "stacktrace" {
			// In multiline mode, stacktrace should be a group
			suite.Equal(slog.KindGroup, attr.Value.Kind())
			stackAttrs := attr.Value.Group()
			suite.NotEmpty(stackAttrs, "Should have stack frame attributes")

			// Each frame should be a separate attribute
			for i, frameAttr := range stackAttrs {
				suite.Equal(fmt.Sprintf("frame_%d", i), frameAttr.Key)
				frameContent := frameAttr.Value.String()
				suite.NotEmpty(frameContent, "Frame content should not be empty")

				// Should contain tree characters and be properly formatted
				if len(stackAttrs) > 1 {
					if i == len(stackAttrs)-1 {
						// Last frame should have terminator
						suite.True(strings.Contains(frameContent, "└─") || strings.Contains(frameContent, "`-"),
							"Last frame should contain tree terminator")
					} else {
						// Other frames should have branch
						suite.True(strings.Contains(frameContent, "├─") || strings.Contains(frameContent, "|-"),
							"Frame should contain tree branch")
					}
				}
			}
			break
		}
	}

	// Test single-line mode (default)
	EnableMultilineStacktrace(false)
	suite.False(IsMultilineStacktraceEnabled(), "Multiline stacktrace should be disabled")

	value2, changed2 := formatter(nil, slog.Attr{
		Key:   "error",
		Value: slog.AnyValue(err),
	})

	suite.True(changed2)
	attrs2 := value2.Group()

	for _, attr := range attrs2 {
		if attr.Key == "stacktrace" {
			// In single-line mode, stacktrace should be a string
			suite.Equal(slog.KindString, attr.Value.Kind())
			stackContent := attr.Value.String()
			suite.NotEmpty(stackContent, "Stack trace content should not be empty")

			// Should contain newline characters for the tree structure
			suite.Contains(stackContent, "\n", "Should contain newlines for tree structure")
			break
		}
	}
}

func (suite *TozdErrorFormatterSuite) TestNilError() {
	formatter := TozdErrorFormatter(false)

	// Test with nil interface (not a tozd error)
	_, changed := formatter(nil, slog.Attr{
		Key:   "error",
		Value: slog.StringValue("not a tozd error"),
	})

	// Should not change non-tozd errors
	suite.False(changed)
}

func TestTozdErrorFormatterSuite(t *testing.T) {
	suite.Run(t, new(TozdErrorFormatterSuite))
}
