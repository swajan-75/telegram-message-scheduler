package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-co-op/gocron"
	"github.com/go-resty/resty/v2"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"
)

type ScheduledMessage struct {
	AddID    int    `json:"add_id"`
	ChatID   string `json:"chatID"`
	BotToken string `json:"botToken"`
	Time     string `json:"time"`
	Message  string `json:"message"`
}

type Adds struct {
	ScheduledMessages []ScheduledMessage `json:"scheduled_messages"`
}

const (
	// Default values
	defaultChatID   = "@bulksmtp"
	defaultBotToken = "7256894652:AAEDWmHITv14QB_peuYRyPpqPOEOWbD85iM"
	adminUserID     = 1274939394
)

var (
	userState = make(map[int]string)
	tempData  = make(map[int]string)
	scheduler *gocron.Scheduler
)

func sendTelegramMessage(botToken, chatID, message string) error {
	client := resty.New()

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	resp, err := client.R().
		SetQueryParams(map[string]string{
			"chat_id": chatID,
			"text":    message,
		}).
		Post(url)

	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("telegram API returned error: %s", resp.String())
	}

	fmt.Println("Message sent successfully!")
	return nil
}

func scheduleTelegramMessage(botToken, chatID, timeStr, message string) {

	_, err := scheduler.Every(1).Day().At(timeStr).Do(func() {
		err := sendTelegramMessage(botToken, chatID, message)
		if err != nil {
			log.Printf("Error sending message: %v", err)
			os.Exit(1)
		}
	})
	if err != nil {
		log.Printf("Error scheduling message: %v", err)
	}
	fmt.Printf("Message scheduled at %s: %s\n", timeStr, message)
}

func loadJSONFile(filename string) (Adds, error) {
	var adds Adds

	fileContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return adds, fmt.Errorf("failed to read JSON file: %w", err)
	}

	err = json.Unmarshal(fileContent, &adds)
	if err != nil {
		return adds, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return adds, nil
}

