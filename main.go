package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var filePath map[string]string

func main() {

	filePath = make(map[string]string)

	var (
		Token = flag.String("t", "", "Discord Authentication Token")
	)
	flag.Parse()

	if *Token == "" {
		log.Println("No authentication token supplied, please run bot with flag \"-t TOKEN\"")
	} else {
		dg, e := discordgo.New("Bot " + *Token)
		if err(e, "") {
			return
		}
		dg.AddHandler(onGuildCreate)    //Add a handler that activates whenever the bot restarts, or is added to a guild.
		dg.AddHandler(voiceStateUpdate) //Add a handler that activates whenever a user's voice state changes.
		dg.AddHandler(messageCreate)    //Add a handler that activates whenever a message is posted in a channel.
		e = dg.Open()
		if err(e, "") {
			return
		}
		log.Println("Bot started successfully")
	}
	<-make(chan struct{})
	return
}

//Initialize file structure, or ghostbust if already initialized.
func onGuildCreate(s *discordgo.Session, gc *discordgo.GuildCreate) {
	if gc.Guild.Unavailable == true {
		return
	} else {
		setPath(gc.Guild)
		if _, e := os.Stat(filePath["VOICEPATH"]); os.IsNotExist(e) {
			fileInit(gc.Guild)
			for _, vs := range gc.Guild.VoiceStates {
				voiceStateToData(s, gc.Guild, vs, true)
			}
		} else {
			ghostbusting(s, gc.Guild)
		}
		if _, e := os.Stat(filePath["TEXTPATH"]); os.IsNotExist(e) {
			fileInit(gc.Guild)
		} else {
			backfillMessages(s, gc.Guild)
		}
	}
}

//Write voice state data to log when a user's voice state changes.
func voiceStateUpdate(s *discordgo.Session, vsu *discordgo.VoiceStateUpdate) {
	g, e := s.Guild(vsu.GuildID)
	if err(e, "") {
		return
	}
	voiceStateToData(s, g, vsu.VoiceState, false)
}

//Write message data to log when a user creates a message in a channel.
func messageCreate(s *discordgo.Session, mc *discordgo.MessageCreate) {
	g := getGuildFromChannel(s, mc.ChannelID)
	messageToData(s, g, mc.Message)
}

//Initialize file structure.
func fileInit(g *discordgo.Guild) {
	setPath(g)
	if _, e := os.Stat(filePath["GUILDPATH"]); os.IsNotExist(e) {
		os.MkdirAll(filePath["GUILDPATH"], os.ModeDir)
		log.Println("")
		log.Println("No working directory found, creating one...")
		if _, e := os.Stat(filePath["GUILDPATH"]); os.IsNotExist(e) {
			log.Println("Directory creation failed")
			return
		} else {
			log.Println("Directory creation successful")
		}
	}
	if _, e := os.Stat(filePath["VOICEPATH"]); os.IsNotExist(e) {
		os.Create(filePath["VOICEPATH"])
		ioutil.WriteFile(filePath["VOICEPATH"], []byte("{\n\t\"Array\":[\n\t\t{\n\t\t\t\"Username\": \"placeholder\",\n\t\t\t\"UserID\": \"placeholder\",\n\t\t\t\"Action\": \"placeholder\",\n\t\t\t\"Channel\": \"placeholder\",\n\t\t\t\"ChannelID\": \"placeholder\",\n\t\t\t\"Time\": \"placeholder\",\n\t\t\t\"Duration\": \"placeholder\"\n\t\t}\n\t]\n}"), os.ModePerm)
		log.Println("")
		log.Println("No '" + filePath["VOICEPATH"] + "' file found, creating one...")
		if _, e := os.Stat(filePath["VOICEPATH"]); os.IsNotExist(e) {
			log.Println("Voice data file creation failed")
			return
		} else {
			log.Println("Voice data file creation successful")
		}
	}
	if _, e := os.Stat(filePath["TEXTPATH"]); os.IsNotExist(e) {
		os.Create(filePath["TEXTPATH"])
		ioutil.WriteFile(filePath["TEXTPATH"], []byte("{\n\t\"Array\":[\n\t\t{\n\t\t\t\"ID\": \"placeholder\",\n\t\t\t\"Username\": \"placeholder\",\n\t\t\t\"UserID\": \"placeholder\",\n\t\t\t\"Type\": \"placeholder\",\n\t\t\t\"Channel\": \"placeholder\",\n\t\t\t\"ChannelID\": \"placeholder\",\n\t\t\t\"Time\": \"placeholder\"\n\t\t}\n\t]\n}"), os.ModePerm)
		log.Println("")
		log.Println("No '" + filePath["TEXTPATH"] + "' file found, creating one...")
		if _, e := os.Stat(filePath["TEXTPATH"]); os.IsNotExist(e) {
			log.Println("Message data file creation failed")
			return
		} else {
			log.Println("Message data file creation successful")
		}
	}
}

//Sets the file path for the current guild.
func setPath(g *discordgo.Guild) {
	guildPath := "statbot/" + strings.ToLower(strings.Replace(g.Name, " ", "", -1))
	filePath["GUILDPATH"] = guildPath
	filePath["VOICEPATH"] = guildPath + "/voicedata.json"
	filePath["TEXTPATH"] = guildPath + "/messagedata.json"
}
func getGuildFromChannel(s *discordgo.Session, cID string) *discordgo.Guild {
	c, e := s.State.Channel(cID)
	if err(e, "") {
		return nil
	}
	g, e := s.State.Guild(c.GuildID)
	if err(e, "") {
		return nil
	}
	return g
}

//Error handling.
func err(e error, c string) bool {
	if e != nil {
		if c != "" {
			log.Println(c)
		}
		log.Println("")
		log.Println("Error:", e)
		return true
	} else {
		return false
	}
}
