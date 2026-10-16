package data

import (
	"fmt"
	"html/template"
	"regexp"
	"sort"
	"strings"

	"transcendance/models"
)

// Names of the special effects that may be stored on an acquired card copy.
// They map to the `.card-<name>` classes implemented in
// `frontend/src/CardEffectStyles.svelte`.
//
// Glare is the standard treatment of every card and lives in
// `templates/partials/card.html`, so it is not a stored special effect.
var validEffects = map[string]bool{
	"foil":      true,
	"rainbow":   true,
	"sparkle":   true,
	"watermark": true,
}

// Parameter keys understood by `frontend/src/CardRender.svelte`, which maps
// them to `--card-effect-*` CSS properties. Anything else is refused on write
// and dropped on render.
var validEffectParams = map[string]bool{
	"z":         true,
	"opacity":   true,
	"rest":      true,
	"transform": true,
	"scale":     true,
	"angle":     true,
	"hue":       true,
	"color":     true,
	"band":      true,
	"speed":     true,
	"blend":     true,
	"spectrum":  true,
	"mask":      true,
	"watermark": true,
}

const maxEffectValueLen = 256

// Values are rendered into `data-*` attributes, so only this character set is
// ever let through: no quotes, angle brackets, ampersands or semicolons.
var effectValuePattern = regexp.MustCompile(`^[A-Za-z0-9#.,%():*/+\- ]*$`)

// Default parameters for each effect, used when a copy doesn't override them.
var effectPresets = map[string]map[string]string{
	"foil": {
		"z":       "11",
		"opacity": ".8",
		"scale":   "0.5",
		"color":   "#ffaf5f",
		"mask":    "/cards/assets/foil6.png",
	},
	"rainbow": {
		"z":       "11",
		"opacity": "0.45",
		"scale":   "0.5",
		"mask":    "/cards/assets/foil6.png",
	},
	"sparkle": {
		"z":       "11",
		"opacity": "0.8",
		"speed":   "1",
	},
	"watermark": {
		"z":         "11",
		"opacity":   "0.2",
		"watermark": "/cards/assets/watermark_42.png",
	},
}

// effectLayer is one `.card-effect` layer to render inside a card.
type effectLayer struct {
	Class string
	Attrs template.HTMLAttr
}

// ValidateEffects reports whether every effect name and parameter is known and
// safe to store. Unknown entries are refused so that a poisoned row can never
// make it into the database through a write path.
func ValidateEffects(effects models.CardEffects) error {
	for _, effect := range effects {
		if !validEffects[effect.Name] {
			return fmt.Errorf("unknown effect '%s'", effect.Name)
		}
		for key, value := range effect.Params {
			if !validEffectParams[key] {
				return fmt.Errorf("unknown parameter '%s' for effect '%s'", key, effect.Name)
			}
			if !isSafeEffectValue(value) {
				return fmt.Errorf("invalid value for parameter '%s' of effect '%s'", key, effect.Name)
			}
		}
	}
	return nil
}

// ParseEffects parses a comma-separated list of effect names, e.g. `foil,sparkle`.
// Parameters are left empty and filled in from the effect presets at render time.
func ParseEffects(spec string) (models.CardEffects, error) {
	var effects models.CardEffects
	for _, name := range strings.Split(spec, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if !validEffects[name] {
			return nil, fmt.Errorf("unknown effect '%s'", name)
		}
		effects = append(effects, models.CardEffect{Name: name})
	}
	if err := ValidateEffects(effects); err != nil {
		return nil, err
	}
	return effects, nil
}

func isSafeEffectValue(value string) bool {
	return len(value) <= maxEffectValueLen && effectValuePattern.MatchString(value)
}

// renderEffect builds one `.card-effect` layer from an effect name and its
// parameter overrides. Presets fill in anything the copy doesn't set.
// Unknown names, parameters and unsafe values are dropped: rendering must stay
// safe even on rows written before validation existed.
func renderEffect(name string, params map[string]string) (effectLayer, bool) {
	if !validEffects[name] {
		return effectLayer{}, false
	}

	merged := make(map[string]string, len(effectPresets[name])+len(params))
	for key, value := range effectPresets[name] {
		merged[key] = value
	}
	for key, value := range params {
		merged[key] = value
	}

	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var attrs strings.Builder
	for _, key := range keys {
		value := merged[key]
		if !validEffectParams[key] || !isSafeEffectValue(value) {
			continue
		}
		fmt.Fprintf(&attrs, ` data-%s="%s"`, key, value)
	}

	return effectLayer{
		Class: "card-" + name,
		Attrs: template.HTMLAttr(attrs.String()),
	}, true
}

// InventoryCardEffects builds the extra effect layers for an acquired copy:
// only the special effects that copy actually has. The base glare is rendered
// by the card template for every card and is not repeated here.
func InventoryCardEffects(effects models.CardEffects) []effectLayer {
	layers := make([]effectLayer, 0, len(effects))
	for _, effect := range effects {
		if layer, ok := renderEffect(effect.Name, effect.Params); ok {
			layers = append(layers, layer)
		}
	}
	return layers
}
