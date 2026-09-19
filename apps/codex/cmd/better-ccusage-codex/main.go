// Command better-ccusage-codex is a deprecated shim forwarding to better-ccusage.
package main

import (
	"os"

	"github.com/cobra91/better-ccusage/pkg/shim"
)

func main() {
	os.Exit(shim.Run(shim.Config{
		Target:  "better-ccusage",
		OwnName: "better-ccusage-codex",
		NoticeLines: []string{
			"[@better-ccusage/codex] This package is deprecated. Codex support is now built into better-ccusage.",
			"[@better-ccusage/codex] Run `npx better-ccusage` directly. Forwarding your invocation now.",
			"[@better-ccusage/codex] To silence this notice set CODEX_NO_DEPRECATION_NOTICE=1.",
		},
		OptOutEnv: "CODEX_NO_DEPRECATION_NOTICE",
		Set:       map[string]string{"OFFLINE": "true"},
		SuppressIfUnset: map[string]string{
			"DROID_SESSIONS_DIR": os.DevNull,
			"ZCODE_HOME":         shim.NonexistentTmp("zcode"),
		},
	}, os.Args[1:]))
}
