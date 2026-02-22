package pipe

// Preset defines a reusable prompt configuration that can be referenced by name.
// Presets are loaded from embedded YAML files in internal/prompts/.
type Preset struct {
	Name        string
	Description string
	System      string
}
