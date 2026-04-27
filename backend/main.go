package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"listen-ports/internal/api"
	"listen-ports/internal/cache"
	"listen-ports/internal/docker"
	"listen-ports/internal/mapper"
	"listen-ports/internal/parser"
	"listen-ports/ui"
)

// version is set via -ldflags at build time
var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "/health" {
		fmt.Println("ok")
		os.Exit(0)
	}

	log.Printf("Listen Ports %s starting", version)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	procPath := os.Getenv("PROC_PATH")
	if procPath == "" {
		procPath = "/proc"
	}

	// Startup diagnostic: check if process resolution will work
	checkProcessCapabilities(procPath)

	fetchFunc := func() ([]parser.SocketEntry, error) {
		tcp, err := parser.ParseTCP(procPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse TCP: %w", err)
		}
		tcp6, err := parser.ParseTCP6(procPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse TCP6: %w", err)
		}
		udp, err := parser.ParseUDP(procPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse UDP: %w", err)
		}
		udp6, err := parser.ParseUDP6(procPath)
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

	// Docker collector (optional)
	dockerSocket := os.Getenv("DOCKER_HOST")
	var dockerCache *docker.Cache
	collector, err := docker.NewCollector(dockerSocket)
	if err != nil {
		log.Printf("Docker collector initialization failed: %v", err)
	} else if collector != nil {
		dockerCache = docker.NewCache(collector, 10*time.Second)
		log.Println("Docker monitoring enabled")
	} else {
		log.Println("Docker monitoring disabled (DOCKER_HOST not set)")
	}

	h := api.NewHandler(c, procPath, dockerCache, version)

	adminToken := os.Getenv("ADMIN_TOKEN")
	if adminToken == "" {
		log.Println("ADMIN_TOKEN not set — authentication disabled")
	} else {
		log.Println("ADMIN_TOKEN set — authentication enabled")
	}

	mux := http.NewServeMux()
	mux.Handle("/api/auth", api.AuthHandler(adminToken))
	mux.Handle("/api/sockets", api.AuthMiddleware(adminToken)(h))
	mux.Handle("/api/containers", api.AuthMiddleware(adminToken)(api.ContainersHandler(dockerCache)))
	mux.Handle("/", spaHandler())

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      securityHeaders(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
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

func checkProcessCapabilities(procPath string) {
	// Try reading a known PID's fd directory to detect capability issues early
	entries, err := os.ReadDir(procPath)
	if err != nil {
		log.Printf("WARNING: cannot read %s: %v — socket data may work but process resolution will fail", procPath, err)
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}
		fdPath := filepath.Join(procPath, entry.Name(), "fd")
		if _, err := os.ReadDir(fdPath); err != nil {
			log.Printf("WARNING: cannot read %s: %v", fdPath, err)
			log.Printf("WARNING: process names will be empty — add SYS_PTRACE and DAC_READ_SEARCH capabilities")
			return
		}
		// Successfully read at least one PID's fd dir
		_, nameErr := mapper.BuildProcessMap(procPath)
		if nameErr != nil {
			log.Printf("WARNING: process map build test failed: %v", nameErr)
		} else {
			log.Println("Process resolution: OK")
		}
		return
	}
	log.Println("WARNING: no processes found in", procPath)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func spaHandler() http.Handler {
	distFS, err := fs.Sub(ui.DistFS, "dist")
	if err != nil {
		log.Fatalf("Failed to create sub FS: %v", err)
	}
	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path != "/" {
			stripped := strings.TrimPrefix(path, "/")
			if f, err := distFS.Open(stripped); err == nil {
				f.Close()
				if strings.Contains(stripped, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		r.URL.Path = "/"
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		fileServer.ServeHTTP(w, r)
	})
}