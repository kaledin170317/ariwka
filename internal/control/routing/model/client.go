package model

// ClientRouting is a compact client rule set. Happ downloads the referenced
// Xray geo files; Rules contain category names rather than expanded entries.
type ClientRouting struct {
	Rules      []map[string]any
	GeoIPURL   string
	GeoSiteURL string
}
