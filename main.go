package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var username string

type runner interface {
	Start(context.Context) error
}

func main() {
	flag.StringVar(&username, "u", "", "username for the client, if empty will run server mode.")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		signal.Stop(sig)
		close(sig)
		cancel()
	}()

	// default mode is client mode
	var Runner runner = NewServer()
	if username != "" {
		Runner = NewClient(username)
	}

	// start the runner
	if err := Runner.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
