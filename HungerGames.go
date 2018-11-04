package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/Moonlington/discordflo"
)

func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func find(a []*discordgo.User, x *discordgo.User) int {
	for i, n := range a {
		if x.ID == n.ID {
			return i
		}
	}
	return -1
}

func createHungerGames(ctx *discordflo.Context) {
	if len(ctx.Args) != 0 && ctx.Args[0] != "24" && ctx.Args[0] != "36" && ctx.Args[0] != "48" {
		ctx.SendMessage("The first argument can't be anything but 24, 36 or 48")
		return
	}

	if len(ctx.Args) == 0 {
		ctx.Args = append(ctx.Args, "24")
	}

	form := url.Values{
		"ChangeAll":    {"028"},
		"existinglogo": {"00"},
		"logourl":      {"https://flore.is-a-good-waifu.com/8f7c8c.png"},
		"seasonname":   {"The Comfy Games"},
	}

	ctx.SendMessage("Creating game...")

	var idxs []int

	serverMembers, err := ctx.GetAllUsers()
	if err != nil {
		return
	}

	var manual []string
	input := strings.Join(ctx.Args[1:], " ")
	if input != "" {
		manual = ctx.Args[1:]
		for _, name := range manual {
			user, err := ctx.GetUser(name, ctx.Guild.ID)
			if err != nil {
				return
			}
			idx := find(serverMembers, user)
			idxs = append(idxs, idx)
		}
	}

	playeramount, _ := strconv.Atoi(ctx.Args[0])
	for i := 0; i < playeramount-len(manual); i++ {
		r := rand.Intn(len(serverMembers))
		for contains(idxs, r) {
			r = rand.Intn(len(serverMembers))
		}
		idxs = append(idxs, r)
	}

	for i, j := range idxs {
		form.Add(fmt.Sprintf("cusTribute%02d", i+1), serverMembers[j].Username)
		form.Add(fmt.Sprintf("cusTribute%02dcustom", i+1), "000")
		form.Add(fmt.Sprintf("cusTribute%02dgender", i+1), strconv.Itoa(rand.Intn(2)))
		form.Add(fmt.Sprintf("cusTribute%02dimg", i+1), serverMembers[j].AvatarURL(""))
		form.Add(fmt.Sprintf("cusTribute%02dimgBW", i+1), "BW")
		form.Add(fmt.Sprintf("cusTribute%02dnickname", i+1), serverMembers[j].Username)
	}

	body := bytes.NewBufferString(form.Encode())

	cookieJar, _ := cookiejar.New(nil)

	c := &http.Client{
		Jar: cookieJar,
	}

	_, err = c.Get("http://brantsteele.net/hungergames/disclaimer.php")
	if err != nil {
		return
	}

	_, err = c.Get("http://brantsteele.net/hungergames/agree.php")
	if err != nil {
		return
	}

	_, err = c.Get("http://brantsteele.net/hungergames/ChangeTributes-" + ctx.Args[0] + ".php")
	if err != nil {
		return
	}

	_, err = c.Post("http://brantsteele.net/hungergames/personalize-"+ctx.Args[0]+".php", "application/x-www-form-urlencoded", body)
	if err != nil {
		return
	}

	resp, err := c.Get("http://brantsteele.net/hungergames/save.php")
	if err != nil {
		return
	}

	b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return
	}
	link := regexp.MustCompile(`http:\/\/brantsteele\.net\/hungergames\/r\.php\?c=(.{8})`).FindStringSubmatch(string(b))[0]
	fmt.Println(link)
	ctx.SendMessage(link)
}
