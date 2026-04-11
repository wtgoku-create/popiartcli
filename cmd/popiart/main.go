package main

import (
	"os"

	"github.com/wtgoku-create/popiartcli/internal/cmd"
	"github.com/wtgoku-create/popiartcli/internal/output"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func buildVersion() string {
	v := version
	if commit != "" {
		v += " (" + commit + ")"
	}
	if date != "" {
		v += " built " + date
	}
	return v
}

func main() {
	root := cmd.NewRootCmd(buildVersion())
	if err := root.Execute(); err != nil {
		output.WriteErrorWithMode(os.Stderr, err, output.WantsPlainOutput(os.Args[1:]))
		os.Exit(output.ExitCode(err))
	}
}
