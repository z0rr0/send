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

func versionInfo() string {
	return fmt.Sprintf("%v\n\tVersion: %v\n\tRevision: %v\n\tBuild date: %v\n\tGo version: %v",
		Name, Version, Revision, BuildDate, GoVersion,
	)
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

	info := versionInfo()
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
	baseContext := tpl.Set(context.Background(), templates)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start, code := time.Now(), http.StatusOK
		ctx, e := logging.NewWithContext(baseContext, "")
		if e != nil {
			logger.Error("init new logging context: %v", e)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		lg, e := logging.Get(ctx)
		if e != nil {
			logger.Error("read new logging context: %v", e)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		lg.Info("get new request")
		defer func() {
			lg.Info("%-5v %v\t%-12v\t%v",
				r.Method,
				code,
				time.Since(start),
				r.URL.String(),
			)
		}()
		e = handle.Main(ctx, w, r)
		if e != nil {
			lg.Error("error: %v", e)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		//switch r.URL.Path {
		//case "/version":
		//	code, err = http.StatusOK, getVersion(w)
		//case "/":
		//	code, err = web.Index(w, r, cfg)
		//case "/upload":
		//	code, err = web.Upload(w, r, cfg)
		//case "/u":
		//	code, err = web.UploadShort(w, r, cfg)
		//default:
		//	code, err = web.Download(w, r, cfg)
		//}
		//if err != nil {
		//	loggerError.Println(err)
		//}
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
