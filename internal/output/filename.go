// Package output prepares the output directory and decides file names.
package output

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var unsafeChars = regexp.MustCompile(`[^a-z0-9._-]`)

// Windows rejects these as base names regardless of extension.
var windowsReserved = map[string]bool{
	"con": true, "prn": true, "aux": true, "nul": true,
	"com0": true, "com1": true, "com2": true, "com3": true, "com4": true, "com5": true,
	"com6": true, "com7": true, "com8": true, "com9": true,
	"lpt0": true, "lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true, "lpt5": true,
	"lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
}

const maxBaseLen = 100

// FileName returns a file-system safe base name (without extension) for a
// table name.
//
// Names that are already safe are kept as-is for readable URLs. Any other name
// gets a 32-bit hash of the original name appended, so that names which differ
// only by case or by unsafe characters (e.g. "Users" and "users") practically
// never collide, even on case-insensitive file systems. The result depends only
// on the given name. A collision is still theoretically possible (hash collision,
// or a real table literally named like "users-1a2b3c4d"), so callers that write
// many files should detect duplicates.
func FileName(name string) string {
	base := unsafeChars.ReplaceAllString(strings.ToLower(name), "_")
	// A leading dot would create a hidden file.
	if strings.HasPrefix(base, ".") {
		base = "_" + base[1:]
	}
	if len(base) > maxBaseLen {
		base = base[:maxBaseLen]
	}

	if base != "" && base == name && !isWindowsReserved(base) {
		return base
	}

	if base == "" {
		base = "_"
	}
	sum := sha256.Sum256([]byte(name))
	return base + "-" + hex.EncodeToString(sum[:4])
}

func isWindowsReserved(base string) bool {
	stem, _, _ := strings.Cut(base, ".")
	return windowsReserved[stem]
}
