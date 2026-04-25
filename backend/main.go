package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"listen-ports/internal/api"
	"listen-ports/internal/cache"
	"listen-ports/internal/parser"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fetchFunc := func() ([]parser.SocketEntry, error) {
		tcp, err := parser.ParseTCP()
		if err != nil {
			return nil, fmt.Errorf("failed to parse TCP: %w", err)
		}
		tcp6, err := parser.ParseTCP6()
		if err != nil {
			return nil, fmt.Errorf("failed to parse TCP6: %w", err)
		}
		udp, err := parser.ParseUDP()
		if err != nil {
			return nil, fmt.Errorf("failed to parse UDP: %w", err)
		}
		udp6, err := parser.ParseUDP6()
		if err != nil {
			return nil, fmt.Errorf("failed to parse UDP6: %w", err)
		}

		merged := make([]parser.SocketEntry, 0, len(tcp)+len(tcp6)+len(udp)+len(udp6))
		merged = append(merged, tcp...)
		merged = append(merged, tcp6...)
		merged = append(merged, udp...)
		merged = append(merged, udp6...)

		return merged, nil
	}

	c := cache.NewCache(fetchFunc)
	h := api.NewHandler(c)

	mux := http.NewServeMux()
	mux.Handle("/api/sockets", corsMiddleware(h))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		log.Printf("Server listening on :%s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced shutdown: %v\n", err)
	}
	log.Println("Server stopped")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}