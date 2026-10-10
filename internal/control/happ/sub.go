package happ

var subMetadata = Metadata{
	Title:            "💤💤💤",
	Announcement:     "Должно работать все но медленно",
	UpdateInterval:   1,
	SupportURL:       "https://t.me/kydimik",
	PingOnOpen:       true,
	AutoUpdateOnOpen: true,
	Pin:              true,
}

func URL(userID string) (string, error) {
	return subscriptionURL(userID, "sub")
}
