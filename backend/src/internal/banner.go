package internal

import (
	"fmt"

	"github.com/dianlight/srat/config"
	"moul.io/banner"
)

func Banner(module string) {
	fmt.Println(banner.Inline(module))
	fmt.Printf("SambaNAS2 Rest Administration Interface\n")
	fmt.Printf("Version: %s\n", config.Version)
	fmt.Printf("Commit Hash: %s\n", config.CommitHash)
	fmt.Printf("Build Timestamp: %s\n", config.BuildTimestamp)
	fmt.Printf("Documentation: https://github.com/dianlight/SRAT\n\n")
}
