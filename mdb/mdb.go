// Package mdb contains a number of database wrappers for databases I commonly
// use
package mdb

import (
	"context"

	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
	"google.golang.org/api/option"
)

// GCE wraps configuration parameters commonly used for interacting with GCE
// services.
type GCE struct {
	ctx      context.Context
	Project  string
	CredFile string
}

// WithGCE returns a GCE instance which will be initialized and configured when
// the start event is triggered on the returned Context (see mrun.Start).
// defaultProject is used as the default value for the mcfg parameter this
// function creates.
func WithGCE(parent context.Context, defaultProject string) (context.Context, *GCE) {
	ctx := mctx.NewChild(parent, "gce")
	ctx, credFile := mcfg.WithString(ctx, "cred-file", "", "Path to GCE credientials JSON file, if any")

	var project *string
	const projectUsage = "Name of GCE project to use"
	if defaultProject == "" {
		ctx, project = mcfg.WithRequiredString(ctx, "project", projectUsage)
	} else {
		ctx, project = mcfg.WithString(ctx, "project", defaultProject, projectUsage)
	}

	var gce GCE
	ctx = mrun.WithStartHook(ctx, func(context.Context) error {
		gce.Project = *project
		gce.CredFile = *credFile
		gce.ctx = mctx.Annotate(ctx, "project", gce.Project)
		return nil
	})

	gce.ctx = ctx
	return mctx.WithChild(parent, ctx), &gce
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
	return gce.ctx
}
