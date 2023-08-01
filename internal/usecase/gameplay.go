// Package usecase ..
package usecase

import (
	"fmt"
	"sort"
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

	// GameState is ...
	GameState struct {
		State
		payload any
	}

	// GamePlay is ...
	GamePlay struct {
		players        map[string]int
		state          State
		internalStream chan *internalAction
		externalStream chan *GameState
		questionStream chan *QuestionPayload
		stopStream     chan bool
		questions      []QuestionPayload
		timePerRound   time.Duration
		expected       QuestionPayload
		round          int
	}

	// SubmitAnswerPayload ...
	SubmitAnswerPayload struct {
		Name   string
		Answer bool
	}

	// QuestionPayload ...
	QuestionPayload struct {
		question      string
		answer        bool
		block         chan bool
		playerRetries map[string]int
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
	Questions := []QuestionPayload{
		{
			question:      "1 + 1 = 2",
			answer:        true,
			block:         make(chan bool),
			playerRetries: map[string]int{},
		},
		{
			question:      "1 - 1 = -1",
			answer:        false,
			block:         make(chan bool),
			playerRetries: map[string]int{},
		},
		{
			question:      "1 * 0 = 0",
			answer:        true,
			block:         make(chan bool),
			playerRetries: map[string]int{},
		},
	}

	g := &GamePlay{
		players:        map[string]int{},
		state:          Waiting,
		internalStream: make(chan *internalAction),
		externalStream: make(chan *GameState),
		questionStream: make(chan *QuestionPayload, len(Questions)),
		stopStream:     make(chan bool),
		questions:      Questions,
		timePerRound:   10 * time.Second,
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
			g.externalStream <- &GameState{
				State: OnProgress,
			}
			g.state = OnProgress
			for i := 0; i < len(g.questions); i++ {
				q := g.questions[i]
				g.questionStream <- &q

			}
		case setQuestion:
			g.expected = res.payload.(QuestionPayload)
			g.externalStream <- &GameState{
				State:   OnProgress,
				payload: fmt.Sprintf("round %d: %s", g.round+1, g.expected.question),
			}
		case answerQuestion:
			payload := res.payload.(SubmitAnswerPayload)
			if payload.Answer == g.expected.answer {
				g.players[payload.Name]++
			}

			// calculate the retry
			if _, ok := g.questions[g.round].playerRetries[payload.Name]; ok {
				g.questions[g.round].playerRetries[payload.Name]++
			} else {
				g.questions[g.round].playerRetries[payload.Name] = 0
			}

			if len(g.questions[g.round].playerRetries) == len(g.players) {
				g.expected.block <- true
			}

		case finish:
			g.externalStream <- &GameState{
				State: Done,
			}
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
		g.round = i
		question := <-g.questionStream

		g.setAction(setQuestion, *question)

		select {
		case <-question.block:
		case <-time.After(g.timePerRound):
		}

		// fmt.Printf("total answer %v\n", len(g.questions[i].playerRetries))
	}

	g.setAction(finish, nil)
}

// SubmitAnswer ...
func (g *GamePlay) SubmitAnswer(answer SubmitAnswerPayload) {
	if g.state != OnProgress {
		return
	}

	_, ok := g.players[answer.Name]
	if !ok {
		return
	}

	g.setAction(answerQuestion, answer)
}

// AddPlayer ...
func (g *GamePlay) AddPlayer(name string) { g.players[name] = 0 }

// RemovePlayer ...
func (g *GamePlay) RemovePlayer(name string) { delete(g.players, name) }

// ListenStream ...
func (g *GamePlay) ListenStream() <-chan *GameState { return g.externalStream }

// GetState ...
func (g GamePlay) GetState() {
	players := []struct {
		name  string
		point int
	}{}

	for j, val := range g.players {
		players = append(players, struct {
			name  string
			point int
		}{
			name:  j,
			point: val,
		})
	}

	sort.Slice(players, func(i, j int) bool {
		return players[i].point > players[j].point
	})

	fmt.Println("=== final point ===")

	for i := 0; i < len(players); i++ {
		fmt.Printf("player: %v point %v\n", players[i].name, players[i].point)
	}
}
