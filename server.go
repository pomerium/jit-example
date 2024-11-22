package main

import (
	"context"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"slices"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/pomerium/sdk-go"
)

func serve(ctx context.Context) error {
	addr := net.JoinHostPort("0.0.0.0", config.port)
	log.Info().Str("addr", addr).Msg("starting http server")

	mux := http.NewServeMux()
	mux.HandleFunc("POST /admin/approve-access", handleAdminApproveAccess)
	mux.HandleFunc("POST /admin/revoke-access", handleAdminRevokeAccess)
	mux.HandleFunc("GET /admin", handleAdmin)

	mux.HandleFunc("POST /request-access", handleRequestAccess)
	mux.HandleFunc("GET /", handleIndex)

	authMiddleware, err := newAuthMiddleware(config.jwksEndpoint)
	if err != nil {
		return fmt.Errorf("error creating auth middleware: %w", err)
	}

	loggingMiddleware := newLoggingMiddleware()

	h := authMiddleware(loggingMiddleware(mux))
	srv := http.Server{
		Addr:    addr,
		Handler: h,
	}
	context.AfterFunc(ctx, func() {
		shutdownContext, clearTimeout := context.WithTimeout(context.Background(), time.Second*5)
		defer clearTimeout()
		_ = srv.Shutdown(shutdownContext)
	})
	return srv.ListenAndServe()
}

func newLoggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Info().Str("method", r.Method).Str("path", r.URL.Path).Msg("http request")
			next.ServeHTTP(w, r)
		})
	}
}

func newAuthMiddleware(jwksEndpoint string) (func(http.Handler) http.Handler, error) {
	verifier, err := sdk.New(&sdk.Options{
		JWKSEndpoint: jwksEndpoint,
		Logger:       stdlog.New(log.Logger, "", 0),
	})
	if err != nil {
		return nil, err
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawJWT := sdk.TokenFromHeader(r)
			if rawJWT == "" {
				http.Error(w, "missing attestation header", http.StatusForbidden)
				return
			}

			id, err := verifier.GetIdentity(r.Context(), rawJWT)
			if err != nil {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}

			r = r.WithContext(sdk.NewContext(r.Context(), id, nil))
			next.ServeHTTP(w, r)
		})
	}, nil
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	render(w, r, "Index", nil)
}

func handleAdmin(w http.ResponseWriter, r *http.Request) {
	policy, err := getPolicy(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jitUsers := fromPPL(policy.Ppl)
	render(w, r, "Admin", map[string]any{
		"jitUsers": jitUsers,
	})
}

func handleAdminApproveAccess(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	if email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}

	log.Info().Str("email", email).Msg("approve access")

	policy, err := getPolicy(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jitUsers := slices.DeleteFunc(fromPPL(policy.Ppl), func(jitUser JITUser) bool {
		return jitUser.Email == email
	})
	jitUsers = append(jitUsers, JITUser{
		Email:   email,
		Expires: time.Now().UTC().Add(time.Hour),
	})

	policy.Ppl = toPPL(jitUsers)
	err = updatePolicy(r.Context(), policy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func handleAdminRevokeAccess(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	if email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}

	log.Info().Str("email", email).Msg("revoke access")

	policy, err := getPolicy(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jitUsers := slices.DeleteFunc(fromPPL(policy.Ppl), func(jitUser JITUser) bool {
		return jitUser.Email == email
	})

	policy.Ppl = toPPL(jitUsers)
	err = updatePolicy(r.Context(), policy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func handleRequestAccess(w http.ResponseWriter, r *http.Request) {
	user, err := sdk.FromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Info().Str("email", user.Email).Msg("request-access")

	policy, err := getPolicy(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jitUsers := slices.DeleteFunc(fromPPL(policy.Ppl), func(jitUser JITUser) bool {
		return jitUser.Email == user.Email
	})
	jitUsers = append(jitUsers, JITUser{
		Email: user.Email,
	})

	policy.Ppl = toPPL(jitUsers)
	err = updatePolicy(r.Context(), policy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
