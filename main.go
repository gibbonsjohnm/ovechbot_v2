package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Variables
var (
	Token    = os.Getenv("DISCORD_BOT_TOKEN")
	Channel  = os.Getenv("DISCORD_BOT_CHANNEL")
	homeTeam string
	awayTeam string
)

// Define a struct to match the JSON structure
type Player struct {
	FeaturedStats struct {
		RegularSeason struct {
			Career struct {
				Goals int `json:"goals"`
			} `json:"career"`
		} `json:"regularSeason"`
	} `json:"featuredStats"`
}

// Define a struct to match the JSON structure
type Game struct {
	AwayTeam struct {
		Abbrev string `json:"abbrev"`
	} `json:"awayTeam"`
	HomeTeam struct {
		Abbrev string `json:"abbrev"`
	} `json:"homeTeam"`
	GameState string `json:"gameState"`
}

type ScoreData struct {
	Games []Game `json:"games"`
}

func main() {
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		log.Println("error creating Discord session,", err)
		return
	}

	//dg.AddHandler(messageCreate)
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	err = dg.Open()
	if err != nil {
		log.Println("error opening connection,", err)
		return
	}

	log.Println("Bot is now running. Press CTRL-C to exit.")

	// Create a ticker that ticks every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		var lastGoals int
		for {
			select {
			case <-ticker.C:
				if isOvechkinPlaying() {
					// Update the bot's status.
					err = dg.UpdateWatchStatus(0, homeTeam+" vs. "+awayTeam)
					if err != nil {
						log.Println("Error updating status:", err)
						return
					}
					goals := getGoals()
					if goals > lastGoals && lastGoals != 0 {
						log.Printf("Ovechkin scored goal #%d! Sending message.", goals)
						message := fmt.Sprintf("🚨 **Alexander Ovechkin has scored goal #%d** 🚨\n\n:hockey: ***Goals remaining to tie Gretzky: %d***", goals, 894-goals)
						sendImageWithMessage(dg, Channel, message, "images/8471214.png")
					}
					lastGoals = goals
				} else {
					// Update the bot's status.
					err = dg.UpdateWatchStatus(0, "No Caps Games :(")
					if err != nil {
						log.Println("Error updating status:", err)
						return
					}
				}
			}
		}
	}()

	// Wait here until CTRL-C or other term signal is received.
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	ticker.Stop() // Stop the ticker when shutting down
	dg.Close()
}

// func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
// 	if m.Author.ID == s.State.User.ID {
// 		return
// 	}

// 	log.Printf("Received message: %s\n", m.Content) // Log the message content

// 	if m.Content == "ping" {
// 		s.ChannelMessageSend(m.ChannelID, "Pong!")
// 	} else if m.Content == "pong" {
// 		s.ChannelMessageSend(m.ChannelID, "Ping!")
// 	}
// }

func readJson(url string) string {
	// Make the HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		log.Println("Error making request:", err)
		return ""
	}
	defer resp.Body.Close()

	// Check if the response status is OK (200)
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error: received status code %d\n", resp.StatusCode)
		return ""
	}

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		return ""
	}

	return string(body)
}

func getGoals() int {
	// Call the readJson function to get the JSON data
	data := readJson("https://api-web.nhle.com/v1/player/8471214/landing")

	// Create a new Player struct
	var player Player

	// Unmarshal the JSON data into the Player struct
	err := json.Unmarshal([]byte(data), &player)
	if err != nil {
		log.Println("Error unmarshalling JSON data:", err)
		return 0
	}

	// Return the number of goals
	return player.FeaturedStats.RegularSeason.Career.Goals
}

func isOvechkinPlaying() bool {
	// Call the readJson function to get the JSON data
	data := readJson("https://api-web.nhle.com/v1/score/now")

	// Create a new ScoreData struct
	var scoreData ScoreData

	// Unmarshal the JSON data into the ScoreData struct
	err := json.Unmarshal([]byte(data), &scoreData)
	if err != nil {
		log.Println("Error unmarshalling JSON data:", err)
		return false
	}

	// Check if Ovechkin is playing
	for _, game := range scoreData.Games {
		if game.AwayTeam.Abbrev == "WSH" || game.HomeTeam.Abbrev == "WSH" {
			awayTeam = game.AwayTeam.Abbrev
			homeTeam = game.HomeTeam.Abbrev
			if game.GameState == "LIVE" || game.GameState == "CRIT" {
				return true
			}
		}
	}

	return false
}

func sendImageWithMessage(s *discordgo.Session, channelID, message, imagePath string) {
	// Open the image file
	file, err := os.Open(imagePath)
	if err != nil {
		log.Println("error opening file,", err)
		return
	}
	defer file.Close()

	// Send the message with the image
	_, err = s.ChannelFileSendWithMessage(channelID, message, "image.png", file)
	if err != nil {
		log.Println("error sending message with image,", err)
	}
}
