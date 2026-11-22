package model

type Filter struct {
	Country    string
	NotCountry string
}

type Policy struct {
	Tag            string
	RuleFile       string
	RuleCode       string
	Source         Filter
	Target         Filter
	Active         bool
	ApplyToNodes   bool
	ApplyToClients bool
}
