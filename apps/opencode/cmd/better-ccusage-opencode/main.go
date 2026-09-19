// Command better-ccusage-opencode is a deprecated shim forwarding to better-ccusage.
package main

import (
	"os"

	"github.com/cobra91/better-ccusage/pkg/shim"
)

func main() {
	os.Exit(shim.Run(shim.Config{
		Target:  "better-ccusage",
		OwnName: "better-ccusage-opencode",
		NoticeLines: []string{
			"[@better-ccusage/opencode] This package is deprecated. OpenCode support is now built into better-ccusage.",
			"[@better-ccusage/opencode] Run `npx better-ccusage` directly. Forwarding your invocation now.",
			"[@better-ccusage/opencode] To silence this notice set OPENCODE_NO_DEPRECATION_NOTICE=1.",
		},
		OptOutEnv: "OPENCODE_NO_DEPRECATION_NOTICE",
		Set:       map[string]string{"OFFLINE": "true"},
		SuppressIfUnset: map[string]string{
			"DROID_SESSIONS_DIR": os.DevNull,
			"ZCODE_HOME":         shim.NonexistentTmp("zcode"),
			"CODEX_HOME":         shim.NonexistentTmp("codex"),
		},
	}, os.Args[1:]))
}
