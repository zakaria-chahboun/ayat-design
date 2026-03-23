package web

import (
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/zakaria-chahboun/AyatDesingBot/web/components"
)

func Run(port string) error {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	r.Get("/", indexHandler)

	safeFileServer(r, "/static", "web/static")
	safeFileServer(r, "/examples", "examples")

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		slog.Info("Web server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Web server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down web server...")
	return srv.Close()
}

func safeFileServer(r chi.Router, urlPath string, fsRoot string) {
	root, err := os.Getwd()
	if err != nil {
		return
	}
	fs := http.Dir(filepath.Join(root, fsRoot))
	handler := http.StripPrefix(urlPath, http.FileServer(fs))
	r.Handle(urlPath+"/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "/") {
			http.NotFound(w, req)
			return
		}
		handler.ServeHTTP(w, req)
	}))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	components.IndexPage().Render(r.Context(), w)
}
