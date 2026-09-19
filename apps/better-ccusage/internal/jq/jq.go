package jq

import (
	"encoding/json"
	"fmt"

	"github.com/itchyny/gojq"
)

// Process runs the given jq expression on input bytes and returns the result.
func Process(expression string, input []byte) ([]byte, error) {
	q, err := gojq.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("parsing jq expression: %w", err)
	}
	var value any
	if err := json.Unmarshal(input, &value); err != nil {
		return nil, fmt.Errorf("unmarshaling input: %w", err)
	}
	iter := q.Run(value)
	v, ok := iter.Next()
	if !ok {
		return nil, nil
	}
	if err, ok := v.(error); ok {
		return nil, err
	}
	return gojq.Marshal(v)
}
