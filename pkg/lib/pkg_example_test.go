package lib_test

import (
	"fmt"
	"io"
	"net/http"

	sloth "github.com/slok/sloth/pkg/lib"
)

// This example shows a basic usage of sloth library by exposing sloth SLO generation functionality as a rest HTTP API.
func Example() {
	// Check with `curl -XPOST http://127.0.0.1:8080/sloth/generate -d "$(cat ./examples/getting-started.yml)"`.

	gen, err := sloth.NewPrometheusSLOGenerator(sloth.PrometheusSLOGeneratorConfig{
		ExtraLabels: map[string]string{"source": "slothlib-example"},
	})
	if err != nil {
		panic(fmt.Errorf("could not create SLO generator: %w", err))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /sloth/generate", func(w http.ResponseWriter, r *http.Request) {
		// Get body from request.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Generate SLOs.
		result, err := gen.GenerateFromRaw(r.Context(), body)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not generate SLOs: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		err = sloth.WriteResultAsPrometheusStd(r.Context(), *result, w)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not write result: %v", err), http.StatusInternalServerError)
			return
		}
	})

	httpServer := &http.Server{Addr: ":8080", Handler: mux}

	fmt.Println("Starting server at :8080")

	err = httpServer.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
