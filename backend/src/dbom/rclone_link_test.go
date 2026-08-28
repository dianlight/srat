package dbom

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRcloneLinkTableName(t *testing.T) {
	assert.Equal(t, "rclone_links", RcloneLink{}.TableName())
}
