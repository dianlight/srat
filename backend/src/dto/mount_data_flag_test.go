package dto

import (
	"encoding/json"
	"testing"
)

func TestMounDataFlagsValue(t *testing.T) {
	var flags MounDataFlags

	flags.Add(MS_RDONLY)
	flags.Add(MS_NOSUID)

	value, err := flags.Value()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := int64(3)
	if value.(int64) != expected {
		t.Errorf("Expected %d, but got %d", expected, value.(int64))
	}

}

func TestMounDataFlagsScan(t *testing.T) {
	var flags MounDataFlags
	var expectedFlags MounDataFlags
	expectedFlags.Add(MS_RDONLY)

	value := 1

	err := flags.Scan(value)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(flags) != 1 {
		t.Errorf("Expected %d, but got %d", 1, len(flags))
	}

	if flags[0] != expectedFlags[0] {
		t.Errorf("Expected %d, but got %d", expectedFlags[0], flags[0])
	}
}

func TestMounDataFlagsMarshalJSON(t *testing.T) {
	var flags MounDataFlags

	flags.Add(MS_RDONLY)
	flags.Add(MS_REMOUNT)
	flags.Add(MS_NOUSER)

	json1, err := flags.MarshalJSON()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	json2, err := json.Marshal(flags)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if string(json1) != string(json2) {
		t.Errorf("Expected %s, but got %s", string(json2), string(json1))
	}

}

func TestMounDataFlagsUnMarshalJSON(t *testing.T) {
	var flags MounDataFlags
	flags.Add(MS_RDONLY)
	flags.Add(MS_REMOUNT)
	flags.Add(MS_NOUSER)

	json1, err := flags.MarshalJSON()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	var flags2 MounDataFlags
	err = flags2.UnmarshalJSON(json1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	var flags3 MounDataFlags
	err = json.Unmarshal(json1, &flags3)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(flags2) != len(flags) {
		t.Errorf("Direct Expected %d, but got %d", len(flags), len(flags2))
	}

	if len(flags3) != len(flags) {
		t.Errorf("JSON Expected %d, but got %d", len(flags), len(flags3))
	}

	for i, flag := range flags {
		if flags2[i] != flag {
			t.Errorf("Direct Expected %d, but got %d", flag, flags2[i])
		}
		if flags3[i] != flag {
			t.Errorf("JSON Expected %d, but got %d", flag, flags3[i])
		}
	}

	// t.Logf("JSON: %v, Direct %v", flags2, flags3)
	// t.Error("Test not implemented")
}
