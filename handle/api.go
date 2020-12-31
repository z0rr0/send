package handle

import (
	"encoding/json"
	"io"
)

func version(w io.Writer, p *Params) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(p.Version)
}
