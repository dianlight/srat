package osutil_test

import (
	"fmt"
	"testing"

	"github.com/dianlight/srat/internal/osutil"
)

func TestPrintHashes(t *testing.T) {
	for _, p := range []string{"newpass"} {
		fmt.Printf("NTHash(%q) = %s\n", p, osutil.NTHash(p))
	}
}
