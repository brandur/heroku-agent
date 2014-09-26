package main

import (
	flag "github.com/ogier/pflag"
	"io"
	"io/ioutil"
	"log"
	"os"
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
	case len(flag.Args()) == 1:
		Run(flag.Arg(0))
	default:
		printUsage()
	}
}
