// Package usecase ..
package usecase

import (
	"time"
)

type (
	// State ...
	State int

	// action ...
	action int

	internalAction struct {
		action
		payload any
	}

	// GamePlay is ...
	GamePlay struct {
		players        map[string]int
		state          State
		internalStream chan *internalAction
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
	start action = iota
	setQuestion
	answerQuestion
	finish

	// Waiting is
	Waiting State = iota
	// OnProgress is
	OnProgress
	// Done is
	Done
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
		state:          Waiting,
		internalStream: make(chan *internalAction),
		externalStream: make(chan string),
		questionStream: make(chan *QuestionPayload, len(Questions)),
		questions:      Questions,
		timePerRound:   3 * time.Second,
	}

	go g.listenInternalStream()
	go g.listenQuestion()

	return g
}

func (g *GamePlay) setAction(action action, payload any) {
	g.internalStream <- &internalAction{
		action:  action,
		payload: payload,
	}
}

func (g *GamePlay) listenInternalStream() {
	for res := range g.internalStream {
		switch res.action {
		case start:
			g.externalStream <- "game started"
			g.state = OnProgress
			for question, answer := range g.questions {
				g.questionStream <- &QuestionPayload{question, answer}
			}
		case setQuestion:
			g.expected = res.payload.(QuestionPayload)
			g.externalStream <- g.expected.question
		case answerQuestion:
			payload := res.payload.(SubmitAnswerPayload)
			if payload.answer == g.expected.answer {
				g.players[payload.name]++
			}
		case finish:
			g.externalStream <- "game finished"
			g.state = Done
		}
	}
}

// Start ...
func (g *GamePlay) Start() {
	g.setAction(start, nil)
}

func (g *GamePlay) listenQuestion() {
	for i := 0; i < len(g.questions); i++ {
		question := <-g.questionStream

		g.setAction(setQuestion, *question)

		time.Sleep(g.timePerRound)
	}

	g.setAction(finish, nil)
}

// SubmitAnswer ...
func (g *GamePlay) SubmitAnswer(name string, answer bool) {
	if g.state != OnProgress {
		return
	}

	_, ok := g.players[name]
	if !ok {
		return
	}

	g.setAction(answerQuestion, SubmitAnswerPayload{name, answer})
}

// AddPlayer ...
func (g *GamePlay) AddPlayer(name string) { g.players[name] = 0 }

// ListenStream ...
func (g *GamePlay) ListenStream() <-chan string { return g.externalStream }

// GetStats ...
func (g *GamePlay) GetStats() string {
	return "g.externalStream"
}
