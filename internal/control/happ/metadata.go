package happ

import (
	"strconv"
	"strings"
)

type Metadata struct {
	Title            string
	UpdateInterval   int
	WebPageURL       string
	SupportURL       string
	Announcement     string
	PingOnOpen       bool
	AutoUpdateOnOpen bool
	Pin              bool
}

func (m Metadata) Lines() []string {
	lines := []string{
		"#profile-title: " + m.Title,
		"#profile-update-interval: " + strconv.Itoa(m.UpdateInterval),
	}

	if m.WebPageURL != "" {
		lines = append(lines, "#profile-web-page-url: "+m.WebPageURL)
	}

	if m.SupportURL != "" {
		lines = append(lines, "#support-url: "+m.SupportURL)
	}

	if m.Announcement != "" {
		lines = append(lines, "#announce: "+strings.ReplaceAll(m.Announcement, "\n", " "))
	}

	if m.PingOnOpen {
		lines = append(lines, "#subscription-ping-onopen-enabled: 1")
	}

	if m.AutoUpdateOnOpen {
		lines = append(lines, "#subscription-auto-update-open-enable: 1")
	}

	if m.Pin {
		lines = append(lines, "#subscription-pin: 1")
	}

	return lines
}
