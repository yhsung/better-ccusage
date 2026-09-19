package pricing

import _ "embed"

// EmbeddedPrices contains the bundled model_prices_and_context_window.json.
// Source: litellm/model_prices_and_context_window.json (mirrors TS).
//
//go:embed testdata/prices.json
var EmbeddedPrices []byte
