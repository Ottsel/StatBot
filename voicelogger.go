package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type VoiceStateArray struct {
	VoiceStateLog []VoiceStateData `json:"Array"`
}
type VoiceStateData struct {
	Username  string `json:"Username"`
	UserID    string `json:"UserID"`
	Action    string `json:"Action"`
	Channel   string `json:"Channel"`
	ChannelID string `json:"ChannelID"`
	Time      string `json:"Time"`
	Duration  string `json:"Duration"`
}

const (
	logTimeFormat string = "01/02/06 - 15:04:05"
	dateFormat    string = "01-02-06"
)

//Converts a voice state into, much cooler, voice state data.
func voiceStateToData(s *discordgo.Session, g *discordgo.Guild, vs *discordgo.VoiceState, isInit bool) {
	vsd := VoiceStateData{"", vs.UserID, "-", "", vs.ChannelID, time.Now().Format(logTimeFormat), "-"}
	user, e := s.User(vs.UserID)
	if vs.UserID != "" {
		if err(e, "") {
			vsd.Action = "Error"
			writeToVoiceLog(g, vsd)
		}
		vsd.Username = user.Username
	}
	if vs.ChannelID != "" {
		channel, e := s.Channel(vs.ChannelID)
		if err(e, "") {
		} else {
			vsd.Channel = channel.Name
		}
	}

	last := getLastVoiceState(g, vs.UserID)
	if vsd.ChannelID == "" {
		vsd.Action = "Disconnected"
		if last.ChannelID != g.AfkChannelID {
			vsd.Duration = getDuration(vsd.Time, last.Time)
		}
		vsd.ChannelID = last.ChannelID
		vsd.Channel = last.Channel
		writeToVoiceLog(g, vsd)
	} else if vsd.ChannelID == g.AfkChannelID {
		vsd.Action = "AFK"
		writeToVoiceLog(g, vsd)
	} else if isInit || last.Action == "placeholder" {
		vsd.Action = "Joined"
		writeToVoiceLog(g, vsd)
	} else if last.Action == "Disconnected" {
		vsd.Action = "Joined"
		writeToVoiceLog(g, vsd)
	} else if vsd.ChannelID != last.ChannelID {
		vsd.Action = "Changed Channel"
		if last.ChannelID != g.AfkChannelID {
			vsd.Duration = getDuration(vsd.Time, last.Time)
		}
		writeToVoiceLog(g, VoiceStateData{vsd.Username, vsd.UserID, vsd.Action, last.Channel, last.ChannelID, vsd.Time, vsd.Duration})

		vsd.Action = "Joined"
		vsd.Time = time.Now().Format(logTimeFormat)
		vsd.Duration = "-"
		writeToVoiceLog(g, vsd) //Write another entry to log which channel the user moved to.
	}
}

//Writes voice state data to log.
func writeToVoiceLog(g *discordgo.Guild, vsd VoiceStateData) {
	setPath(g)

	//Create a JSON Object based on voice state data.
	newObjectText := "{\n\t\"Array\":[\n\t\t{\n\t\t\t\"Username\": \"" + vsd.Username + "\",\n\t\t\t\"UserID\": \"" + vsd.UserID + "\",\n\t\t\t\"Action\": \"" + vsd.Action + "\",\n\t\t\t\"Channel\": \"" + vsd.Channel + "\",\n\t\t\t\"ChannelID\": \"" + vsd.ChannelID + "\",\n\t\t\t\"Time\": \"" + vsd.Time + "\",\n\t\t\t\"Duration\": \"" + vsd.Duration + "\"\n\t\t},"
	logText, readErr := ioutil.ReadFile(filePath["VOICEPATH"])
	if err(readErr, "Failed to read voice log file") {
		return
	}

	//Replace the beginning of the voice data file with a new object. This way the JSON decoder can read the most recent data first.
	newLog := strings.Replace(string(logText[:]), "{\n\t\"Array\":[", newObjectText, 1)
	writeErr := ioutil.WriteFile(filePath["VOICEPATH"], []byte(newLog), os.ModePerm)
	if err(writeErr, "Failed to store voice state data for user: "+vsd.Username) {
		return
	} else {
		log.Println("")
		log.Println("Stored voice state data for user: " + vsd.Username)
	}
}

//Calculates the difference between the join and disconnect time of a user. Useful for usage metrics.
func getDuration(currentTime string, lastTime string) string {
	current, e := time.Parse(logTimeFormat, currentTime)
	err(e, "Failed to parse \"Time\" value. Surely you didn't mess with the data, did you?")
	last, e := time.Parse(logTimeFormat, lastTime)
	err(e, "Failed to parse \"Time\" value. Surely you didn't mess with the data, did you?")
	duration := current.Sub(last)
	if duration.Seconds() < 60 {
		return "under a minute"
	}
	return duration.String()
}

//Gets the last voice state of the specified user.
func getLastVoiceState(g *discordgo.Guild, userID string) VoiceStateData {
	for _, v := range getVoiceLog(g).VoiceStateLog {
		if v.UserID == userID {
			return v
		}
	}

	return getVoiceLog(g).VoiceStateLog[0] //If no previous voice state data exists for this user, return placeholder data.
}

//Decodes voice log file for the specified guild.
func getVoiceLog(g *discordgo.Guild) VoiceStateArray {
	setPath(g)
	voiceData := VoiceStateArray{}
	if filePath["VOICEPATH"] != "" {
		voiceDataFile, _ := os.Open(filePath["VOICEPATH"])
		decoder := json.NewDecoder(voiceDataFile)
		decodeErr := decoder.Decode(&voiceData)
		if err(decodeErr, "Failed to decode voice data file, renaming corrupt file: \""+filePath["VOICEPATH"]+"_CORRUPT_"+time.Now().Format(dateFormat)+".txt\"") {
			closeErr := voiceDataFile.Close()
			err(closeErr, "")
			renameErr := os.Rename(filePath["VOICEPATH"], filePath["VOICEPATH"]+"_CORRUPT_"+time.Now().Format(dateFormat)+".txt")
			err(renameErr, "")
			log.Println("Please restart bot to reinitialize voice data file")
		}
	}
	return voiceData
}

//Compares a guild's voice states to the voice logs latest entries, then logs phantom users as disconnected without recording thier usage data.
func ghostbusting(s *discordgo.Session, g *discordgo.Guild) {
	var (
		latestUserVoiceData   []VoiceStateData
		confirmedVoiceLogData []VoiceStateData
	)
	for _, vsdata := range getVoiceLog(g).VoiceStateLog {
		latest := getLastVoiceState(g, vsdata.UserID)
		latestUserVoiceData = append(latestUserVoiceData, latest)
		if latest.Action == "Joined" {
			for _, vs := range g.VoiceStates {
				if latest.UserID == vs.UserID {
					confirmedVoiceLogData = append(confirmedVoiceLogData, latest)
				}
			}
		}
	}
	for _, latest := range latestUserVoiceData {
		pass := false
		for _, confirmed := range confirmedVoiceLogData {
			if latest.UserID == confirmed.UserID {
				pass = true
			}
		}
		if !pass && latest.UserID != "placeholder" {
			newVSD := VoiceStateData{latest.Username, latest.UserID, "Disconnected", latest.Channel, latest.ChannelID, time.Now().Format(logTimeFormat), "-"}
			writeToVoiceLog(g, newVSD)
		}
	}
	for _, vs := range g.VoiceStates {
		voiceStateToData(s, g, vs, true)
	}
}
