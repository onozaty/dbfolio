// Package cli defines the dbfolio command line interface.
package cli

import (
	"runtime/debug"

	"github.com/spf13/cobra"
)

// NewRootCmd builds the root command. version is injected at build time via
// ldflags; an empty value falls back to the module version recorded by go install.
func NewRootCmd(version string) *cobra.Command {
	v := resolveVersion(version)

	root := &cobra.Command{
		Use:   "dbfolio",
		Short: "Generate browsable static HTML database documentation from tbls schema.json",
		// Usage is noise for runtime errors such as a missing file; argument
		// errors still print it because cobra handles them before RunE.
		SilenceUsage: true,
		Version:      v,
	}
	root.AddCommand(newBuildCmd(), newVersionCmd(v))
	return root
}

func resolveVersion(version string) string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
