package country

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Service struct{}

func (Service) Code(ctx context.Context, host string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://ip-api.com/json/"+url.PathEscape(host), nil)
	if err != nil {
		return "", err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("determine country: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("determine country: HTTP %d", response.StatusCode)
	}

	var location struct {
		Status      string `json:"status"`
		CountryCode string `json:"countryCode"`
	}

	err = json.NewDecoder(response.Body).Decode(&location)
	if err != nil {
		return "", fmt.Errorf("determine country: %w", err)
	}

	if location.Status != "success" || location.CountryCode == "" {
		return "", errors.New("could not determine country")
	}

	return location.CountryCode, nil
}
