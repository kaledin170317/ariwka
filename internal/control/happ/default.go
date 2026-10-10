package happ

var defaultMetadata = Metadata{
	Title:            "💤💤💤",
	Announcement:     "Прямое подключение через выбранную ноду",
	UpdateInterval:   1,
	SupportURL:       "https://t.me/kydimik",
	PingOnOpen:       true,
	AutoUpdateOnOpen: true,
	Pin:              true,
}

func DefaultURL(userID string) (string, error) {
	return subscriptionURL(userID, "default")
}
