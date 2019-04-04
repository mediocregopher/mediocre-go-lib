package mcfg

import (
	"context"
	"os"
	"strings"

	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/merr"
)

// SourceEnv is a Source which will parse configuration from the process
// environment.
//
// Possible Env options are generated by joining a Param's Path and Name with
// underscores and making all characters uppercase, as well as changing all
// dashes to underscores.
//
//	ctx := mctx.New()
//	ctx = mctx.ChildOf(ctx, "foo")
//	ctx = mctx.ChildOf(ctx, "bar")
//	addr := mcfg.String(ctx, "srv-addr", "", "Some address")
//	// the Env option to fill addr will be "FOO_BAR_SRV_ADDR"
//
type SourceEnv struct {
	// In the format key=value. Defaults to os.Environ() if nil.
	Env []string

	// If set then all expected Env options must be prefixed with this string,
	// which will be uppercased and have dashes replaced with underscores like
	// all the other parts of the option names.
	Prefix string
}

func (env *SourceEnv) expectedName(path []string, name string) string {
	out := strings.Join(append(path, name), "_")
	if env.Prefix != "" {
		out = env.Prefix + "_" + out
	}
	out = strings.Replace(out, "-", "_", -1)
	out = strings.ToUpper(out)
	return out
}

// Parse implements the method for the Source interface
func (env *SourceEnv) Parse(ctx context.Context, params []Param) (context.Context, []ParamValue, error) {
	kvs := env.Env
	if kvs == nil {
		kvs = os.Environ()
	}

	pM := map[string]Param{}
	for _, p := range params {
		name := env.expectedName(mctx.Path(p.Context), p.Name)
		pM[name] = p
	}

	pvs := make([]ParamValue, 0, len(kvs))
	for _, kv := range kvs {
		split := strings.SplitN(kv, "=", 2)
		if len(split) != 2 {
			ctx := mctx.Annotate(context.Background(), "kv", kv)
			return nil, nil, merr.New("malformed environment key/value pair", ctx)
		}
		k, v := split[0], split[1]
		if p, ok := pM[k]; ok {
			pvs = append(pvs, ParamValue{
				Name:  p.Name,
				Path:  mctx.Path(p.Context),
				Value: p.fuzzyParse(v),
			})
		}
	}

	return ctx, pvs, nil
}
