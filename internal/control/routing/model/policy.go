package model

type Filter struct {
	Country    string
	NotCountry string
}

type Policy struct {
	Tag     string
	RuleSet string
	Source  Filter
	Target  Filter
	Active  bool
}
