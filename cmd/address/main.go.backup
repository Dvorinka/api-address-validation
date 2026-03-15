package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"apiservices/address-validation/internal/address/api"
	"apiservices/address-validation/internal/address/auth"
	"apiservices/address-validation/internal/address/geo"
)

func main() {
	logger := log.New(os.Stdout, "[address] ", log.LstdFlags)

	port := envString("PORT", "8083")
	apiKey := envString("ADDRESS_API_KEY", "dev-address-key")
	providerURL := envString("ADDRESS_PROVIDER_URL", "")
	userAgent := envString("ADDRESS_USER_AGENT", "")
	defaultRegion := envString("ADDRESS_DEFAULT_REGION", "")
	cacheSeconds := envInt("ADDRESS_CACHE_SECONDS", 300)

	if apiKey == "dev-address-key" {
		logger.Println("ADDRESS_API_KEY not set, using default development key")
	}

	provider := geo.NewNominatimProvider(providerURL, userAgent, 10*time.Second)
	service := geo.NewService(provider, defaultRegion, time.Duration(cacheSeconds)*time.Second)
	handler := api.NewHandler(service)

	mux := http.NewServeMux()
	mux.Handle("/v1/address/", auth.Middleware(apiKey)(handler))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadTimeout:       20 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Printf("service listening on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("server failed: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Printf("shutdown error: %v", err)
	}
}

func envString(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
