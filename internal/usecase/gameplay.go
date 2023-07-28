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
		players        map[string]int
		state          State
		internalStream chan *GameState
		externalStream chan string
		questionStream chan *QuestionPayload
		questions      map[string]bool
		timePerRound   time.Duration
		expected       QuestionPayload
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
	// Wait is state when waiting all the players
	Wait State = iota
	// Start is state when game is started
	Start
	// SetQuestion is state when game is started
	SetQuestion
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
		players:        map[string]int{},
		state:          Wait,
		internalStream: make(chan *GameState),
		externalStream: make(chan string),
		questionStream: make(chan *QuestionPayload, len(Questions)),
		questions:      Questions,
		timePerRound:   3 * time.Second,
	}

	go g.listenInternalStream()
	go g.listenQuestion()

	return g
}

func (g *GamePlay) listenInternalStream() {
	for res := range g.internalStream {
		switch res.State {
		case Start:
			g.externalStream <- "game started"
			g.state = Start
			for question, answer := range g.questions {
				g.questionStream <- &QuestionPayload{
					question: question,
					answer:   answer,
				}
			}
		case SetQuestion:
			g.expected = res.Payload.(QuestionPayload)
			g.externalStream <- g.expected.question
		case AnswerQuestion:
			payload := res.Payload.(SubmitAnswerPayload)
			if payload.answer == g.expected.answer {
				g.players[payload.name]++
			}
		case Finish:
			g.externalStream <- "game finished"
			g.state = Finish
		}
	}
}

// Start ...
func (g *GamePlay) Start() {
	g.internalStream <- &GameState{
		State: Start,
	}
}

func (g *GamePlay) listenQuestion() {
	for i := 0; i < len(g.questions); i++ {
		v := <-g.questionStream

		g.internalStream <- &GameState{
			State:   SetQuestion,
			Payload: *v,
		}

		time.Sleep(g.timePerRound)
	}

	g.internalStream <- &GameState{
		State: Finish,
	}
}

// SubmitAnswer ...
func (g *GamePlay) SubmitAnswer(name string, answer bool) {
	if g.state != Start {
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
func (g *GamePlay) AddPlayer(name string) { g.players[name] = 0 }

// ListenStream ...
func (g *GamePlay) ListenStream() <-chan string { return g.externalStream }

// GetStats ...
func (g *GamePlay) GetStats() string {
	return "g.externalStream"
}
