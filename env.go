package gatekit

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

// EnvVarName returns the environment-variable name for feature f under the given
// prefix: prefix + the canonical suffix for the feature. The prefix stays the
// consumer's (gatekit is general-purpose and bakes in no consumer-specific
// prefix); gatekit owns only the suffix convention, so every consumer — and every
// language port — produces identical names for the same feature.
//
// The suffix is the feature's CamelCase identifier converted to UPPER_SNAKE_CASE:
// an underscore is inserted before an upper-case rune that follows a lower-case
// rune or a digit, then everything is upper-cased. For example:
//
//	EnvVarName("ConsensusNodeOperatorManaged", "SOLO_PROVISIONER_ENABLE_")
//	  -> "SOLO_PROVISIONER_ENABLE_CONSENSUS_NODE_OPERATOR_MANAGED"
func EnvVarName(f Feature, prefix string) string {
	runes := []rune(string(f))
	var b strings.Builder
	b.Grow(len(prefix) + len(runes) + 8)
	b.WriteString(prefix)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			if prev := runes[i-1]; unicode.IsLower(prev) || unicode.IsDigit(prev) {
				b.WriteByte('_')
			}
		}
		b.WriteRune(unicode.ToUpper(r))
	}
	return b.String()
}

// WithEnvOverrides scans the process environment for variables beginning with
// prefix and applies each as a per-feature override, mapping a variable's name
// back to a feature via EnvVarName(f, prefix). An env override is a real override:
// it beats version/maturity resolution exactly like WithOverrides, and an explicit
// WithOverrides applied after WithEnvOverrides still wins per feature (option order
// is last-writer).
//
// It fails loudly rather than silently ignoring a mistyped opt-in: the Option
// returns an error (so New returns it and the consumer's binary stops) when a
// <prefix>* variable does not map to any registered feature, or its value is not a
// parseable bool (strconv.ParseBool). The error names the offending variable.
//
// Forcing a feature on before its ActiveFrom is not an error — an override
// intentionally bypasses the version gate (the existing escape-hatch semantics);
// only unknown or malformed variables error.
func WithEnvOverrides(prefix string) Option {
	return func(c *config) error {
		// Canonical env-var name -> feature, for every registered feature.
		byName := make(map[string]Feature, len(c.registry))
		for f := range c.registry {
			byName[EnvVarName(f, prefix)] = f
		}
		for _, kv := range os.Environ() {
			name, value, ok := strings.Cut(kv, "=")
			if !ok || !strings.HasPrefix(name, prefix) {
				continue
			}
			f, known := byName[name]
			if !known {
				return fmt.Errorf("gatekit: environment variable %q does not match any known feature", name)
			}
			v, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("gatekit: environment variable %q has an unparseable bool value %q", name, value)
			}
			if c.overrides == nil {
				c.overrides = make(map[Feature]bool)
			}
			c.overrides[f] = v
		}
		return nil
	}
}
