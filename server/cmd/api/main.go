package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ManadaHerath/realtime-grid-server/internal/api"
	"github.com/ManadaHerath/realtime-grid-server/internal/grid"
)

func main() {
	store := grid.NewStore()
	apiHandler := api.NewAPI(store)

	mux := http.NewServeMux()
	apiHandler.RegisterRoutes(mux)

	addr := ":8080"
	fmt.Println("Server listening on", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
