package controllers

// An option of the filter and sort selects
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
