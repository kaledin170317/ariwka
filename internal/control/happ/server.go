package happ

import (
	"log"
	"net/http"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

func Serve(domain string, profiles Profiles, users Users, stats Stats) error {
	mux := http.NewServeMux()
	mux.Handle("GET /happ/sub/{userID}", Handler{Profiles: profiles, Users: users, Stats: stats, Metadata: subMetadata})
	mux.Handle("GET /happ/default/{userID}", Handler{Profiles: profiles, Users: users, Stats: stats, Default: true, Metadata: defaultMetadata})
	mux.Handle("GET /happ/device/{deviceID}", DeviceHandler{Profiles: profiles, Stats: stats, Metadata: subMetadata})

	manager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domain),
		Cache:      autocert.DirCache("/var/lib/control/certs"),
	}

	go func() {
		log.Fatal(http.ListenAndServe(":80", manager.HTTPHandler(nil)))
	}()

	server := &http.Server{
		Addr:              ":443",
		Handler:           mux,
		TLSConfig:         manager.TLSConfig(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	return server.ListenAndServeTLS("", "")
}
