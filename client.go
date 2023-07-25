// package main
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	quiz "github.com/elangreza14/grpc-quiz/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Client is ...
type Client struct {
	name     string
	serverID string
	client   quiz.QuizClient
}

// NewClient is ...
func NewClient(name string) *Client {
	return &Client{
		name: name,
	}
}

// Start is ...
func (c *Client) Start(ctx context.Context) error {
	conn, err := grpc.DialContext(ctx, ":9950", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	c.client = quiz.NewQuizClient(conn)

	return c.login(ctx)
}

func (c *Client) login(ctx context.Context) error {
	res, err := c.client.Register(ctx, &quiz.RegisterRequest{
		Name:     c.name,
		ServerId: c.serverID,
	})
	if err != nil {
		return err
	}

	fmt.Println(res)

	return c.stream(ctx)
}

func (c *Client) stream(ctx context.Context) error {
	md := metadata.New(map[string]string{"player": c.name})
	ctx = metadata.NewOutgoingContext(ctx, md)

	streamer, err := c.client.Stream(ctx)
	if err != nil {
		return err
	}

	go c.streamSend(streamer)

	return c.streamReceive(streamer)
}

func (c *Client) streamReceive(streamer quiz.Quiz_StreamClient) error {
	for {
		res, err := streamer.Recv()

		if sts, ok := status.FromError(err); ok && sts.Code() == codes.Canceled {
			fmt.Printf("got err %v", sts.Code())
			return fmt.Errorf("got error %v", sts.Code())
		} else if err == io.EOF {
			fmt.Printf("got err %v", err.Error())
			return errors.New("stream closed")
		} else if err != nil {
			fmt.Printf("got err %v", err.Error())
			return err
		}

		switch res.Event.(type) {
		case *quiz.StreamResponse_ServerAnnouncement:
			fmt.Println(res.GetServerAnnouncement().Message)
		case *quiz.StreamResponse_ServerShutdown:
			fmt.Println("server shuting down")
		}
	}
}

func (c *Client) streamSend(streamer quiz.Quiz_StreamClient) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanLines)

	for {
		select {
		case <-streamer.Context().Done():
			return
		default:
			if scanner.Scan() {
				message := &quiz.StreamRequest{Message: scanner.Text()}
				if s, ok := status.FromError(streamer.Send(message)); ok {
					if s.Code() != codes.OK {
						fmt.Printf("got error %v", s.Code())
						return
					}
				}
			}
		}
	}
}
