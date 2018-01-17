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
func voiceStateToData(s *discordgo.Session, vs *discordgo.VoiceState) VoiceStateData {
	var vsd VoiceStateData
	user, e := s.User(vs.UserID)
	if err(e, "") {
	}
	if vs.ChannelID != "" {
		channel, e := s.Channel(vs.ChannelID)
		if err(e, "") {
		}
		vsd = VoiceStateData{user.Username, user.ID, "-", channel.Name, channel.ID, time.Now().Format(logTimeFormat), "-"}
	} else {
		vsd = VoiceStateData{user.Username, user.ID, "-", "", vs.ChannelID, time.Now().Format(logTimeFormat), "-"}
	}
	return vsd
}

//Sorts voice state data and writes it to file.
func writeToVoiceLog(s *discordgo.Session, g *discordgo.Guild, vsd VoiceStateData, isInit bool) {
	setPath(g)
	for _, m := range g.Members {
		if m.User.ID == vsd.UserID {

			channelID := vsd.ChannelID
			channel := vsd.Channel
			action := vsd.Action
			duration := vsd.Duration

			if action == "-" {
				last := getLastVoiceState(g, m.User.ID)
				if vsd.ChannelID == "" {
					action = "Disconnected"
					if last.ChannelID != g.AfkChannelID {
						duration = getDuration(vsd.Time, last.Time)
					}
					channelID = last.ChannelID
					channel = last.Channel
				} else if vsd.ChannelID == g.AfkChannelID {
					action = "AFK"
				} else if isInit || last.Action == "placeholder" {
					action = "Joined"
				} else if last.Action == "Disconnected" {
					action = "Joined"
				} else if vsd.ChannelID != last.ChannelID {
					action = "Changed Channel"
					if last.ChannelID != g.AfkChannelID {
						duration = getDuration(vsd.Time, last.Time)
					}
					channelID = last.ChannelID
					channel = last.Channel
				} else {
					return
				}
			}
			//Create a JSON Object based on voice state data.
			newObjectText := "{\n\t\"Array\":[\n\t\t{\n\t\t\t\"Username\": \"" + m.User.Username + "\",\n\t\t\t\"UserID\": \"" + m.User.ID + "\",\n\t\t\t\"Action\": \"" + action + "\",\n\t\t\t\"Channel\": \"" + channel + "\",\n\t\t\t\"ChannelID\": \"" + channelID + "\",\n\t\t\t\"Time\": \"" + vsd.Time + "\",\n\t\t\t\"Duration\": \"" + duration + "\"\n\t\t},"
			logText, readErr := ioutil.ReadFile(filePath["VOICEPATH"])
			if err(readErr, "Failed to read voice log file") {
				return
			}

			//Replace the beginning of the voice data file with a new object. This way the JSON decoder can read the most recent logs first.
			newLog := strings.Replace(string(logText[:]), "{\n\t\"Array\":[", newObjectText, 1)
			writeErr := ioutil.WriteFile(filePath["VOICEPATH"], []byte(newLog), os.ModePerm)
			if err(writeErr, "Failed to store voice state data for user: "+m.User.Username) {
				return
			} else {
				log.Println("")
				log.Println("Stored voice state data for user: " + m.User.Username)
			}
			//If the user changed voice channels, write a forced entry containing their current channel.
			if action == "Changed Channel" {
				newVSD := VoiceStateData{vsd.Username, vsd.UserID, "Joined", vsd.Channel, vsd.ChannelID, time.Now().Format(logTimeFormat), "-"}
				writeToVoiceLog(s, g, newVSD, false)
			}
		}
	}
	return
}

//Calculates the difference between the join and disconnect time of a user. Useful for usage metrics.
func getDuration(currentTime string, lastTime string) string {
	current, e := time.Parse(logTimeFormat, currentTime)
	err(e, "Failed to parse \"Time\" value. Surely you didn't mess with the data, did you?")
	last, e := time.Parse(logTimeFormat, lastTime)
	err(e, "Failed to parse \"Time\" value. Surely you didn't mess with the data, did you?")
	return current.Sub(last).String()
}

//Gets the last voice state of the specified user.
func getLastVoiceState(g *discordgo.Guild, userID string) VoiceStateData {
	for _, v := range getVoiceLog(g).VoiceStateLog {
		if v.UserID == userID {
			return v
		}
	}

	return getVoiceLog(g).VoiceStateLog[0] //If no state exists for this user, return placeholder state.
}

//Decodes voice log file for the specified guild.
func getVoiceLog(g *discordgo.Guild) VoiceStateArray {
	setPath(g)
	voiceData := VoiceStateArray{}
	if filePath["VOICEPATH"] != "" {
		voiceDataFile, _ := os.Open(filePath["VOICEPATH"])
		decoder := json.NewDecoder(voiceDataFile)
		decodeErr := decoder.Decode(&voiceData)
		if err(decodeErr, "Failed to decode voice data file, creating new file and renaming corrupt file: \""+filePath["VOICEPATH"]+"_CORRUPT_"+time.Now().Format(dateFormat)+".txt\"") {
			closeErr := voiceDataFile.Close()
			err(closeErr, "")
			renameErr := os.Rename(filePath["VOICEPATH"], filePath["VOICEPATH"]+"_CORRUPT_"+time.Now().Format(dateFormat)+".txt")
			err(renameErr, "")
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
		if !pass {
			newVSD := VoiceStateData{latest.Username, latest.UserID, "Disconnected", latest.Channel, latest.ChannelID, time.Now().Format(logTimeFormat), "-"}
			writeToVoiceLog(s, g, newVSD, false)
		}
	}
	for _, vs := range g.VoiceStates {
		writeToVoiceLog(s, g, voiceStateToData(s, vs), true)
	}
}
