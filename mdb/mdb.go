// Package mdb contains a number of database wrappers for databases I commonly
// use
package mdb

import (
	"context"

	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mcmp"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
	"google.golang.org/api/option"
)

// GCE wraps configuration parameters commonly used for interacting with GCE
// services.
type GCE struct {
	cmp      *mcmp.Component
	Project  string
	CredFile string
}

type gceOpts struct {
	defaultProject string
}

// GCEOption is a value which adjusts the behavior of InstGCE.
type GCEOption func(*gceOpts)

// GCEDefaultProject sets the given string to be the default project of the GCE
// instance. The default project will still be configurable via mcfg regardless
// of what this is set to.
func GCEDefaultProject(defaultProject string) GCEOption {
	return func(opts *gceOpts) {
		opts.defaultProject = defaultProject
	}
}

// InstGCE instantiates a GCE which will be initialized when the Init event is
// triggered on the given Component. defaultProject is used as the default value
// for the mcfg parameter this function creates.
func InstGCE(cmp *mcmp.Component, options ...GCEOption) *GCE {
	var opts gceOpts
	for _, opt := range options {
		opt(&opts)
	}

	gce := GCE{cmp: cmp.Child("gce")}
	credFile := mcfg.String(gce.cmp, "cred-file",
		mcfg.ParamUsage("Path to GCE credientials JSON file, if any"))
	project := mcfg.String(gce.cmp, "project",
		mcfg.ParamDefaultOrRequired(opts.defaultProject),
		mcfg.ParamUsage("Name of GCE project to use"))

	mrun.InitHook(gce.cmp, func(ctx context.Context) error {
		gce.Project = *project
		gce.CredFile = *credFile
		gce.cmp.Annotate("project", gce.Project)
		mlog.From(gce.cmp).Info("GCE config initialized", ctx)
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

// Context returns the annotated Context from this instance's initialization.
func (gce *GCE) Context() context.Context {
	return gce.cmp.Context()
}
