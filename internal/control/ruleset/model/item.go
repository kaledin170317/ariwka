package model

//go:generate go tool go-enum --marshal --output-suffix=.enum.gen

// ENUM(source=1, binary)
type Format int32

const UpdateIntervalSeconds uint32 = 24 * 60 * 60

type Item struct {
	Tag    string
	URL    string
	Format Format
	Active bool
}
