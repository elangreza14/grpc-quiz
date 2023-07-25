package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"

	quiz "github.com/elangreza14/grpc-quiz/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	eventType int
	playState int

	event struct {
		eventType eventType
		payload   any
	}

	server struct {
		serverID      string
		users         map[string]chan *quiz.StreamResponse
		priorityQueue chan *event
		state         playState

		quiz.UnimplementedQuizServer
	}
)

const (
	InsertUsers eventType = iota
	Broadcast
	StartGame

	Waiting playState = iota
	Playing
	Finish
)

func NewServer(serverID string) *server {
	srv := &server{
		serverID:      serverID,
		users:         map[string]chan *quiz.StreamResponse{},
		priorityQueue: make(chan *event, 100),
		state:         Waiting,
	}

	// listen all the event
	go srv.ListenerPriorityQueue()

	return srv
}

func (s *server) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	srv := grpc.NewServer()
	quiz.RegisterQuizServer(srv, s)

	go s.ListenTerminal(ctx)

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		return err
	}

	if err := srv.Serve(listener); err != nil {
		return err
	}

	// wait until ctx is done
	<-ctx.Done()

	srv.GracefulStop()

	return nil
}

func (s *server) Register(ctx context.Context, req *quiz.RegisterRequest) (*quiz.RegisterResponse, error) {
	_, ok := s.users[req.Name]
	if ok {
		return nil, status.Errorf(codes.AlreadyExists, "username already exist")
	}

	s.PublisherPriorityQueue(&event{
		eventType: InsertUsers,
		payload:   req.Name,
	})

	return &quiz.RegisterResponse{
		Message: fmt.Sprintf("hi %v, welcome to the game", req.Name),
	}, nil
}

func (s *server) Stream(stream quiz.Quiz_StreamServer) error {

	// go func() {
	// 	// stream.Send()
	// }()

	for {
		req, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		fmt.Println(req)
		// s.PublisherPriorityQueue(&event{
		// 	eventType: Broadcast,
		// 	message:   "",
		// })
	}

	<-stream.Context().Done()
	return stream.Context().Err()
}

func (s *server) PublisherPriorityQueue(evt *event) {
	s.priorityQueue <- evt
}

func (s *server) ListenerPriorityQueue() {
	for evt := range s.priorityQueue {
		switch evt.eventType {
		case InsertUsers:
			s.users[evt.payload.(string)] = make(chan *quiz.StreamResponse, 100)
			fmt.Printf("player %s joined. total %d players \n", evt.payload, len(s.users))
		case StartGame:
			s.state = Playing
			fmt.Println("game started")
		case Broadcast:
		default:
			// no operation
		}
	}
}

func (s *server) ListenTerminal(ctx context.Context) {
	fmt.Println("start the game. minimum two player (Y/N)")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanLines)

	for {
		select {
		case <-ctx.Done():
			break
		default:
			if scanner.Scan() {
				if s.state == Waiting && len(s.users) >= 2 {
					s.PublisherPriorityQueue(&event{
						eventType: StartGame,
					})
				}
			}
		}
	}
}
