# Grpc-Quiz
Grpc-Quiz is a command-based quiz with real-time communication. built with *golang* and *grpc*

## run as server

to run as server, enter this line into your terminal
```bash
❯ make server
```

yoi will receive 
```bash
Waiting players to join. press (Y) to start
```

to start the game you can press (Y)

## run as client

to run as client, enter this line into your terminal 
```bash
❯ make client  
insert your name... John
```

you will receive 
```bash
message:"hi John, welcome to the game"
```

## Game Play
This grpc quiz is hosted a server to run the quiz. The Server is streaming a question to each member, and each member must be answer within default durations.

Grpc-quiz have a default 3 rounds. Each Rounds have a 10 seconds timeout for player to answer. 

```yaml
durationPerRound: 10 // in second
questions:
    -   question:   "1 + 1 = 2",
        answer:     true
    -   question:   "1 - 1 = -1",
        answer:     false
    -   question:   "1 * 0 = 0",
        answer:     true
```

If all the players answer the question within defined timeout, the round will be change. If all the round is passed the quiz will be ended.

If the quiz is finished, the server will receive the total points for each player in ascending order.

## Example game

server will run the default config. The Player will be 2 players. John and Alex.

on the server side will be like this

```bash
❯ make server
go run cmd/quiz/main.go
Waiting players to join. press (Y) to start
player John joined. total 1 players 
player Alex joined. total 2 players 
y
round 1: 1 + 1 = 2
=== current point ===
player: John point 0
player: Alex point 0
round 2: 1 - 1 = -1
=== current point ===
player: John point 1
player: Alex point 1
round 3: 1 * 0 = 0
=== current point ===
player: John point 2
player: Alex point 2
=== final point ===
player: John point 3
player: Alex point 3
shutting down the server
player John left. total 1 players 
player Alex left. total 0 players
```

on John side will be like this
```bash
❯ make client
insert your name... John
message:"hi John, welcome to the game"
game started
round 1: 1 + 1 = 2
y
round 2: 1 - 1 = -1
n
round 3: 1 * 0 = 0
y
game finished
server shuting down
```

on Alex side will be like this
```bash
❯ make client
insert your name... Alex
message:"hi Alex, welcome to the game"
game started
round 1: 1 + 1 = 2
y
round 2: 1 - 1 = -1
n
round 3: 1 * 0 = 0
y
game finished
server shuting down
```