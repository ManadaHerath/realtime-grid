package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/ManadaHerath/realtime-grid-server/internal/api"
	"github.com/ManadaHerath/realtime-grid-server/internal/grid"
)

func main() {
	redisAddr := getenv("REDIS_ADDR", "localhost:6379")
	redisPass := getenv("REDIS_PASSWORD", "")
	redisDB := 0

	store := grid.NewRedisStore(redisAddr, redisPass, redisDB)

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPass,
		DB:       redisDB,
	})

	apiHandler := api.NewAPI(store, rdb)

	mux := http.NewServeMux()
	apiHandler.RegisterRoutes(mux)

	addr := ":8080"
	fmt.Println("Using Redis at", redisAddr)
	fmt.Println("Server listening on", addr)
	if err := http.ListenAndServe(addr, corsMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
