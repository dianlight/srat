package dbom

import (
	"testing"
)

func TestMounDataFlagsScanString(t *testing.T) {
	var flags MounDataFlags
	value := "ro,noexec,uid=1000,gid=1000"

	err := flags.Scan(value)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	expectedFlags := MounDataFlags{
		{Name: "ro", NeedsValue: false},
		{Name: "noexec", NeedsValue: false},
		{Name: "uid", NeedsValue: true, FlagValue: "1000"},
		{Name: "gid", NeedsValue: true, FlagValue: "1000"},
	}

	if len(flags) != len(expectedFlags) {
		t.Fatalf("Expected %d flags, got %d", len(expectedFlags), len(flags))
	}

	for i := range expectedFlags {
		if flags[i] != expectedFlags[i] {
			t.Errorf("Expected flag %+v, got %+v", expectedFlags[i], flags[i])
		}
	}
}

func TestMounDataFlagsScanNil(t *testing.T) {
	var flags MounDataFlags
	err := flags.Scan(nil)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(flags) != 0 {
		t.Errorf("Expected 0 flags, got %d", len(flags))
	}
}

func TestMounDataFlagsScanInvalidType(t *testing.T) {
	var flags MounDataFlags
	value := 123 // Invalid type

	err := flags.Scan(value)
	if err == nil {
		t.Fatalf("Scan should have failed with invalid type")
	}

	expectedErrorMessage := "invalid value type for MounDataFlags: int"
	if err.Error() != expectedErrorMessage {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrorMessage, err.Error())
	}
}

func TestMounDataFlagsScanEmptyString(t *testing.T) {
	var flags MounDataFlags
	value := ""

	err := flags.Scan(value)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(flags) != 0 {
		t.Errorf("Expected 0 flags, got %d", len(flags))
	}
}

func TestMounDataFlagsScanOnlyComma(t *testing.T) {
	var flags MounDataFlags
	value := ","

	err := flags.Scan(value)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(flags) != 0 {
		t.Errorf("Expected 0 flags, got %d", len(flags))
	}
}
func TestMounDataFlagsValue(t *testing.T) {
	flags := MounDataFlags{
		{Name: "ro", NeedsValue: false},
		{Name: "noexec", NeedsValue: false},
		{Name: "uid", NeedsValue: true, FlagValue: "1000"},
		{Name: "gid", NeedsValue: true, FlagValue: "1000"},
	}

	value, err := flags.Value()
	if err != nil {
		t.Fatalf("Value failed: %v", err)
	}

	expectedValue := "gid=1000,noexec,ro,uid=1000"
	if value != expectedValue {
		t.Errorf("Expected value '%s', got '%s'", expectedValue, value)
	}
}

func TestMounDataFlagsValueEmpty(t *testing.T) {
	var flags MounDataFlags

	value, err := flags.Value()
	if err != nil {
		t.Fatalf("Value failed: %v", err)
	}

	expectedValue := ""
	if value != expectedValue {
		t.Errorf("Expected value '%s', got '%s'", expectedValue, value)
	}
}

func TestMounDataFlagsValueSingle(t *testing.T) {
	flags := MounDataFlags{
		{Name: "ro", NeedsValue: false},
	}

	value, err := flags.Value()
	if err != nil {
		t.Fatalf("Value failed: %v", err)
	}

	expectedValue := "ro"
	if value != expectedValue {
		t.Errorf("Expected value '%s', got '%s'", expectedValue, value)
	}
}

func TestMounDataFlagsValueSingleWithValue(t *testing.T) {
	flags := MounDataFlags{
		{Name: "uid", NeedsValue: true, FlagValue: "1000"},
	}

	value, err := flags.Value()
	if err != nil {
		t.Fatalf("Value failed: %v", err)
	}

	expectedValue := "uid=1000"
	if value != expectedValue {
		t.Errorf("Expected value '%s', got '%s'", expectedValue, value)
	}
}
