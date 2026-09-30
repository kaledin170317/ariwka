package subscription

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"

	"vpn/internal/control/user/model"
)

type Profiles interface {
	Get(context.Context, string) ([]string, error)
}

type Users interface {
	Get(context.Context, string) (model.User, error)
}

type Handler struct {
	Profiles Profiles
	Users    Users
}

func URL(userID string) (string, error) {
	domain := os.Getenv("SUBSCRIPTION_DOMAIN")
	if domain == "" {
		return "", errors.New("SUBSCRIPTION_DOMAIN is not set")
	}

	return "https://" + domain + "/sub/" + userID, nil
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

	connections, err := h.Profiles.Get(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to load subscription", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Profile-Title", "VPN")
	w.Header().Set("Profile-Update-Interval", "1")

	_, _ = w.Write([]byte(strings.Join(connections, "\n") + "\n"))
}
