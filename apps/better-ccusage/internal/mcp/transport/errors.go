// Package transport owns MCP transport wiring and the sentinel-to-MCP
// error map for the MCP server.
package transport

import (
	"encoding/json"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cobra91/better-ccusage/apps/better-ccusage/internal/errs"
)

// ErrorPayload is the structured MCP tool-error body.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

// MapError maps a sentinel error to an (MCP code, message) pair.
func MapError(err error) (code, message string) {
	switch {
	case errors.Is(err, errs.ErrNoData):
		return "NO_DATA", err.Error()
	case errors.Is(err, errs.ErrUnknownModel):
		return "UNKNOWN_MODEL", err.Error()
	case errors.Is(err, errs.ErrInvalidMode), errors.Is(err, errs.ErrInvalidArgs), errors.Is(err, errs.ErrIncompatibleMode):
		return "INVALID_ARGS", err.Error()
	default:
		return "INTERNAL", err.Error()
	}
}

// HintFor returns the remediation hint for an MCP error code.
func HintFor(code string) string {
	switch code {
	case "NO_DATA":
		return "no usage entries in range; check --since/--until or data directories"
	case "UNKNOWN_MODEL":
		return "model missing from pricing table; check embedded pricing version"
	case "INVALID_ARGS":
		return "check mode (auto|calculate|display) and RFC3339 since/until"
	default:
		return "unexpected error; retry or inspect server logs"
	}
}

// ErrorResult builds an isError MCP result with a structured payload.
// It returns a nil error so the LLM client sees the failure as tool
// output (per CallToolResult docs) instead of a protocol error.
func ErrorResult(code, message, hint string) (*mcp.CallToolResult, any, error) {
	body, _ := json.Marshal(ErrorPayload{Code: code, Message: message, Hint: hint})
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(body)}},
		IsError: true,
	}, nil, nil
}

// ToolError maps err via MapError and returns it as an isError result.
// Pass the original err so errors.Is chains survive.
func ToolError(err error) (*mcp.CallToolResult, any, error) {
	code, message := MapError(err)
	return ErrorResult(code, message, HintFor(code))
}
