package handle

import (
	"encoding/json"
	"io"
)

// version is API handler for version info.
func version(w io.Writer, p *Params) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(p.Version)
}
