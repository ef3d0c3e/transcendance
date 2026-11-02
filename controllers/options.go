package controllers

import "strconv"

// Highest page number accepted, beyond it offsets would overflow
const maxPage = 100000

// PageParam is a page number from a query value: 1 when missing or invalid, at most maxPage
func PageParam(raw string) int {
	page, err := strconv.Atoi(raw)
	if err != nil || page < 1 {
		return 1
	}
	return min(page, maxPage)
}

type SelectOption struct {
	Value    string
	Label    string
	Selected bool
}

func SelectOptions(values []string, current string, label func(value string) string) []SelectOption {
	options := make([]SelectOption, len(values))
	for i, value := range values {
		options[i] = SelectOption{Value: value, Label: label(value), Selected: value == current}
	}
	return options
}
