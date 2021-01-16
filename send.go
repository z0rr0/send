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
	"runtime/debug"
	"syscall"
	"time"

	"github.com/z0rr0/send/cfg"
	"github.com/z0rr0/send/db"
	"github.com/z0rr0/send/handle"
	"github.com/z0rr0/send/logging"
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
	defer func() {
		if r := recover(); r != nil {
			logging.ErrorLog().Printf("abnormal termination [%v]: %v\n%v", Version, r, string(debug.Stack()))
		}
	}()
	version := flag.Bool("version", false, "show version")
	config := flag.String("config", Config, "configuration file")
	logFile := flag.String("log", "", "log file name (default stdout)")
	flag.Parse()

	ver := &handle.Version{Version: Version, Revision: Revision, Build: BuildDate, Environment: GoVersion}
	info := versionInfo(ver)
	if *version {
		// show oly version
		fmt.Println(info)
		return
	}
	// configure custom logging
	if fileName := *logFile; fileName == "" {
		logging.SetUp(Name, os.Stdout, os.Stderr, log.LstdFlags, log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		logFileFd, err := logging.SetUpFile(Name, fileName, log.LstdFlags, log.Ldate|log.Ltime|log.Lshortfile)
		if err != nil {
			panic(err)
		}
		defer func() {
			if e := logFileFd.Close(); e != nil {
				logging.ErrorLog().Printf("close log file: %v", e)
			}
		}()
	}
	logger := logging.New("main")
	// read config and check html templates
	c, err := cfg.New(*config)
	if err != nil {
		panic(err)
	}
	delItem := make(chan db.Item, 1) // to delete items after attempts expirations
	defer func() {
		if e := c.Close(); e != nil {
			logger.Error("close cfg error: %v", e)
		}
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
		reqLogger := logging.New("")
		reqLogger.Info("request\t%s", r.URL.String())
		params := &handle.Params{
			Log: reqLogger, DB: c.Storage.Db, Settings: &c.Settings, Request: r,
			Version: ver, DelItem: delItem, Storage: &c.Storage, Secure: c.Server.Secure,
		}
		r.BasicAuth()

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer func() {
			var checkCode bool
			if r := recover(); r != nil {
				reqLogger.Error("request panic: %v", r)
				code, checkCode = http.StatusInternalServerError, true
				reqLogger.Error("stack:\n%v\n", string(debug.Stack()))
			}
			reqLogger.Info("%-5v %v\t%-12v\t%v", r.Method, code, time.Since(start), r.URL.String())
			if checkCode && code == http.StatusInternalServerError {
				if params.IsAPI() {
					w.WriteHeader(code)
					if _, e := fmt.Fprint(w, "{\"error\": \"internal error\"}"); e != nil {
						reqLogger.Error("failed error response: %v", e)
					}
				} else {
					http.Error(w, "internal error", code)
				}
			}
			cancel()
		}()
		if params.IsAPI() {
			w.Header().Add("Content-Type", "application/json")
		}
		code = handle.Main(ctx, w, params)
	})
	// run GC monitoring
	gcShutdown := make(chan struct{}) // to close GC monitor
	gcStopped := make(chan struct{})  // to wait GC stopping
	go db.GCMonitor(delItem, gcShutdown, gcStopped, c.Storage.Db, c.GCPeriod(), c.DbPeriod(), logger)

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
	close(delItem)
	logger.Info("service %v stopped", Name)
}
