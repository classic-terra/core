package types

import (
	"regexp"
	"strings"
)

var IBCRegexp = regexp.MustCompile("^ibc/[a-fA-F0-9]{64}$")

// IsIBCDenom returns true if the denom matches the IBC hash format ibc/<64-hex>
func IsIBCDenom(denom string) bool {
	return IBCRegexp.MatchString(strings.ToLower(denom))
}
