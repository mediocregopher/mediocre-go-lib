// Package mdb contains a number of database wrappers for databases I commonly
// use
package mdb

import (
	"context"

	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"google.golang.org/api/option"
)

// DefaultGCEProject can be set before any of the Cfg* functions are called for
// GCE services, and will be used as the default value for the "project"
// configuration parameter for each service.
var DefaultGCEProject string

// GCE wraps configuration parameters commonly used for interacting with GCE
// services.
type GCE struct {
	Project  string
	CredFile string
}

// CfgGCE configures and returns a GCE instance which will be usable once Run is
// called on the passed in Cfg instance.
func CfgGCE(cfg *mcfg.Cfg) *GCE {
	proj := cfg.ParamString("project", DefaultGCEProject, "name of GCE project")
	credFile := cfg.ParamString("cred-file", "", "path to GCE credientials json file, if any")
	var gce GCE
	cfg.Start.Then(func(ctx context.Context) error {
		gce.Project = *proj
		gce.CredFile = *credFile
		return nil
	})
	return &gce
}

// ClientOptions generates and returns the ClientOption instances which can be
// passed into most GCE client drivers.
func (gce *GCE) ClientOptions() []option.ClientOption {
	var opts []option.ClientOption
	if gce.CredFile != "" {
		opts = append(opts, option.WithCredentialsFile(gce.CredFile))
	}
	return opts
}

// KV implements the mlog.KVer interface
func (gce *GCE) KV() mlog.KV {
	return mlog.KV{
		"project": gce.Project,
	}
}
