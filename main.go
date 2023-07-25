package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var playerName = flag.String("p", "", "player name is optional, if exist will create client runner.")

type runner interface {
	Start(context.Context) error
}

func main() {
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
	if *playerName != "" {
		Runner = NewClient(*playerName)
	}

	// start the runner
	if err := Runner.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
