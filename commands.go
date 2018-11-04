package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Moonlington/discordflo"
	"github.com/bwmarrin/discordgo"
)

const prefix = "comf."
const channel = "474556628625260544"

func addImage(name, url, image string) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}

	file := resp.Body
	defer file.Close()

	imagefile, err := os.Create("images/" + image)
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
			cfg.Cmds[i].Images = append(cfg.Cmds[i].Images, image)
			updateSettings()
			return
		}
	}

	cfg.Cmds = append(cfg.Cmds, imageCommand{Cmd: name, Images: []string{image}})
	updateSettings()
}

func democraticLoop() {
	for {
		msgl, err := ffs.ChannelMessages(channel, 100, "", "", "")
		if err != nil {
			return
		}

		for _, msg := range msgl {
			if strings.HasPrefix(msg.Content, "VOTE:") {
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
					commandname := regexp.MustCompile("command `(.+)`").FindStringSubmatch(msg.Content)[1]
					ffs.ChannelMessageEdit(channel, msg.ID, fmt.Sprintf("DONE: Image added to `%s`", commandname))
					addImage(commandname, msg.Attachments[0].URL, msg.Attachments[0].Filename)
					continue
				} else if no > (yes+no)/2 {
					ffs.ChannelMessageEdit(channel, msg.ID, "DONE: Image got denied by democracy")
					continue
				}
			}
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

	msg, err := ffs.ChannelFileSendWithMessage(
		channel,
		fmt.Sprintf("VOTE: <@%s> wants to add this image to the command `%s` (Voting will end in 5 minutes...)", ctx.Mess.Author.ID, name),
		ctx.Mess.ID+filepath.Ext(image),
		file)
	if err != nil {
		return
	}

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

	if strings.ToLower(imagename) == "addimage" || strings.ToLower(imagename) == "help" || strings.ToLower(imagename) == "createhungergames" {
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

	ffs.AddCommand("Main", discordflo.NewCommand(
		"createHungerGames",
		"Picks random people from the server to participate",
		"[24, 36, 48] [list of names to include...]",
		"",
		createHungerGames,
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
