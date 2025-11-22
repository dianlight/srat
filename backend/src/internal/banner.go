package internal

import (
	"fmt"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/tlog"
	"github.com/fatih/color"
	"moul.io/banner"
)

func Banner(module string) {
	colTitle := color.New(color.FgHiCyan, color.Bold)
	colInfo := color.New(color.FgHiWhite)
	colLink := color.New(color.FgHiBlue, color.Underline)

	colTitle.Println(banner.Inline(module))
	colTitle.Println("SambaNAS2 Rest Administration Interface")
	colVersion := color.New(color.FgHiGreen, color.Bold)
	colHash := color.New(color.FgHiMagenta)
	colTime := color.New(color.FgHiCyan)

	colInfo.Print("Version: ")
	colVersion.Printf("%s ", config.Version)
	if config.CommitHash != "" {
		colHash.Printf("(%s) ", config.CommitHash)
	}
	if config.BuildTimestamp != "" {
		colTime.Println("- " + config.BuildTimestamp)
	} else {
		// ensure we end the line if no timestamp provided
		fmt.Println()
	}
	colInfo.Print("Documentation: ")
	colLink.Println("https://github.com/dianlight/SRAT")
	level := tlog.GetLevelString()
	colLevel := color.New(color.FgHiWhite)

	switch level {
	case "debug", "DEBUG", "Debug":
		colLevel = color.New(color.FgHiMagenta)
	case "info", "INFO", "Info":
		colLevel = color.New(color.FgHiGreen)
	case "warn", "warning", "WARN", "Warning", "WARNING":
		colLevel = color.New(color.FgHiYellow)
	case "error", "ERROR", "Error":
		colLevel = color.New(color.FgHiRed)
	case "fatal", "FATAL", "Fatal", "panic", "PANIC", "Panic":
		colLevel = color.New(color.FgHiRed, color.Bold)
	case "trace", "TRACE", "Trace":
		colLevel = color.New(color.FgHiCyan)
	default:
		colLevel = colInfo
	}

	colInfo.Print("Log level: ")
	colLevel.Println(level)
}
