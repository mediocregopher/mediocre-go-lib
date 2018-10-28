// Package m is the glue which holds all the other packages in this project
// together. While other packages in this project are intended to be able to be
// used separately and largely independently, this package combines them in ways
// which I specifically like.
package m

import (
	"github.com/mediocregopher/mediocre-go-lib/mcfg"
)

// CfgSource returns an mcfg.Source which takes in configuration info from the
// environment and from the CLI.
func CfgSource() mcfg.Source {
	return mcfg.Sources{
		mcfg.SourceEnv{},
		mcfg.SourceCLI{},
	}
}
