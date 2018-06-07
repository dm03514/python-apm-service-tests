package main

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	log.SetLevel(log.InfoLevel)
}

func main() {
	var addr = flag.String("addr", "127.0.0.1:5000", "addr/port for the tes")
	var path = flag.String("path", "/", "path to run test against")
	var cmd = flag.String("cmd", "wait-ready", "command to use")
	var testServerAddr = flag.String("test-server-addr", ":9000", "")
	flag.Parse()
	var err error

	switch *cmd {
	case "wait-ready":
		err = waitHTTPReady(*addr, *path, time.Second*15)
		if err != nil {
			panic(err)
		}
	case "http-surfacer-metrics-correct":
		err = testHTTPSurfacer(*addr, *path, *testServerAddr, time.Second*15)
	default:
		panic(fmt.Errorf("Unknown cmd: %s", *cmd))
	}

	if err != nil {
		panic(err)
	}
}
