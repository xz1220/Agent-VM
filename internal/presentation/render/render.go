// Package render holds shared text/JSON rendering helpers used by the
// presentation layer. Keeping this small and pure makes it trivial for
// CLI commands to stay free of business logic.
package render

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSON writes v as indented JSON to w.
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Linef writes a single formatted line to w with a trailing newline.
func Linef(w io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(w, format+"\n", args...)
	return err
}
