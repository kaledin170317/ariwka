package happ

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"vpn/internal/control/user/model"
)

type Profiles interface {
	GetSubscription(context.Context, string, bool) ([]string, error)
	DeviceSubscription(context.Context, string, string) ([]string, error)
}

type Users interface {
	Get(context.Context, string) (model.User, error)
}

type Handler struct {
	Profiles Profiles
	Users    Users
	Metadata Metadata
	Default  bool
}

func DeviceURL(id string) (string, error) {
	return subscriptionURL(id, "device")
}

func subscriptionURL(userID, network string) (string, error) {
	domain := os.Getenv("SUBSCRIPTION_DOMAIN")
	if domain == "" {
		return "", errors.New("SUBSCRIPTION_DOMAIN is not set")
	}

	return "https://" + domain + "/happ/" + network + "/" + userID, nil
}

type DeviceHandler struct {
	Profiles Profiles
	Metadata Metadata
}

var hwidPattern = regexp.MustCompile(`^[a-zA-Z0-9=-]{10,64}$`)

func (h DeviceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("deviceID")

	parsed, err := uuid.Parse(id)
	if err != nil || parsed.String() != id {
		http.NotFound(w, r)
		return
	}

	hwids := r.Header.Values("X-HWID")
	if len(hwids) != 1 || !hwidPattern.MatchString(hwids[0]) {
		w.Header().Set("X-HWID-Not-Supported", "true")
		http.Error(w, "device ID required", http.StatusForbidden)

		return
	}

	connections, err := h.Profiles.DeviceSubscription(r.Context(), id, hwids[0])
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}

	if err != nil {
		http.Error(w, "failed to load subscription", http.StatusBadGateway)
		return
	}

	w.Header().Set("X-HWID-Active", "true")
	w.Header().Set("Subscription-Always-HWID-Enable", "1")
	writeSubscription(w, h.Metadata, connections)
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")

	id, err := uuid.Parse(userID)
	if err != nil || id.String() != userID {
		http.NotFound(w, r)
		return
	}

	_, err = h.Users.Get(r.Context(), userID)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}

	if err != nil {
		http.Error(w, "failed to load subscription", http.StatusInternalServerError)
		return
	}

	connections, err := h.Profiles.GetSubscription(r.Context(), userID, h.Default)
	if err != nil {
		http.Error(w, "failed to load subscription", http.StatusBadGateway)
		return
	}

	writeSubscription(w, h.Metadata, connections)
}

func writeSubscription(w http.ResponseWriter, metadata Metadata, connections []string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	lines := append(metadata.Lines(), connections...)
	_, _ = w.Write([]byte(strings.Join(lines, "\n") + "\n"))
}
