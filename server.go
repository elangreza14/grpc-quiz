// package main
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/elangreza14/grpc-quiz/internal/usecase"
	quiz "github.com/elangreza14/grpc-quiz/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type (
	// Server is default structure for creating communication
	Server struct {
		Room *usecase.Room

		quiz.UnimplementedQuizServer
	}
)

// NewServer define a grpc server
func NewServer() *Server {
	return &Server{
		Room: usecase.NewRoom(),
	}
}

// Start is gateway to grpc server
func (s *Server) Start(ctx context.Context) error {
	srv := grpc.NewServer()
	quiz.RegisterQuizServer(srv, s)

	// listen all the event
	go s.Room.ListenQueue(ctx)
	go s.listenTerminal(ctx)

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		return err
	}

	go func() {
		_ = srv.Serve(listener)
	}()

	// wait until ctx is done
	<-ctx.Done()

	s.Room.BroadcastToShutdown()

	srv.GracefulStop()

	return nil
}

// Register is handler for register player
func (s *Server) Register(_ context.Context, req *quiz.RegisterRequest) (*quiz.RegisterResponse, error) {
	_, ok := s.Room.GetPlayerDetail(req.Name)
	if ok {
		return nil, status.Errorf(codes.AlreadyExists, "player already exist")
	}

	s.Room.PublishQueue(&usecase.Event{
		EventType: usecase.InsertPlayer,
		Payload:   req.Name,
	})

	return &quiz.RegisterResponse{
		Message: fmt.Sprintf("hi %v, welcome to the game", req.Name),
	}, nil
}

// Stream is handler for streaming player state
func (s *Server) Stream(stream quiz.Quiz_StreamServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Errorf(codes.Unauthenticated, "player not found")
	}

	name := md.Get("player")
	if len(name) == 0 {
		return status.Errorf(codes.Unauthenticated, "player not found")
	}

	streamPlayer, ok := s.Room.GetPlayerDetail(name[0])
	if !ok {
		return status.Errorf(codes.Unauthenticated, "player not found")
	}
	defer func() {
		close(streamPlayer)
		s.Room.RemovePlayer(name[0])
	}()

	go s.streamSend(stream, streamPlayer)

	// receive the stream
	for {
		req, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		s.Room.PublishQueue(&usecase.Event{
			EventType: usecase.Broadcast,
			Payload:   req.Message,
		})
	}
}

func (s *Server) streamSend(stream quiz.Quiz_StreamServer, streamPlayer <-chan *quiz.StreamResponse) {
	for {
		select {
		case <-stream.Context().Done():
			return
		case msg := <-streamPlayer:
			if s, ok := status.FromError(stream.Send(msg)); ok {
				if s.Code() != codes.OK {
					fmt.Printf("got error %v\n", s.Code())
					return
				}
			}
		}
	}
}

func (s *Server) listenTerminal(ctx context.Context) {
	fmt.Println("start the game. minimum two player (Y/N)")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanLines)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if scanner.Scan() {
				if s.Room.State == usecase.Waiting && s.Room.TotalPlayer() >= 2 {
					s.Room.PublishQueue(&usecase.Event{
						EventType: usecase.StartGame,
					})
				}
			}
		}
	}
}
