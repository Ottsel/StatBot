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
				writeToVoiceLog(s, gc.Guild, voiceStateToData(s, vs), true)
			}
		} else {
			ghostbusting(s, gc.Guild)
		}
	}
}

//Write voice state data to log when a user's voice state changes.
func voiceStateUpdate(s *discordgo.Session, vsu *discordgo.VoiceStateUpdate) {
	g, e := s.Guild(vsu.GuildID)
	if err(e, "") {
		return
	}
	writeToVoiceLog(s, g, voiceStateToData(s, vsu.VoiceState), false)
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
}

//Sets the file path for the current guild.
func setPath(g *discordgo.Guild) {
	guildPath := "statbot/" + strings.ToLower(strings.Replace(g.Name, " ", "", -1))
	filePath["GUILDPATH"] = guildPath
	filePath["VOICEPATH"] = guildPath + "/voicedata.json"
	filePath["TEXTPATH"] = guildPath + "/textdata.json"
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
