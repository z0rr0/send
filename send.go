package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/z0rr0/send/cfg"
	"github.com/z0rr0/send/db"
	"github.com/z0rr0/send/handle"
	"github.com/z0rr0/send/logging"
	"github.com/z0rr0/send/tpl"
)

const (
	// Name is a program name.
	Name = "Send"
	// Config is default configuration file name.
	Config = "config.toml"
)

var (
	// Version is git version
	Version = ""
	// Revision is revision number
	Revision = ""
	// BuildDate is build date
	BuildDate = ""
	// GoVersion is runtime Go language version
	GoVersion = runtime.Version()
)

func versionInfo(ver *handle.Version) string {
	return fmt.Sprintf("%s\n%s", Name, ver.String())
}

func main() {
	logging.SetUp(Name, os.Stdout, os.Stderr, log.LstdFlags, log.Ldate|log.Ltime|log.Lshortfile)
	logger, err := logging.New("main")
	if err != nil {
		if _, e := fmt.Fprintf(os.Stderr, "failed logging creation: %v\n", err); e != nil {
			err = fmt.Errorf("failed logging creation: %w", err)
			fmt.Printf("errors: %v / %v\n", err, e)
		}
		os.Exit(1)
	}
	defer func() {
		if r := recover(); r != nil {
			logger.Info("abnormal termination [%v]: \n\t%v", Version, r)
		}
	}()
	version := flag.Bool("version", false, "show version")
	config := flag.String("config", Config, "configuration file")
	flag.Parse()

	ver := &handle.Version{Version: Version, Revision: Revision, Build: BuildDate, Environment: GoVersion}
	info := versionInfo(ver)
	if *version {
		fmt.Println(info)
		return
	}
	c, err := cfg.New(*config)
	if err != nil {
		panic(err)
	}
	templates, err := tpl.Load(c.Settings.Templates)
	if err != nil {
		panic(err)
	}
	delItem := make(chan db.Item) // to delete items by after attempts expirations
	defer func() {
		if e := c.Close(); e != nil {
			logger.Error("close cfg error: %v", e)
		}
		close(delItem)
	}()
	timeout := c.Timeout()
	srv := &http.Server{
		Addr:           c.Addr(),
		Handler:        http.DefaultServeMux,
		ReadTimeout:    timeout,
		WriteTimeout:   timeout,
		MaxHeaderBytes: c.MaxFileSize(),
		ErrorLog:       logging.ErrorLog(),
	}
	logger.Info("\n%v\n%s\nlisten addr: %v", info, c.Storage.String(), srv.Addr)
	logger.Info("static=%v", c.Settings.Static)

	fileServer := http.FileServer(http.Dir(c.Settings.Static))
	http.Handle("/static/", http.StripPrefix("/static", fileServer))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start, code := time.Now(), http.StatusOK
		lg, e := logging.New("")
		if e != nil {
			logger.Error("init logging context: %v", e)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		lg.Info("income request")
		defer func() {
			lg.Info("%-5v %v\t%-12v\t%v", r.Method, code, time.Since(start), r.URL.String())
			if code == http.StatusInternalServerError {
				http.Error(w, "internal error", code)
			}
		}()
		params := &handle.Params{
			Log: lg, Settings: &c.Settings, Request: r, Templates: templates, Version: ver, DelItem: delItem,
		}
		if strings.HasPrefix(r.URL.Path, "/api") {
			w.Header().Add("Content-Type", "application/json")
		}
		e = handle.Main(w, params)
		if e != nil {
			lg.Error("error: %v", e)
			code = http.StatusInternalServerError
			return
		}
	})
	// run GC monitoring
	gcShutdown := make(chan struct{}) // to close GC monitor
	gcStopped := make(chan struct{})  // to wait GC stopping
	go db.GCMonitor(delItem, gcShutdown, gcStopped, c.Storage.Db, c.GCPeriod(), logger)

	idleConnsClosed := make(chan struct{}) // to wait http server shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, os.Signal(syscall.SIGTERM), os.Signal(syscall.SIGQUIT))
		<-sigint

		ctx, cancel := context.WithTimeout(context.Background(), c.Shutdown())
		defer cancel()
		if e := srv.Shutdown(ctx); e != nil {
			logger.Error("HTTP server shutdown: %v", e)
		} else {
			logger.Info("HTTP server successfully stopped")
		}
		close(idleConnsClosed)
		close(gcShutdown)
	}()
	if e := srv.ListenAndServe(); e != http.ErrServerClosed {
		logger.Error("HTTP server ListenAndServe: %v", e)
	}
	<-idleConnsClosed
	<-gcStopped
	logger.Info("service %v stopped", Name)
}
