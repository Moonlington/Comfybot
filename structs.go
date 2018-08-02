package main

type imageCommand struct {
	Cmd    string   `json:"cmd"`
	Images []string `json:"images"`
}

type config struct {
	Token          string         `json:"token"`
	Cmds           []imageCommand `json:"cmds"`
	Votingmessages []string       `json:"votingmessages"`
}
