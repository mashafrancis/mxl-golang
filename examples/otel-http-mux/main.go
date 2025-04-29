package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	mxlgolang "github.com/mashafrancis/mxl-golang"
	"github.com/mashafrancis/mxl-golang/http/middleware"
	"github.com/mashafrancis/mxl-golang/logs"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	slog.Info("Testing otlp http")

	if err := run(); err != nil {
		panic(err)
	}
}

func run() (err error) {
	// Handle SIGINT (CTRL+C) gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// OTEL SETUP //.
	cfg := mxlgolang.NewConfig().SetOTLPEndpoint("http://localhost:4318").SetInsecure(true).SetHeader("authorization", "testauth").SetServiceName("example-otlp-http").SetUseHttpOverGrpc(true)

	otelShutdown, err := mxlgolang.Init(ctx, cfg)
	if err != nil {
		return (err)
	}
	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()
	logs.AttachZerologToOTel()

	// Start HTTP server. //
	srv := &http.Server{
		Addr:         ":12345",
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
		ReadTimeout:  time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      newHTTPHandler(),
	}
	srvErr := make(chan error, 1)
	go func() {
		srvErr <- srv.ListenAndServe()
	}()

	// Wait for interruption.
	select {
	case err = <-srvErr:
		// Error when starting HTTP server.
		return
	case <-ctx.Done():
		// Wait for first CTRL+C.
		// Stop receiving signal notifications as soon as possible.
		stop()
	}

	// When Shutdown is called, ListenAndServe immediately returns ErrServerClosed.
	err = srv.Shutdown(context.Background())
	return
}

func newHTTPHandler() http.Handler {
	r := mux.NewRouter()

	middleware.AttachMux(r, "example-otlp-http")

	// Register handlers.
	r.HandleFunc("/rolldice/", rolldice)
	r.HandleFunc("/rolldice/{player}", rolldice)

	// Add HTTP instrumentation for the whole server.
	return r
}

func rolldice(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tracer := otel.Tracer("rolling dices")
	ctx, span := tracer.Start(ctx, "rolldice", trace.WithAttributes(attribute.String("player", mux.Vars(r)["player"])))
	defer span.End()
	roll := 1 + rand.Intn(6)
	resp := strconv.Itoa(roll) + "\n"
	log.Info().Str("ss", "something very random").Ctx(ctx).Msgf("rolling dice: %s\n", resp)
	if _, err := io.WriteString(w, resp); err != nil {

		log.Info().Ctx(ctx).Msgf("Write failed: %v\n", err)
	}
}
