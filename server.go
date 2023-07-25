// package main
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
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type (
	eventType int
	playState int

	event struct {
		eventType eventType
		payload   any
	}

	// Server is default structure for creating communication
	Server struct {
		players       map[string]chan *quiz.StreamResponse
		priorityQueue chan *event
		state         playState

		quiz.UnimplementedQuizServer
	}
)

const (
	//  InsertPlayer is event for inserting players to players
	InsertPlayer eventType = iota
	//  Broadcast is event for broadcast all the player
	Broadcast
	//  StartGame is event for start the game
	StartGame

	// Waiting is state when waiting all the players
	Waiting playState = iota
	// Started is state when game is started
	Started
	// TODO Finish is state when game is finished
	// TODO Finish
)

// NewServer define a grpc server
func NewServer() *Server {
	return &Server{
		players:       map[string]chan *quiz.StreamResponse{},
		priorityQueue: make(chan *event, 100),
		state:         Waiting,
	}
}

// Start is gateway to grpc server
func (s *Server) Start(ctx context.Context) error {
	srv := grpc.NewServer()
	quiz.RegisterQuizServer(srv, s)

	// listen all the event
	go s.listenerPriorityQueue(ctx)
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

	srv.GracefulStop()

	return nil
}

// Register is handler for register player
func (s *Server) Register(_ context.Context, req *quiz.RegisterRequest) (*quiz.RegisterResponse, error) {
	_, ok := s.players[req.Name]
	if ok {
		return nil, status.Errorf(codes.AlreadyExists, "player already exist")
	}

	s.publisherPriorityQueue(&event{
		eventType: InsertPlayer,
		payload:   req.Name,
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

	streamPlayer, ok := s.players[name[0]]
	if !ok {
		return status.Errorf(codes.Unauthenticated, "player not found")
	}

	go s.streamSender(stream, streamPlayer)

	for {
		req, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		s.publisherPriorityQueue(&event{
			eventType: Broadcast,
			payload:   req.Message,
		})
	}

	<-stream.Context().Done()
	return stream.Context().Err()
}

func (s *Server) streamSender(stream quiz.Quiz_StreamServer, streamPlayer <-chan *quiz.StreamResponse) {
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

func (s *Server) publisherPriorityQueue(evt *event) {
	s.priorityQueue <- evt
}

func (s *Server) listenerPriorityQueue(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-s.priorityQueue:
			switch evt.eventType {
			case InsertPlayer:
				s.players[evt.payload.(string)] = make(chan *quiz.StreamResponse, 100)
				fmt.Printf("player %s joined. total %d players \n", evt.payload, len(s.players))
			case StartGame:
				s.state = Started
				fmt.Println("game started")
			case Broadcast:
				for _, playerStream := range s.players {
					playerStream <- &quiz.StreamResponse{
						Timestamp: timestamppb.Now(),
						Event: &quiz.StreamResponse_ServerAnnouncement{
							ServerAnnouncement: &quiz.StreamResponse_Message{
								Message: evt.payload.(string),
							},
						},
					}
				}
			default:
				// no operation
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
				if s.state == Waiting && len(s.players) >= 2 {
					s.publisherPriorityQueue(&event{
						eventType: StartGame,
					})
				}
			}
		}
	}
}
