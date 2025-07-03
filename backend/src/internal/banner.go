package internal

import (
	"fmt"

	"github.com/common-nighthawk/go-figure"
	"github.com/dianlight/srat/config"
)

func Banner(module string, command string) {
	//fmt.Println(banner.Inline(module))
	figure.NewColorFigure(module, "bulbhead", "green", true).Print()
	if command != "" {
		fmt.Printf("( %s )\n", command)
	}
	fmt.Printf("SambaNAS2 Rest Administration Interface\n")
	fmt.Printf("Version: %s (%s) - %s\n", config.Version, config.CommitHash, config.BuildTimestamp)
	fmt.Printf("Documentation: https://github.com/dianlight/SRAT\n\n")
}
