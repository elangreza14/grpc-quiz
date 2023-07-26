package usecase

import (
	"context"
	"fmt"

	quiz "github.com/elangreza14/grpc-quiz/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type (
	eventType int
	playState int

	Event struct {
		EventType eventType
		Payload   any
	}

	// Room is default structure for creating communication
	Room struct {
		players map[string]chan *quiz.StreamResponse
		queue   chan *Event
		State   playState
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

func NewRoom() *Room {
	return &Room{
		players: map[string]chan *quiz.StreamResponse{},
		queue:   make(chan *Event, 100),
		State:   Waiting,
	}
}

// PublishQueue is ...
func (r *Room) PublishQueue(evt *Event) {
	r.queue <- evt
}

// ListenQueue is ...
func (r *Room) ListenQueue(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-r.queue:
			switch evt.EventType {
			case InsertPlayer:
				// initialize the player
				r.players[evt.Payload.(string)] = make(chan *quiz.StreamResponse, 100)
				fmt.Printf("player %s joined. total %d players \n", evt.Payload, len(r.players))
			case StartGame:
				r.State = Started
				r.BroadcastToPlayer("game started")
			case Broadcast:
				r.BroadcastToPlayer(evt.Payload.(string))
			default:
				// no operation
			}
		}
	}
}

// BroadcastToPlayer is ...
func (r *Room) BroadcastToPlayer(msg string) {
	for i := range r.players {
		r.players[i] <- &quiz.StreamResponse{
			Timestamp: timestamppb.Now(),
			Event: &quiz.StreamResponse_ServerAnnouncement{
				ServerAnnouncement: &quiz.StreamResponse_Message{
					Message: msg,
				},
			},
		}
	}
}

// BroadcastToShutdown is ...
func (r *Room) BroadcastToShutdown() {
	for i := range r.players {
		r.players[i] <- &quiz.StreamResponse{
			Timestamp: timestamppb.Now(),
			Event: &quiz.StreamResponse_ServerShutdown{
				ServerShutdown: &quiz.StreamResponse_Shutdown{},
			},
		}
	}
}

// GetPlayerDetail is ...
func (r *Room) GetPlayerDetail(player string) (chan *quiz.StreamResponse, bool) {
	res, ok := r.players[player]
	return res, ok
}

// RemovePlayer is ...
func (r *Room) RemovePlayer(name string) {
	delete(r.players, name)
}

// TotalPlayer is ...
func (r Room) TotalPlayer() int {
	return len(r.players)
}
