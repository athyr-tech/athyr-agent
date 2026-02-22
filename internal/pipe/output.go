package pipe

import "io"

// Formatter controls how LLM responses are written to the output stream.
type Formatter struct {
	Writer io.Writer
}
