package output

import (
	"encoding/json"
	"io"
)

// EncodeJSON writes v as indented JSON to w. Matches the TS JSON output
// shape (2-space indent, sorted keys, trailing newline).
func EncodeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return err
	}
	return nil
}