package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMetadataVersion_Empty(t *testing.T) {
	require.Nil(t, parseMetadataVersion(nil))
	require.Nil(t, parseMetadataVersion([]byte{}))
}

func TestParseMetadataVersion_NullsOnly(t *testing.T) {
	require.Nil(t, parseMetadataVersion([]byte{0, 0, 0}))
}

func TestParseMetadataVersion_InvalidJSON(t *testing.T) {
	require.Nil(t, parseMetadataVersion([]byte("not-json")))
}

func TestParseMetadataVersion_MissingVersionKey(t *testing.T) {
	require.Nil(t, parseMetadataVersion([]byte("{}")))
	require.Nil(t, parseMetadataVersion([]byte("{\n \t\"name\": \"srat\" }")))
}

func TestParseMetadataVersion_Valid(t *testing.T) {
	v := parseMetadataVersion([]byte("{\"version\":\"1.2.3\"}"))
	require.NotNil(t, v)
	require.Equal(t, "1.2.3", v.String())
}

func TestParseMetadataVersion_ValidWithTrailingNulls(t *testing.T) {
	data := append([]byte("{\"version\":\"9.9.9\"}"), 0, 0, 0)
	v := parseMetadataVersion(data)
	require.NotNil(t, v)
	require.Equal(t, "9.9.9", v.String())
}
