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

type MessageArray struct {
	MessageLog []MessageData `json:"Array"`
}
type MessageData struct {
	ID        string `json: "ID"`
	Username  string `json:"Username"`
	UserID    string `json:"UserID"`
	Type      string `json:"Type"`
	Channel   string `json:"Channel"`
	ChannelID string `json:"ChannelID"`
	Time      string `json:"Time"`
}

func messageToData(s *discordgo.Session, g *discordgo.Guild, m *discordgo.Message) {
	time, e := m.Timestamp.Parse()
	if err(e, "") {
	}
	md := MessageData{m.ID, m.Author.Username, m.Author.ID, "Message", "", m.ChannelID, time.Format(logTimeFormat)}
	if m.ChannelID != "" {
		channel, e := s.Channel(m.ChannelID)
		if err(e, "") {
		} else {
			md.Channel = channel.Name
		}
	}
	writeToTextLog(g, md)
}
func writeToTextLog(g *discordgo.Guild, md MessageData) {
	setPath(g)
	//Create a JSON Object based on message data.
	newObjectText := "{\n\t\"Array\":[\n\t\t{\n\t\t\t\"ID\": \"" + md.ID + "\",\n\t\t\t\"Username\": \"" + md.Username + "\",\n\t\t\t\"UserID\": \"" + md.UserID + "\",\n\t\t\t\"Type\": \"" + md.Type + "\",\n\t\t\t\"Channel\": \"" + md.Channel + "\",\n\t\t\t\"ChannelID\": \"" + md.ChannelID + "\",\n\t\t\t\"Time\": \"" + md.Time + "\"\n\t\t},"
	logText, readErr := ioutil.ReadFile(filePath["TEXTPATH"])
	if err(readErr, "Failed to read message log file") {
		return
	}

	//Replace the beginning of the message data file with a new object. This way the JSON decoder can read the most recent data first.
	newLog := strings.Replace(string(logText[:]), "{\n\t\"Array\":[", newObjectText, 1)
	writeErr := ioutil.WriteFile(filePath["TEXTPATH"], []byte(newLog), os.ModePerm)
	if err(writeErr, "Failed to store message data for user: "+md.Username) {
		return
	} else {
		log.Println("")
		log.Println("Stored message data for user: " + md.Username)
	}
}
func getMessageLog(g *discordgo.Guild) MessageArray {
	setPath(g)
	messageData := MessageArray{}
	if filePath["TEXTPATH"] != "" {
		messageDataFile, _ := os.Open(filePath["TEXTPATH"])
		decoder := json.NewDecoder(messageDataFile)
		decodeErr := decoder.Decode(&messageData)
		if err(decodeErr, "Failed to decode message data file, renaming corrupt file: \""+filePath["TEXTPATH"]+"_CORRUPT_"+time.Now().Format(dateFormat)+".txt\"") {
			closeErr := messageDataFile.Close()
			err(closeErr, "")
			renameErr := os.Rename(filePath["TEXTPATH"], filePath["TEXTPATH"]+"_CORRUPT_"+time.Now().Format(dateFormat)+".txt")
			err(renameErr, "")
			log.Println("Please restart bot to reinitialize message data file")
		}
	}
	return messageData
}
func backfillMessages(s *discordgo.Session, g *discordgo.Guild) {
	for _, c := range g.Channels {
		after := getLastMessageData(g, c.ID)
		if after.ID != "placeholder" && after.ID != c.LastMessageID {
			messages, e := s.ChannelMessages(c.ID, 100, c.LastMessageID, after.ID, "")
			if err(e, "") {
				return
			}
			for _, m := range messages {
				messageToData(s, g, m)
			}
			if len(messages) == 100 {
				owner, e := s.User(g.OwnerID)
				if err(e, "Error getting guild owner for guild: \""+g.Name+"\"") {
					return
				}
				log.Println("Too many messages to backfill in channel: \"" + c.Name + "\" in guild: \"" + g.Name + "\". Alerting guild owner: " + owner.Username + "#" + owner.Discriminator + "\"")
			}
		}
	}
}

//Gets the last message data object from a specific channel
func getLastMessageData(g *discordgo.Guild, cID string) MessageData {
	for _, md := range getMessageLog(g).MessageLog {
		if md.ChannelID == cID {
			return md
		}
	}
	return getMessageLog(g).MessageLog[0] //If no previous message data exists for this channel, return placeholder data.
}
