package schema

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestConfigSchema_ParsesAsJSON(t *testing.T) {
	var v map[string]any
	if err := json.Unmarshal([]byte(ConfigSchema), &v); err != nil {
		t.Fatalf("ConfigSchema does not parse as JSON: %v", err)
	}
	if v["type"] != "object" {
		t.Errorf("schema type: got %v, want object", v["type"])
	}
	if v["additionalProperties"] != false {
		t.Errorf("schema additionalProperties: got %v, want false", v["additionalProperties"])
	}
	props, ok := v["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema missing properties object")
	}
	for _, field := range []string{"pricingUrl", "offline", "defaultMode", "extraConfigDirs"} {
		if _, ok := props[field]; !ok {
			t.Errorf("schema missing property %q", field)
		}
	}
}

func TestConfigSchema_DefaultModeEnum(t *testing.T) {
	var v struct {
		Properties struct {
			DefaultMode struct {
				Enum []string `json:"enum"`
			} `json:"defaultMode"`
		} `json:"properties"`
	}
	if err := json.Unmarshal([]byte(ConfigSchema), &v); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"auto": true, "calculate": true, "display": true}
	if len(v.Properties.DefaultMode.Enum) != len(want) {
		t.Fatalf("defaultMode enum: got %v, want [auto calculate display]", v.Properties.DefaultMode.Enum)
	}
	for _, m := range v.Properties.DefaultMode.Enum {
		if !want[m] {
			t.Errorf("defaultMode enum: unexpected value %q", m)
		}
	}
}

func TestValidate_ValidObject(t *testing.T) {
	if err := Validate([]byte(`{"offline":true,"defaultMode":"calculate"}`)); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestValidate_InvalidJSON(t *testing.T) {
	if err := Validate([]byte(`{invalid`)); err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
}

func TestValidate_NonObject(t *testing.T) {
	if err := Validate([]byte(`[1,2,3]`)); err == nil {
		t.Error("expected error for JSON array, got nil")
	}
}

func TestWrite_ProducesJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf); err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Fatal("Write produced no output")
	}
	var got, want any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Write output is not valid JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(ConfigSchema), &want); err != nil {
		t.Fatal(err)
	}
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	if !bytes.Equal(gotJSON, wantJSON) {
		t.Error("Write output differs semantically from ConfigSchema")
	}
}
