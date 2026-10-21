package happ

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"

	statsmodel "vpn/internal/control/stats/model"
	"vpn/internal/control/user/model"
)

type Profiles interface {
	GetSubscription(context.Context, string, bool) ([]string, error)
	DeviceSubscription(context.Context, string, string) ([]string, bool, error)
}

type Users interface {
	Get(context.Context, string) (model.User, error)
}

type Stats interface {
	Device(context.Context, string) (statsmodel.Usage, error)
	User(context.Context, string, bool) (statsmodel.Usage, error)
}

type Handler struct {
	Profiles Profiles
	Users    Users
	Stats    Stats
	Metadata Metadata
	Default  bool
}

func DeviceURL(id string) (string, error) {
	return subscriptionURL(id, "device")
}

func DeviceJSONURL(id string) (string, error) {
	return subscriptionURL(id, "device-json")
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
	Stats    Stats
	Metadata Metadata
	JSON     bool
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

	connections, defaultNetwork, err := h.Profiles.DeviceSubscription(r.Context(), id, hwids[0])
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}

	if err != nil {
		http.Error(w, "failed to load subscription", http.StatusBadGateway)
		return
	}

	usage, err := h.Stats.Device(r.Context(), id)
	if err != nil {
		http.Error(w, "failed to load traffic", http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-HWID-Active", "true")
	w.Header().Set("Subscription-Always-HWID-Enable", "1")
	if h.JSON {
		if defaultNetwork {
			http.Error(w, "failover JSON requires a routed device", http.StatusUnprocessableEntity)
			return
		}
		body, err := xraySubscription(connections)
		if err != nil {
			http.Error(w, "failed to build failover config", http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(append(body, '\n'))
		return
	}

	metadata := h.Metadata
	if defaultNetwork {
		metadata = defaultMetadata
	}

	writeSubscription(w, metadata, connections, usage)
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

	usage, err := h.Stats.User(r.Context(), userID, h.Default)
	if err != nil {
		http.Error(w, "failed to load traffic", http.StatusInternalServerError)
		return
	}

	writeSubscription(w, h.Metadata, connections, usage)
}

func writeSubscription(w http.ResponseWriter, metadata Metadata, connections []string, usage statsmodel.Usage) {
	info := fmt.Sprintf("upload=%d; download=%d; total=0", usage.Uploaded, usage.Downloaded)
	w.Header().Set("Subscription-Userinfo", info)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	lines := append([]string{dnsRoutingLink}, metadata.Lines()...)
	lines = append(lines, "#subscription-userinfo: "+info)
	lines = append(lines, connections...)
	_, _ = w.Write([]byte(strings.Join(lines, "\n") + "\n"))
}
