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
		players  sync.Map
		queue    chan *Event
		Started  bool
		Game     *GamePlay
		PowerOff chan bool
	}

	// BroadcastPersonalPayload is ...
	BroadcastPersonalPayload struct {
		Name, Message string
	}
)

const (
	//  InsertPlayer is event for inserting players to players
	InsertPlayer eventType = iota
	//  Broadcast is event for broadcast all the player
	Broadcast
	//  BroadcastPersonal is event for broadcast for specific the player
	BroadcastPersonal
	//  StartGame is event for start the game
	StartGame
	//  SubmitAnswer is event for submit the answer
	SubmitAnswer
)

// NewRoom is
func NewRoom() *Room {
	return &Room{
		players:  sync.Map{},
		queue:    make(chan *Event, 100),
		Game:     NewGamePlay(),
		PowerOff: make(chan bool),
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
					r.BroadcastToAllPlayer(gameRes.payload.(string))
				}
			case Done:
				r.BroadcastToAllPlayer("game finished")
				r.PowerOff <- true
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
				r.BroadcastToAllPlayer("game started")
				r.Game.Start()
				r.Started = true
			case Broadcast:
				r.BroadcastToAllPlayer(evt.Payload.(string))
			case BroadcastPersonal:
				r.BroadcastToSpecificPlayer(evt.Payload.(BroadcastPersonalPayload))
			case SubmitAnswer:
				r.Game.SubmitAnswer(evt.Payload.(SubmitAnswerPayload))
			default:
				// no operation
			}
		}
	}
}

// BroadcastToAllPlayer is ...
func (r *Room) BroadcastToAllPlayer(msg string, playerException ...string) {
	r.players.Range(func(key, value any) bool {
		for i := 0; i < len(playerException); i++ {
			if playerException[i] == key.(string) {
				return true
			}
		}

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

// BroadcastToSpecificPlayer is ...
func (r *Room) BroadcastToSpecificPlayer(req BroadcastPersonalPayload) {
	msgPlayer, ok := r.players.Load(req.Name)
	if ok {
		ch, okChan := msgPlayer.(chan *quiz.StreamResponse)
		if okChan {
			ch <- &quiz.StreamResponse{
				Timestamp: timestamppb.Now(),
				Event: &quiz.StreamResponse_ServerAnnouncement{
					ServerAnnouncement: &quiz.Message{
						Message: req.Message,
					},
				},
			}
		}
	}
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
	r.Game.RemovePlayer(player)
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

// Done is ...
func (r *Room) Done() <-chan bool {
	return r.PowerOff
}
