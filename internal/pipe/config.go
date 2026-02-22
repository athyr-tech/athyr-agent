package pipe

// Config holds runtime configuration for a pipe invocation.
// Configuration is resolved via flag > env > config file > default inheritance.
type Config struct {
	Server   string
	Model    string
	Preset   string
	Verbose  bool
	Insecure bool
}
