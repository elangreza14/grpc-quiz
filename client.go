package main

import (
	"context"
	"fmt"

	quiz "github.com/elangreza14/grpc-quiz/proto"
	"google.golang.org/grpc"
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

	return nil
}
