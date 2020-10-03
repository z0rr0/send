package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/z0rr0/send/cfg"
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

	// internal loggers
	loggerError = log.New(os.Stderr, fmt.Sprintf("ERROR [%v]: ", Name),
		log.Ldate|log.Ltime|log.Lshortfile)
	loggerInfo = log.New(os.Stdout, fmt.Sprintf("INFO [%v]: ", Name),
		log.Ldate|log.Ltime|log.Lshortfile)
)

func versionInfo() string {
	return fmt.Sprintf("%v\n\tVersion: %v\n\tRevision: %v\n\tBuild date: %v\n\tGo version: %v",
		Name, Version, Revision, BuildDate, GoVersion,
	)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			loggerError.Printf("abnormal termination [%v]: \n\t%v\n", Version, r)
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
		if err := c.Close(); err != nil {
			loggerError.Println(err)
		}
	}()
	timeout := c.Timeout()
	srv := &http.Server{
		Addr:           c.Addr(),
		Handler:        http.DefaultServeMux,
		ReadTimeout:    timeout,
		WriteTimeout:   timeout,
		MaxHeaderBytes: c.MaxFileSize(),
		ErrorLog:       loggerInfo,
	}
	loggerInfo.Printf("\n%v\nstorage: %v\nlisten addr: %v\n", info, c.Storage.Db, srv.Addr)
}
