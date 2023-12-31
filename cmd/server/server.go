// Package server ....
package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"

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
		Room     *usecase.Room
		Terminal *usecase.Terminal
		PowerOff chan bool

		quiz.UnimplementedQuizServer
	}
)

// NewServer define a grpc server
func NewServer() *Server {
	return &Server{
		Room:                    usecase.NewRoom(),
		Terminal:                usecase.NewTerminal(),
		PowerOff:                make(chan bool),
		UnimplementedQuizServer: quiz.UnimplementedQuizServer{},
	}
}

// Start is gateway to grpc server
func (s *Server) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

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
	select {
	case <-ctx.Done():
		break
	case <-s.Room.PowerOff:
		break
	}

	fmt.Println("shutting down the server")

	s.Room.ShutdownClient()

	srv.GracefulStop()

	return nil
}

// Register is handler for register player
func (s *Server) Register(_ context.Context, req *quiz.RegisterRequest) (*quiz.Message, error) {
	_, ok := s.Room.GetPlayerDetail(req.Player)
	if ok {
		return nil, status.Errorf(codes.AlreadyExists, "player already exist")
	}

	s.Room.PublishQueue(&usecase.Event{
		EventType: usecase.InsertPlayer,
		Payload:   req.Player,
	})

	return &quiz.Message{
		Message: fmt.Sprintf("hi %v, welcome to the game", req.Player),
	}, nil
}

// Stream is handler for streaming player state
func (s *Server) Stream(stream quiz.Quiz_StreamServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Errorf(codes.Unauthenticated, "player not found")
	}

	player := md.Get("player")
	if len(player) == 0 {
		return status.Errorf(codes.Unauthenticated, "player not found")
	}

	streamPlayer, ok := s.Room.GetPlayerDetail(player[0])
	if !ok {
		return status.Errorf(codes.Unauthenticated, "player not found")
	}

	defer func() {
		close(streamPlayer)
		s.Room.RemovePlayer(player[0])
		fmt.Printf("player %s left. total %d players \n", player[0], s.Room.TotalPlayer())
	}()

	// send stream from server
	go s.streamSend(stream, streamPlayer)

	// receive stream from client
	return s.streamReceive(player[0], stream)
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

func (s *Server) streamReceive(name string, stream quiz.Quiz_StreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		// if game not yet started
		if !s.Room.Started {
			s.Room.PublishQueue(&usecase.Event{
				EventType: usecase.Broadcast,
				Payload:   req.Message,
			})
			continue
		}

		msg := strings.ToLower(req.Message)
		if msg == "y" || msg == "n" {
			s.Room.PublishQueue(&usecase.Event{
				EventType: usecase.SubmitAnswer,
				Payload: usecase.SubmitAnswerPayload{
					Name:   name,
					Answer: msg == "y",
				},
			})
		} else {
			s.Room.PublishQueue(&usecase.Event{
				EventType: usecase.BroadcastPersonal,
				Payload: usecase.BroadcastPersonalPayload{
					Name:    name,
					Message: "only accept (Y/N) when game is started",
				},
			})
		}
	}
}

func (s *Server) listenTerminal(ctx context.Context) {
	fmt.Println("Waiting players to join. press (Y) to start")
	for {
		select {
		case <-ctx.Done():
			return
		default:
			decision, err := s.Terminal.ValBoolean()
			if err != nil {
				fmt.Println(err.Error())
				continue
			}

			if decision && s.Room.TotalPlayer() > 0 {
				s.Room.PublishQueue(&usecase.Event{
					EventType: usecase.StartGame,
				})
			}
		}
	}
}
