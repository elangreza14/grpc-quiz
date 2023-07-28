// Package usecase ..
package usecase

import (
	"time"
)

type (
	// State ...
	State int

	// GameState ...
	GameState struct {
		State   State
		Payload any
	}

	// GamePlay is ...
	GamePlay struct {
		players          map[string]int
		state            State
		internalStream   chan *GameState
		externalStream   chan string
		questions        map[string]bool
		timePerRound     time.Duration
		expectedQuestion string
		expectedAnswer   bool
	}

	// SubmitAnswerPayload ...
	SubmitAnswerPayload struct {
		name   string
		answer bool
	}

	// QuestionPayload ...
	QuestionPayload struct {
		question string
		answer   bool
	}
)

const (
	// Waiting is state when waiting all the players
	Waiting State = iota
	// Started is state when game is started
	Started
	// SetQuestion is state when game is started
	SetQuestion
	// Announcement is state when game is started
	Announcement
	//  AnswerQuestion is state when game is finished
	AnswerQuestion
	//  Finish is state when game is finished
	Finish
)

// NewGamePlay is ...
func NewGamePlay() *GamePlay {
	Questions := map[string]bool{
		"1 + 1 = 2":  true,
		"1 - 1 = -1": false,
		"1 * 0 = 0":  true,
	}
	g := &GamePlay{
		players:          map[string]int{},
		state:            Waiting,
		internalStream:   make(chan *GameState),
		externalStream:   make(chan string),
		questions:        Questions,
		timePerRound:     3 * time.Second,
		expectedQuestion: "",
		expectedAnswer:   true,
	}
	go g.listenInternalStream()
	go g.listenQuestion()

	return g
}

// Start ...
func (g *GamePlay) Start() {
	g.internalStream <- &GameState{
		State: Started,
	}
}

func (g *GamePlay) listenInternalStream() {
	for res := range g.internalStream {
		switch res.State {
		case Started:
			g.externalStream <- "game started"
			g.state = Started
		case SetQuestion:
			payload := res.Payload.(QuestionPayload)
			g.expectedAnswer = payload.answer
			g.expectedQuestion = payload.question
			g.externalStream <- g.expectedQuestion
		case AnswerQuestion:
			payload := res.Payload.(SubmitAnswerPayload)
			if payload.answer == g.expectedAnswer {
				g.players[payload.name]++
			}
		case Finish:
			g.externalStream <- "game finished"
			g.state = Finish
		}
	}
}

// SubmitAnswer ...
func (g *GamePlay) SubmitAnswer(name string, answer bool) {
	if g.state != Started {
		return
	}

	_, ok := g.players[name]
	if !ok {
		return
	}

	g.internalStream <- &GameState{
		State: AnswerQuestion,
		Payload: SubmitAnswerPayload{
			name:   name,
			answer: answer,
		},
	}
}

// AddPlayer ...
func (g *GamePlay) AddPlayer(name string) {
	g.players[name] = 0
}

// ListenStream ...
func (g *GamePlay) ListenStream() <-chan string {
	return g.externalStream
}

func (g *GamePlay) listenQuestion() {
	for {
		if g.state != Started {
			continue
		}

		for i, v := range g.questions {
			g.internalStream <- &GameState{
				State: SetQuestion,
				Payload: QuestionPayload{
					question: i,
					answer:   v,
				},
			}

			time.Sleep(g.timePerRound)
		}

		g.internalStream <- &GameState{
			State: Finish,
		}
		return
	}
}
