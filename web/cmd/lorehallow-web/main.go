// Command lorehallow-web builds lorehallow.com (ADR 0011).
//
//	lorehallow-web build [-root ..] [-out dist]    write the site to web/dist/
//	lorehallow-web serve [-root ..] [-addr ...]    preview it locally, drafts
//	                                               included, rebuilt on change
//
// Run it from web/ (make web-build and make web-dev do).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dunamismax/lorehallow/web"
	"github.com/dunamismax/lorehallow/web/internal/site"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "build":
		err = build(os.Args[2:])
	case "serve":
		err = serve(os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "lorehallow-web:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: lorehallow-web build [-root ..] [-out dist] | serve [-root ..] [-addr 127.0.0.1:4321]")
	os.Exit(2)
}

func build(args []string) error {
	fl := flag.NewFlagSet("build", flag.ExitOnError)
	root := fl.String("root", "..", "repository root")
	out := fl.String("out", "dist", "output directory (replaced)")
	_ = fl.Parse(args)
	start := time.Now()
	o, err := site.Build(site.Config{Root: *root, Files: web.Files, Git: true})
	if err != nil {
		return err
	}
	if err := o.Write(*out); err != nil {
		return err
	}
	var size int
	for _, b := range o.Files {
		size += len(b)
	}
	fmt.Printf("lorehallow-web: %d pages, %d files, %.1f MB in %s -> %s\n",
		len(o.Pages), len(o.Files), float64(size)/(1<<20), time.Since(start).Round(time.Millisecond), *out)
	return nil
}

// serve builds into memory and serves the result, rebuilding before a
// request whenever a source file has changed. Templates and CSS are read
// from disk, not the embedded copy, so edits show up on reload.
func serve(args []string) error {
	fl := flag.NewFlagSet("serve", flag.ExitOnError)
	root := fl.String("root", "..", "repository root")
	addr := fl.String("addr", "127.0.0.1:4321", "listen address")
	_ = fl.Parse(args)
	absRoot, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	s := &server{root: absRoot, files: os.DirFS(filepath.Join(absRoot, "web"))}
	if err := s.rebuild(); err != nil {
		slog.Error("build failed", "err", err)
	}
	srv := &http.Server{Addr: *addr, Handler: s, ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shut, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shut)
	}()
	fmt.Printf("lorehallow-web: serving http://%s (drafts included; reload to rebuild)\n", *addr)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

type server struct {
	root  string
	files fs.FS

	mu    sync.Mutex
	stamp string
	out   *site.Output
	err   error
}

// sources are watched for changes.
var sources = []string{"BUILD.md", "web/templates", "web/static", "web/content", "web/data", "web/assets/art",
	"docs/walkthroughs", "art/style/palette.hex", ".git/HEAD", ".git/refs/heads"}

func (s *server) fingerprint() string {
	var b strings.Builder
	for _, src := range sources {
		_ = filepath.WalkDir(filepath.Join(s.root, src), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if info, err := d.Info(); err == nil {
				fmt.Fprintf(&b, "%s %d %d\n", p, info.Size(), info.ModTime().UnixNano())
			}
			return nil
		})
	}
	return b.String()
}

func (s *server) rebuild() error {
	stamp := s.fingerprint()
	if stamp == s.stamp && (s.out != nil || s.err != nil) {
		return s.err
	}
	start := time.Now()
	out, err := site.Build(site.Config{Root: s.root, Files: s.files, Drafts: true, Git: true})
	s.stamp, s.err = stamp, err
	if err == nil {
		s.out = out
		slog.Info("built", "pages", len(out.Pages), "took", time.Since(start).Round(time.Millisecond))
	}
	return err
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	err := s.rebuild()
	out := s.out
	s.mu.Unlock()
	if err != nil {
		http.Error(w, "build failed:\n\n"+err.Error(), http.StatusInternalServerError)
		return
	}
	p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if strings.HasSuffix(r.URL.Path, "/") && p != "" {
		p += "/"
	}
	switch {
	case p == "" || p == ".":
		p = "index.html"
	case strings.HasSuffix(p, "/"):
		p += "index.html"
	}
	data, ok := out.Files[p]
	if !ok && !strings.Contains(path.Base(p), ".") {
		if _, dir := out.Files[p+"/index.html"]; dir {
			http.Redirect(w, r, "/"+p+"/", http.StatusMovedPermanently)
			return
		}
	}
	status := http.StatusOK
	if !ok {
		p, data, status = "404.html", out.Files["404.html"], http.StatusNotFound
	}
	if ct := contentType(p); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'none'; font-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func contentType(p string) string {
	switch path.Ext(p) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".png":
		return "image/png"
	case ".svg":
		return "image/svg+xml"
	case ".woff":
		return "font/woff"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml; charset=utf-8"
	case ".txt":
		return "text/plain; charset=utf-8"
	case ".ico":
		return "image/x-icon"
	}
	return ""
}
