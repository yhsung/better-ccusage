package schema

import (
	"encoding/json"
	"fmt"
	"io"
)

// ConfigSchema is the hand-written JSON Schema for the user config file.
// It mirrors apps/better-ccusage/internal/config.Config.
const ConfigSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "better-ccusage config",
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "pricingUrl": { "type": "string", "format": "uri" },
    "offline": { "type": "boolean" },
    "defaultMode": { "type": "string", "enum": ["auto", "calculate", "display"] },
    "extraConfigDirs": {
      "type": "array",
      "items": { "type": "string" }
    }
  }
}`

// Write outputs the config schema (pretty-printed) to w.
func Write(w io.Writer) error {
	var v any
	if err := json.Unmarshal([]byte(ConfigSchema), &v); err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Validate checks that data is a valid JSON object (basic check; full schema
// validation requires a JSON Schema library which is out of scope for v1).
func Validate(data []byte) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if _, ok := v.(map[string]any); !ok {
		return fmt.Errorf("invalid config: expected a JSON object")
	}
	return nil
}
