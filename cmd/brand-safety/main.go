// Package main 是 adortb-brand-safety 服务入口。
// 提供 IAB Content Taxonomy 分类、关键词黑名单检测和广告主安全检查。
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adortb/adortb-brand-safety/internal/api"
	"github.com/adortb/adortb-brand-safety/internal/blocklist"
	"github.com/adortb/adortb-brand-safety/internal/classifier"
	"github.com/adortb/adortb-brand-safety/internal/scorer"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	port := getEnv("PORT", "8092")
	addr := ":" + port

	sc := scorer.New()
	urlCls := classifier.NewURLClassifier()
	kwCls := classifier.NewKeywordClassifier()
	advBL := blocklist.NewAdvertiserBlocklist()
	platBL := blocklist.NewPlatformBlocklist()

	h := api.New(sc, urlCls, kwCls, advBL, platBL)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Prometheus metrics
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = fmt.Fprintln(w, "# brand-safety metrics placeholder")
	})

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("brand-safety server starting", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Error("server error", slog.String("error", err.Error()))
		os.Exit(1)
	case sig := <-quit:
		log.Info("shutting down", slog.String("signal", sig.String()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("brand-safety server stopped")
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
