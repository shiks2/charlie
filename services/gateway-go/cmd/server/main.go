package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	ocpp16 "github.com/lorenzodonini/ocpp-go/ocpp1.6"
	"github.com/lorenzodonini/ocpp-go/ws"

	"github.com/shiks2/charlie-gateway/internal/db"
	"github.com/shiks2/charlie-gateway/internal/emitter"
	"github.com/shiks2/charlie-gateway/internal/ocpp"
	"github.com/shiks2/charlie-gateway/internal/store"
)

func main() {
	if !loadEnvFile() {
		log.Println("no .env file found, reading from environment directly")
	}

	databaseURL := mustEnv("DATABASE_URL")
	orgID := mustEnv("ORG_ID")

	initCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	database, err := db.New(initCtx, databaseURL)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer database.Close()

	st := store.NewMemoryStore()

	// 1. Initialize Emitter (Using JSON for now, can be swapped for Kafka)
	baseEmitter := emitter.NewJSONEmitter()

	// 2. Wrap with AsyncEmitter for decoupling
	asyncEmitter := emitter.NewAsyncEmitter(baseEmitter, 1024, 5)
	asyncEmitter.Start()

	handler := ocpp.NewCharlieCoreHandler(st, asyncEmitter, database, orgID)

	wsServer := ws.NewServer()
	wsServer.AddSupportedSubprotocol("ocpp1.6")
	wsServer.AddSupportedSubprotocol("ocpp1.6j")
	wsServer.SetCheckClientHandler(checkClientHandler)

	centralSystem := ocpp16.NewCentralSystem(nil, wsServer)
	centralSystem.SetCoreHandler(handler)

	centralSystem.SetNewChargePointHandler(func(chargePoint ocpp16.ChargePointConnection) {
		log.Printf("new charge point connected: %s", chargePoint.ID())
	})

	// Graceful shutdown handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("charlie gateway listening on :8887")
		centralSystem.Start(8887, "/ocpp/{ws}")
	}()

	sig := <-sigs
	log.Printf("received signal %v, shutting down...", sig)

	// Gracefully shut down the emitter (drains the buffer)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		asyncEmitter.Close()
		close(done)
	}()

	select {
	case <-done:
		log.Println("emitter shut down gracefully")
	case <-shutdownCtx.Done():
		log.Println("shutdown timed out, some events may have been lost")
	}

	log.Println("gateway stopped")
}

func loadEnvFile() bool {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		if err := godotenv.Load(); err == nil {
			log.Println("loaded .env from current directory")
			return true
		}
		return false
	}

	moduleRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	candidates := []string{
		".env",
		filepath.Join(moduleRoot, ".env"),
	}

	for _, candidate := range candidates {
		if err := godotenv.Load(candidate); err == nil {
			log.Printf("loaded env from %s", candidate)
			return true
		}
	}

	return false
}

func mustEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		panic(fmt.Sprintf("%s is required", name))
	}

	return value
}

func checkClientHandler(id string, r *http.Request) bool {
	log.Printf("[CONNECT] charger %s requested subprotocol: %s", id, r.Header.Get("Sec-WebSocket-Protocol"))
	return true
}
