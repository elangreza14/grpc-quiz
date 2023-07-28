package usecase

import (
	"context"
	"fmt"
	"sync"

	quiz "github.com/elangreza14/grpc-quiz/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type (
	eventType int

	// Event is ...
	Event struct {
		EventType eventType
		Payload   any
	}

	// Room is default structure for creating communication
	Room struct {
		// players  map[string]chan *quiz.StreamResponse
		players sync.Map
		queue   chan *Event
		// State   State
		Game *GamePlay
	}
)

const (
	//  InsertPlayer is event for inserting players to players
	InsertPlayer eventType = iota
	//  Broadcast is event for broadcast all the player
	Broadcast
	//  StartGame is event for start the game
	StartGame
)

// NewRoom is
func NewRoom() *Room {
	return &Room{
		players: sync.Map{},
		queue:   make(chan *Event, 100),
		Game:    NewGamePlay(),
	}
}

// PublishQueue is ...
func (r *Room) PublishQueue(evt *Event) {
	r.queue <- evt
}

// ListenQueue is ...
func (r *Room) ListenQueue(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return
		case gameRes := <-r.Game.ListenStream():
			switch gameRes.State {
			case OnProgress:
				if gameRes.payload != nil {
					fmt.Println(gameRes.payload)
				}
			case Done:
				fmt.Println("game finished")
				r.ShutdownClient()
			default:
			}
		case evt := <-r.queue:
			switch evt.EventType {
			case InsertPlayer:
				// initialize the player
				player := evt.Payload.(string)
				r.players.Store(player, make(chan *quiz.StreamResponse, 100))
				r.Game.AddPlayer(player)
				fmt.Printf("player %s joined. total %d players \n", player, r.TotalPlayer())
			case StartGame:
				// r.State = Started
				r.BroadcastToPlayer("game started")
				r.Game.Start()
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
	r.players.Range(func(key, value any) bool {
		ch, okChan := value.(chan *quiz.StreamResponse)
		if okChan {
			ch <- &quiz.StreamResponse{
				Timestamp: timestamppb.Now(),
				Event: &quiz.StreamResponse_ServerAnnouncement{
					ServerAnnouncement: &quiz.Message{
						Message: msg,
					},
				},
			}
		}

		return true
	})
}

// ShutdownClient is ...
func (r *Room) ShutdownClient() {
	r.players.Range(func(key, value any) bool {
		ch, okChan := value.(chan *quiz.StreamResponse)
		if okChan {
			ch <- &quiz.StreamResponse{
				Timestamp: timestamppb.Now(),
				Event: &quiz.StreamResponse_ServerShutdown{
					ServerShutdown: &quiz.Shutdown{},
				},
			}
		}

		return true
	})
}

// GetPlayerDetail is ...
func (r *Room) GetPlayerDetail(player string) (chan *quiz.StreamResponse, bool) {
	res, ok := r.players.Load(player)
	if ok {
		ch, okChan := res.(chan *quiz.StreamResponse)
		if okChan {
			return ch, true
		}
	}

	return nil, false
}

// RemovePlayer is ...
func (r *Room) RemovePlayer(player string) {
	r.players.Delete(player)
}

// TotalPlayer is ...
func (r *Room) TotalPlayer() int {
	total := 0
	r.players.Range(func(_, _ any) bool {
		total++
		return true
	})

	return total
}

// GetState is ...
// func (r *Room) GetState() State {
// 	return r.State
// }
