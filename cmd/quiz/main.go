package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	client "github.com/elangreza14/grpc-quiz/cmd/client"
	server "github.com/elangreza14/grpc-quiz/cmd/server"
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
	var Runner runner = server.NewServer()
	if *playerName != "" {
		Runner = client.NewClient(*playerName)
	}

	// start the runner
	if err := Runner.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
