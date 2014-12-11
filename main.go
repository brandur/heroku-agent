package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"

	flag "github.com/ogier/pflag"
)

const (
	Version = "0.1.0"
)

var (
	logger *log.Logger
)

func initLogger(verbose bool) *log.Logger {
	var writer io.Writer
	if verbose {
		writer = os.Stdout
	} else {
		writer = ioutil.Discard
	}
	return log.New(writer, "[heroku-agent] ", log.Ltime)
}

func main() {
	verbose := flag.BoolP("verbose", "v", false, "Verbose mode")
	flag.Parse()

	logger = initLogger(*verbose)

	switch {
	case len(flag.Args()) == 0:
		Serve()
	case len(flag.Args()) >= 1:
		RunCommand(flag.Arg(0), flag.Args()[1:])
	default:
		printUsage()
		os.Exit(2)
	}
}
