package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Moonlington/discordflo"
	"github.com/bwmarrin/discordgo"
)

const prefix = "comf."
const channel = "474556628625260544"

var tmp []int

func checkExists(fileName string) string {
	if _, err := os.Stat("images/" + fileName); !os.IsNotExist(err) {
		fileName = "a" + fileName
		fileName = checkExists(fileName)
	}
	return fileName
}

func addImage(name, image string) {
	resp, err := http.Get(image)
	if err != nil {
		return
	}

	file := resp.Body
	defer file.Close()

	tokens := strings.Split(image, "/")
	fileName := tokens[len(tokens)-1]

	checkExists(fileName)

	imagefile, err := os.Create("images/" + fileName)
	if err != nil {
		return
	}

	defer imagefile.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}

	imagefile.Write(bytes)

	for i, cmd := range cfg.Cmds {
		if cmd.Cmd == name {
			cfg.Cmds[i].Images = append(cfg.Cmds[i].Images, fileName)
			updateSettings()
			return
		}
	}

	cfg.Cmds = append(cfg.Cmds, imageCommand{Cmd: name, Images: []string{fileName}})
	updateSettings()
}

func removeIndexes() {
	tmpp := cfg.Votingmessages[:0]
	for index := range tmp {
		for i, p := range cfg.Votingmessages {
			if i != index {
				tmpp = append(tmpp, p)
			}
		}
	}
	cfg.Votingmessages = tmpp[:0]
	updateSettings()
	tmp = []int{}
}

func democraticLoop() {
	for {
		for i, id := range cfg.Votingmessages {
			msgl, err := ffs.ChannelMessages(channel, 1, id, id, id)
			if err != nil || len(msgl) == 0 {
				tmp = append(tmp, i)
				continue
			}
			msg := msgl[0]

			timeposted, _ := msg.Timestamp.Parse()

			if timeposted.Add(time.Minute * 5).After(time.Now()) {
				continue
			}

			reactions := msg.Reactions

			var yes, no int

			for _, r := range reactions {
				if r.Emoji.Name == "✅" {
					yes = r.Count
				}
				if r.Emoji.Name == "⛔" {
					no = r.Count
				}
			}

			if yes > (yes+no)/2 {
				ffs.ChannelMessageEdit(channel, id, "We did it people, democracy works! Image added to the command.")
				addImage(regexp.MustCompile("command `(.+)`").FindStringSubmatch(msg.Content)[1], msg.Attachments[0].URL)
				tmp = append(tmp, i)
				continue
			} else if no > (yes+no)/2 {
				ffs.ChannelMessageEdit(channel, id, "The image did not get added.")
				tmp = append(tmp, i)
				continue
			}
		}
		if len(tmp) != 0 {
			removeIndexes()
		}
		time.Sleep(10 * time.Second)
	}
}

func isValidURL(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}
	return true
}

func createDemocracy(ctx *discordflo.Context, name, image string) {
	resp, err := http.Get(image)
	if err != nil {
		return
	}

	file := resp.Body
	defer file.Close()

	tokens := strings.Split(image, "/")
	fileName := tokens[len(tokens)-1]

	msg, err := ffs.ChannelFileSendWithMessage(channel, fmt.Sprintf("<@%s> wants to add this image to the command `%s` (Voting will end in 5 minutes...)", ctx.Mess.Author.ID, name), fileName, file)
	if err != nil {
		return
	}

	cfg.Votingmessages = append(cfg.Votingmessages, msg.ID)
	updateSettings()

	ffs.MessageReactionAdd(msg.ChannelID, msg.ID, "✅")
	ffs.MessageReactionAdd(msg.ChannelID, msg.ID, "⛔")
}

func addImageCommand(ctx *discordflo.Context) {
	if len(ctx.Args) < 2 {
		ctx.SendMessage("Not enough arguments supplied.")
		return
	}

	imagename := ctx.Args[0]
	imagelink := ctx.Args[1]

	if strings.ToLower(imagename) == "addimage" || strings.ToLower(imagename) == "help" {
		ctx.SendMessage("Don't attempt it.")
		return
	}

	if !isValidURL(imagelink) {
		ctx.SendMessage("The link you provided is not an actual URL")
		return
	}

	createDemocracy(ctx, imagename, imagelink)
}

func handleComfy(ctx *discordflo.Context) {
	for _, cmd := range cfg.Cmds {
		if cmd.Cmd == ctx.Invoked {
			file, err := os.Open("images/" + cmd.Images[rand.Intn(len(cmd.Images))])
			if err != nil {
				return
			}
			ffs.ChannelFileSend(ctx.Mess.ChannelID, file.Name(), file)
			return
		}
	}
}

func initializeBot(token string) (ffs *discordflo.FloFloSession) {
	rand.Seed(time.Now().Unix())
	// Create a new Discord session using the provided bot token.
	ffs, err := discordflo.New("Bot "+token, prefix, false)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Change the handler to handle comf
	ffs.ChangeMessageHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore all messages created by other users
		if ffs.Bot {
			if m.Author.ID == s.State.User.ID {
				return
			}
		} else {
			if m.Author.ID != s.State.User.ID {
				return
			}
		}

		if len(m.Content) > 0 && (strings.HasPrefix(strings.ToLower(m.Content), strings.ToLower(ffs.Prefix))) {
			// Setting values for the commands
			var ctx *discordflo.Context
			args := strings.Fields(m.Content[len(ffs.Prefix):])
			invoked := args[0]
			args = args[1:]
			argstr := m.Content[len(ffs.Prefix)+len(invoked):]
			if argstr != "" {
				argstr = argstr[1:]
			}
			channel, err := s.State.Channel(m.ChannelID)
			if err != nil {
				channel, _ = s.State.PrivateChannel(m.ChannelID)
				ctx = &discordflo.Context{Invoked: invoked, Argstr: argstr, Args: args, Channel: channel, Guild: nil, Mess: m, Sess: ffs}
			} else {
				guild, _ := s.State.Guild(channel.GuildID)
				ctx = &discordflo.Context{Invoked: invoked, Argstr: argstr, Args: args, Channel: channel, Guild: guild, Mess: m, Sess: ffs}
			}
			go ffs.HandleCommands(ctx)
			go handleComfy(ctx)
		}
	})

	ffs.AddCommand("Main", discordflo.NewCommand(
		"addimage",
		"Adds an image to a command",
		"<command name> <link to image>",
		"Requires both arguments to work.",
		addImageCommand,
	))

	// Open a websocket connection to Discord and begin listening.
	err = ffs.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	go democraticLoop()

	return ffs
}
