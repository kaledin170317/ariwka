package swagger

import (
	"net/http"

	"github.com/swaggest/swgui/v5emb"
)

func Register(
	mux *http.ServeMux,
	spec []byte,
) {
	const (
		jsonPath = "/swagger.json"
		uiPath   = "/swagger/"
	)

	mux.HandleFunc(jsonPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(spec)
	})

	mux.Handle(
		uiPath,
		v5emb.New(
			"API",
			jsonPath,
			uiPath,
		),
	)
}
