package server

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humahttprouter"
	"github.com/johnmantios/micromanager/internal/handler"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func Routes() http.Handler {
	router := httprouter.New()
	cfg := huma.DefaultConfig("Micromanager API", "1.0.0")
	cfg.DocsRenderer = huma.DocsRendererSwaggerUI
	api := humahttprouter.New(router, cfg)

	huma.Register(api, huma.Operation{
		OperationID:   "create-screentime",
		Method:        http.MethodPost,
		Path:          "/v1/screentime",
		Summary:       "Create screentime entry",
		DefaultStatus: http.StatusCreated,
	}, handler.CreateScreentime)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("micromanager"))
	})

	return router
}
