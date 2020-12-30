package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/z0rr0/send/cfg"
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
	logger.Info("\n%v\nstorage: %v\nlisten addr: %v", info, c.Storage.Db, srv.Addr)
}
