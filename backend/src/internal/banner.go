package internal

import (
	"fmt"

	"github.com/dianlight/srat/config"
	"moul.io/banner"
)

func Banner(module string) {
	fmt.Println(banner.Inline(module))
	fmt.Printf("SambaNAS2 Rest Administration Interface\n")
	fmt.Printf("Version: %s (%s) - %s\n", config.Version, config.CommitHash, config.BuildTimestamp)
	fmt.Printf("Documentation: https://github.com/dianlight/SRAT\n\n")
}
