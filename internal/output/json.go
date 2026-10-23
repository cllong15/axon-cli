package output

import (
	"axon-cli/internal/checker"
	"encoding/json"
	"io"
)

func FormatJSON(w io.Writer, results []checker.CheckResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}
