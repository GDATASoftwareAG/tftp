package main

import (
	"fmt"

	"github.com/gdatasoftwareag/tftp/v2/pkg/tftp"
	"github.com/spf13/cobra"
)

var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(_ *cobra.Command, _ []string) {
			println(fmt.Sprintf("%s - %s (%s)", tftp.GitTag, tftp.GitCommit, tftp.BuildTime))
		},
	}
)
