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
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type client struct {
	name     string
	serverID string
	client   quiz.QuizClient
}

func NewClient(name string) *client {
	return &client{
		name: name,
	}
}

func (c *client) Start(ctx context.Context) error {
	conn, err := grpc.DialContext(ctx, "0.0.0.0:50051", grpc.WithInsecure())
	if err != nil {
		return err
	}

	c.client = quiz.NewQuizClient(conn)

	if err = c.login(ctx); err != nil {
		return err
	}

	return nil
}

func (c *client) login(ctx context.Context) error {
	res, err := c.client.Register(ctx, &quiz.RegisterRequest{
		Name:     c.name,
		ServerId: c.serverID,
	})

	if err != nil {
		return err
	}

	fmt.Println(res)

	c.Stream(ctx)

	return nil
}

func (c *client) Stream(ctx context.Context) error {
	md := metadata.New(map[string]string{"player": c.name})
	ctx = metadata.NewOutgoingContext(ctx, md)

	streamer, err := c.client.Stream(ctx)
	if err != nil {
		return err
	}

	go c.StreamSend(streamer)

	return c.StreamReceive(streamer)
}

func (c *client) StreamReceive(streamer quiz.Quiz_StreamClient) error {
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

func (c *client) StreamSend(streamer quiz.Quiz_StreamClient) error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanLines)

	for {
		select {
		case <-streamer.Context().Done():
			return nil
		default:
			if scanner.Scan() {
				streamer.Send(&quiz.StreamRequest{
					Message: scanner.Text(),
				})
			}
		}
	}
}
