package main

/*
	totp-proxy is a reverse proxy which implements basic time-based one-time
	password (totp) authentication for any website.

	It takes in a JSON object which maps usernames to totp secrets (generated at
	a site like https://freeotp.github.io/qrcode.html), as well as a url to
	proxy requests to. Users are prompted with a basic-auth prompt, and if they
	succeed their totp challenge a cookie is set and requests are proxied to the
	destination.
*/

import (
	"net/http"
	"net/url"
	"time"

	"github.com/mediocregopher/mediocre-go-lib/m"
	"github.com/mediocregopher/mediocre-go-lib/mcfg"
	"github.com/mediocregopher/mediocre-go-lib/mcrypto"
	"github.com/mediocregopher/mediocre-go-lib/mctx"
	"github.com/mediocregopher/mediocre-go-lib/mhttp"
	"github.com/mediocregopher/mediocre-go-lib/mlog"
	"github.com/mediocregopher/mediocre-go-lib/mrand"
	"github.com/mediocregopher/mediocre-go-lib/mrun"
	"github.com/mediocregopher/mediocre-go-lib/mtime"
	"github.com/pquerna/otp/totp"
)

func main() {
	ctx := mctx.New()
	logger := mlog.From(ctx)
	cookieName := mcfg.String(ctx, "cookie-name", "_totp_proxy", "String to use as the name for cookies")
	cookieTimeout := mcfg.Duration(ctx, "cookie-timeout", mtime.Duration{1 * time.Hour}, "Timeout for cookies")

	var userSecrets map[string]string
	mcfg.RequiredJSON(ctx, "users", &userSecrets, "JSON object which maps usernames to their TOTP secret strings")

	var secret mcrypto.Secret
	secretStr := mcfg.String(ctx, "secret", "", "String used to sign authentication tokens. If one isn't given a new one will be generated on each startup, invalidating all previous tokens.")
	mrun.OnStart(ctx, func(mctx.Context) error {
		if *secretStr == "" {
			*secretStr = mrand.Hex(32)
		}
		logger.Info("generating secret")
		secret = mcrypto.NewSecret([]byte(*secretStr))
		return nil
	})

	proxyHandler := new(struct{ http.Handler })
	proxyURL := mcfg.RequiredString(ctx, "dst-url", "URL to proxy requests to. Only the scheme and host should be set.")
	mrun.OnStart(ctx, func(mctx.Context) error {
		u, err := url.Parse(*proxyURL)
		if err != nil {
			return err
		}
		proxyHandler.Handler = mhttp.ReverseProxy(u)
		return nil
	})

	authHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO mlog.FromHTTP?
		authLogger := logger.Clone()
		authLogger.SetKV(mlog.CtxKV(r.Context()))

		unauthorized := func() {
			w.Header().Add("WWW-Authenticate", "Basic")
			w.WriteHeader(http.StatusUnauthorized)
		}

		authorized := func() {
			sig := mcrypto.SignString(secret, "")
			http.SetCookie(w, &http.Cookie{
				Name:   *cookieName,
				Value:  sig.String(),
				MaxAge: int((*cookieTimeout).Seconds()),
			})
			proxyHandler.ServeHTTP(w, r)
		}

		if cookie, _ := r.Cookie(*cookieName); cookie != nil {
			authLogger.Debug("authenticating with cookie", mlog.KV{"cookie": cookie.String()})
			var sig mcrypto.Signature
			if err := sig.UnmarshalText([]byte(cookie.Value)); err == nil {
				err := mcrypto.VerifyString(secret, sig, "")
				if err == nil && time.Since(sig.Time()) < (*cookieTimeout).Duration {
					authorized()
					return
				}
			}
		}

		if user, pass, ok := r.BasicAuth(); ok && pass != "" {
			logger.Debug("authenticating with user/pass", mlog.KV{
				"user": user,
				"pass": pass,
			})
			if userSecret, ok := userSecrets[user]; ok {
				if totp.Validate(pass, userSecret) {
					authorized()
					return
				}
			}
		}

		unauthorized()
	})

	mhttp.MListenAndServe(ctx, authHandler)
	m.Run(ctx)
}
