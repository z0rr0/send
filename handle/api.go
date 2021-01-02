package handle

import (
	"context"
	"encoding/json"
	"net/http"
)

// version is API handler for version info.
func version(_ context.Context, w http.ResponseWriter, p *Params) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(p.Version)
}
