package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	mode string
	name string
)

func main() {
	rand.Seed(time.Now().UnixNano())
	flag.StringVar(&mode, "m", "client", "mode, can be client or server")
	flag.StringVar(&name, "n", "", "username for the client")
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

	if mode == "server" {
		fmt.Println("running server mode")
		serverID := fmt.Sprintf("%d", rand.Intn(1000))
		server := NewServer(serverID)
		err := server.Start(ctx)
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Println("running client mode")
		client := NewClient(name)
		err := client.Start(ctx)
		if err != nil {
			panic(err)
		}
	}
}
