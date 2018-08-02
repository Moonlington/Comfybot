package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/Moonlington/discordflo"
)

var ffs *discordflo.FloFloSession
var cfg *config

func getSettings() {
	var jsonFile *os.File
	if _, err := os.Stat("settings.json"); os.IsNotExist(err) {
		jsonFile, err = os.Create("settings.json")
		if err != nil {
			fmt.Println(err)
		}
	} else {
		jsonFile, err = os.Open("settings.json")
		if err != nil {
			fmt.Println(err)
		}
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	json.Unmarshal(byteValue, &cfg)
}

func updateSettings() {
	newFile, err := json.Marshal(cfg)

	jsonFile, err := os.Create("settings.json")
	if err != nil {
		fmt.Println(err)
	}

	defer jsonFile.Close()

	jsonFile.Write(newFile)
	getSettings()
}

func main() {
	getSettings()
	ffs = initializeBot(cfg.Token)
	updateSettings()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	ffs.Close()
}