func saveJSONFile(filename string, adds Adds) error {
	// Convert the data back to JSON
	fileContent, err := json.MarshalIndent(adds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize JSON: %w", err)
	}

	err = ioutil.WriteFile(filename, fileContent, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

func generateRandomID(adds Adds) int {
	existingIDs := make(map[int]bool)
	for _, msg := range adds.ScheduledMessages {
		existingIDs[msg.AddID] = true
	}

	var newID int
	for {
		newID = rand.Intn(9000) + 1000
		if !existingIDs[newID] {
			break
		}
	}
	return newID
}

func deleteMessageByID(addID int, adds *Adds) bool {
	for i, msg := range adds.ScheduledMessages {
		if msg.AddID == addID {

			adds.ScheduledMessages = append(adds.ScheduledMessages[:i], adds.ScheduledMessages[i+1:]...)
			return true
		}
	}
	return false
}

func receiveTelegramMessages(botToken string) {
	client := resty.New()
	offset := 0

	for {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", botToken)
		resp, err := client.R().
			SetQueryParams(map[string]string{
				"offset": fmt.Sprintf("%d", offset),
			}).
			Get(url)

		if err != nil {
			log.Printf("Error fetching updates: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		var result struct {
			Ok     bool `json:"ok"`
			Result []struct {
				UpdateID int `json:"update_id"`
				Message  struct {
					Text string `json:"text"`
					Chat struct {
						ID int `json:"id"`
					} `json:"chat"`
				} `json:"message"`
			} `json:"result"`
		}

		err = json.Unmarshal(resp.Body(), &result)
		if err != nil {
			log.Printf("Error parsing updates: %v", err)
			continue
		}

		for _, update := range result.Result {
			chatID := update.Message.Chat.ID
			message := update.Message.Text
			offset = update.UpdateID + 1

			if chatID == adminUserID {
				handleAdminCommand(chatID, message)
			} else {
				fmt.Println("Received message from other user:", message)
			}
		}

		time.Sleep(1 * time.Second)
	}
}

func validateTimeFormat(timeStr string) bool {
	_, err := time.Parse("15:04", timeStr)
	return err == nil
}

func handleAdminCommand(chatID int, message string) {
	switch userState[chatID] {
	case "":
		if message == "set_add" {
			userState[chatID] = "waiting_for_time"
			sendTelegramMessage(defaultBotToken, fmt.Sprintf("%d", chatID), "Please enter the time for the new message (e.g., 14:00):")
		} else if message == "show_all" {
			showAllMessages(chatID)
		} else if len(message) > 4 && message[:4] == "del_" {
			deleteID := 0
			fmt.Sscanf(message, "del_%d", &deleteID)
			deleteScheduledMessage(chatID, deleteID)
		}
	case "waiting_for_time":
		if !validateTimeFormat(message) {
			sendTelegramMessage(defaultBotToken, fmt.Sprintf("%d", chatID), "Invalid time format. Please enter the time in HH:mm format (e.g., 14:00):")
			return
		}

		tempData[chatID] = message
		userState[chatID] = "waiting_for_message"
		sendTelegramMessage(defaultBotToken, fmt.Sprintf("%d", chatID), "Please enter the message content:")
	case "waiting_for_message":
		timeStr := tempData[chatID]
		newMessage := message

		adds, err := loadJSONFile("adds.json")
		if err != nil {
			sendTelegramMessage(defaultBotToken, fmt.Sprintf("%d", chatID), "Error loading adds.json file!")
			return
		}

		newScheduledMessage := ScheduledMessage{
			AddID:    generateRandomID(adds),
			ChatID:   defaultChatID,
			BotToken: defaultBotToken,
			Time:     timeStr,
			Message:  newMessage,
		}
		adds.ScheduledMessages = append(adds.ScheduledMessages, newScheduledMessage)

		err = saveJSONFile("adds.json", adds)
		if err != nil {
			sendTelegramMessage(defaultBotToken, fmt.Sprintf("%d", chatID), "Error saving the updated schedule!")
			return
		}

		// Schedule the new message immediately
		scheduleTelegramMessage(defaultBotToken, defaultChatID, timeStr, newMessage)

		// Confirm addition
		sendTelegramMessage(defaultBotToken, fmt.Sprintf("%d", chatID), fmt.Sprintf("New message scheduled successfully with ID: %d!", newScheduledMessage.AddID))

		// Reset the state for the user
		userState[chatID] = ""
		tempData[chatID] = ""
	}
}

func showAllMessages(chatID int) {
	adds, err := loadJSONFile("adds.json")
	if err != nil {
		sendTelegramMessage(defaultBotToken, fmt.Sprintf("%d", chatID), "Error loading adds.json file!")
		return
	}

	var messageList string
	for _, msg := range adds.ScheduledMessages {
		messageList += fmt.Sprintf("ID: %d \n\nTime: %s \n\nMessage: %s\n----------------------------------------------\n", msg.AddID, msg.Time, msg.Message)
	}

	if messageList == "" {
		messageList = "No scheduled messages."
	}

	sendTelegramMessage(defaultBotToken, fmt.Sprintf("%d", chatID), messageList)
}

func deleteScheduledMessage(chatID int, addID int) {
	// Load existing messages from adds.json
	adds, err := loadJSONFile("adds.json")
	if err != nil {
		sendTelegramMessage(defaultBotToken, fmt.Sprintf("%d", chatID), "Error loading adds.json file!")
		return
	}

	if deleteMessageByID(addID, &adds) {

		err = saveJSONFile("adds.json", adds)
		if err != nil {
			sendTelegramMessage(defaultBotToken, fmt.Sprintf("%d", chatID), "Error saving the updated schedule!")
			return
		}

		sendTelegramMessage(defaultBotToken, fmt.Sprintf("%d", chatID), fmt.Sprintf("Message with ID %d has been deleted.", addID))
	} else {
		sendTelegramMessage(defaultBotToken, fmt.Sprintf("%d", chatID), fmt.Sprintf("No message found with ID %d.", addID))
	}
}

func main() {
	// Load the adds.json file
	adds, err := loadJSONFile("adds.json")
	if err != nil {
		log.Fatalf("Error loading JSON file: %v", err)
		os.Exit(1)
	}

	scheduler = gocron.NewScheduler(time.Local)

	for _, messageInfo := range adds.ScheduledMessages {

		chatID := messageInfo.ChatID
		if chatID == "" {
			chatID = defaultChatID
		}

		botToken := messageInfo.BotToken
		if botToken == "" {
			botToken = defaultBotToken
		}

		scheduleTelegramMessage(botToken, chatID, messageInfo.Time, messageInfo.Message)
	}

	go receiveTelegramMessages(defaultBotToken)

	scheduler.StartBlocking()
}
