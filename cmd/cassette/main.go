package main

import (
	"flag"
	"fmt"
	"os"

	"cassette/buildinfo"
	"cassette/cli"
	"cassette/core/utils"
	ui "cassette/ui/v1"
)

var versionFlag bool

func init() {
	flag.BoolVar(&versionFlag, "version", false, "print build metadata")
}

func main() {
	flag.Parse()
	if versionFlag {
		_ = buildinfo.PrintVersion(os.Stdout)
		return
	}
	if flag.NArg() > 0 && flag.Arg(0) == "version" {
		cli.Run(flag.Args())
		return
	}
	if err := utils.ValidateStartupConfig(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	switch {
	case flag.NArg() > 0:
		if cli.Run(flag.Args()) {
			ui.RunTui()
		}
	default:
		ui.RunTui()
	}
}
