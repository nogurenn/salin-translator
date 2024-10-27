package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	DiscordToken string `envconfig:"DISCORD_TOKEN" required:"true"`
	OpenAIToken  string `envconfig:"OPENAI_TOKEN" required:"true"`
}

var (
	// Map of flag emojis to language codes
	flagToLang = map[string]string{
		"🇺🇸": "English",    // US flag
		"🇬🇧": "English",    // UK flag
		"🇪🇸": "Spanish",    // Spain flag
		"🇫🇷": "French",     // French flag
		"🇩🇪": "German",     // German flag
		"🇮🇹": "Italian",    // Italian flag
		"🇯🇵": "Japanese",   // Japanese flag
		"🇰🇷": "Korean",     // Korean flag
		"🇨🇳": "Chinese",    // Chinese flag
		"🇵🇹": "Portuguese", // Portuguese flag
		"🇷🇺": "Russian",    // Russian flag
		// Add more flags as needed
	}
)

type OpenAIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type DiscordHandler struct {
	config *Config
}

func (h *DiscordHandler) reactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	// Ignore bot's own reactions
	if r.UserID == s.State.User.ID {
		return
	}

	// Check if the reaction is a flag emoji we support
	targetLang, ok := flagToLang[r.Emoji.Name]
	if !ok {
		return // Not a supported flag emoji
	}

	// Get the message that was reacted to
	msg, err := s.ChannelMessage(r.ChannelID, r.MessageID)
	if err != nil {
		log.Printf("Error fetching message: %v", err)
		return
	}

	// Don't translate empty messages
	if msg.Content == "" {
		return
	}

	// Translate the message
	translation, err := translateWithOpenAI(msg.Content, targetLang, h.config.OpenAIToken)
	if err != nil {
		log.Printf("Error translating text: %v", err)
		return
	}

	// Create response embed
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    msg.Author.Username,
			IconURL: msg.Author.AvatarURL(""),
		},
		Description: translation,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Translated to %s", targetLang),
		},
		Color: 0x00BFFF, // Light blue color
	}

	// Send the translation as a reply
	_, err = s.ChannelMessageSendEmbed(r.ChannelID, embed)
	if err != nil {
		log.Printf("Error sending translation: %v", err)
	}
}

func main() {

	// Get environment variables and creds
	var c Config
	err := envconfig.Process("", &c)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Create Discord session
	dg, err := discordgo.New("Bot " + c.DiscordToken)
	if err != nil {
		log.Fatal("Error creating Discord session:", err)
	}

	// Register reaction handlers
	handler := &DiscordHandler{config: &c}
	dg.AddHandler(handler.reactionAdd)

	// Open connection to Discord
	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening connection:", err)
	}
	defer dg.Close()

	fmt.Println("Bot is running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
}

func translateWithOpenAI(text, targetLang, openAIToken string) (string, error) {
	log.Printf("Translating text: %s", text)
	log.Printf("Target language: %s", targetLang)
	prompt := fmt.Sprintf("Translate the following text to %s. Only respond with the translation, nothing else: %s", targetLang, text)

	requestBody := OpenAIRequest{
		// Model: "gpt-4o-mini",
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openAIToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("error decoding response: %v", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no translation returned")
	}

	return response.Choices[0].Message.Content, nil
}
