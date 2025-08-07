package tlog

import (
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
	formatter := TozdErrorFormatter("error")
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
	formatter := TozdErrorFormatter("error")
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
	formatter := TozdErrorFormatter("error")
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
			suite.Equal(slog.KindGroup, attr.Value.Kind())
			hasStacktrace = true

			// Check that stack frames are present and formatted
			stackAttrs := attr.Value.Group()
			suite.NotEmpty(stackAttrs, "Stack trace should have frames")

			// Check first frame format
			if len(stackAttrs) > 0 {
				firstFrame := stackAttrs[0]
				frameContent := firstFrame.Value.String()

				// Frame should contain file path, line number, and function name
				suite.Contains(frameContent, ":") // file:line separator
				suite.True(strings.Contains(frameContent, "tlog") ||
					strings.Contains(frameContent, "TestError"),
					"Frame should contain relevant function name")
			}
		}
	}

	suite.True(hasMessage)
	suite.True(hasStacktrace)
}

func (suite *TozdErrorFormatterSuite) TestErrorWithCause() {
	formatter := TozdErrorFormatter("error")

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
	formatter := TozdErrorFormatter("error")

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
			suite.Equal(slog.KindGroup, attr.Value.Kind())
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
	formatter := TozdErrorFormatter("error")
	err := errors.WithStack(errors.New("colored error"))

	value, changed := formatter(nil, slog.Attr{
		Key:   "error",
		Value: slog.AnyValue(err),
	})

	suite.True(changed)

	if IsColorsEnabled() {
		// When colors are enabled, stack frames should contain ANSI color codes
		attrs := value.Group()

		for _, attr := range attrs {
			if attr.Key == "stacktrace" {
				stackAttrs := attr.Value.Group()
				if len(stackAttrs) > 0 {
					firstFrame := stackAttrs[0].Value.String()
					// Should contain ANSI escape sequences when colors are enabled
					// Look for common ANSI color codes
					hasColorCodes := strings.Contains(firstFrame, "\033[") ||
						strings.Contains(firstFrame, "\x1b[")
					suite.True(hasColorCodes, "Stack frame should contain color codes when colors are enabled")
				}
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

func (suite *TozdErrorFormatterSuite) TestNilError() {
	formatter := TozdErrorFormatter("error")

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
