package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Telegram API structures
type TelegramUpdate struct {
	UpdateID int64           `json:"update_id"`
	Message  TelegramMessage `json:"message"`
}

type TelegramMessage struct {
	MessageID int64           `json:"message_id"`
	From      TelegramUser    `json:"from"`
	Chat      TelegramChat    `json:"chat"`
	Date      int64           `json:"date"`
	Text      string          `json:"text"`
	Photo     []TelegramPhoto `json:"photo"`
}

type TelegramUser struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type TelegramChat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

type TelegramPhoto struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int    `json:"file_size"`
}

type TelegramGetFileResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		FileID   string `json:"file_id"`
		FilePath string `json:"file_path"`
	} `json:"result"`
}

// OpenAI API structures
type OpenAIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

type Content struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type OpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// Global variables
var (
	telegramBotToken string
	openAIAPIKey     string
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	telegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	openAIAPIKey = os.Getenv("OPENAI_API_KEY")

	if telegramBotToken == "" || openAIAPIKey == "" {
		log.Fatal("Missing required environment variables: TELEGRAM_BOT_TOKEN and OPENAI_API_KEY")
	}

	// Initialize Gin router
	router := gin.Default()

	// Routes
	router.GET("/", healthCheck)
	router.POST("/webhook", handleWebhook)
	router.POST("/test-image", handleTestImage)

	// Get port from environment (Render provides this)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)
	router.Run(":" + port)
}

func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Bot is live 🚀",
		"status":  "healthy",
	})
}

func handleWebhook(c *gin.Context) {
	var update TelegramUpdate

	if err := c.ShouldBindJSON(&update); err != nil {
		log.Printf("Error parsing webhook: %v", err)
		c.JSON(400, gin.H{"error": "Invalid JSON"})
		return
	}

	// Debug logging
	log.Printf("Received webhook - UpdateID: %d, MessageID: %d, ChatID: %d, Text: '%s', Photos: %d",
		update.UpdateID, update.Message.MessageID, update.Message.Chat.ID, update.Message.Text, len(update.Message.Photo))

	// Check if message has photos
	if len(update.Message.Photo) > 0 {
		log.Printf("Processing photo - Available photos: %d", len(update.Message.Photo))

		// Get the last uploaded photo (most recent/highest quality)
		latestPhoto := update.Message.Photo[len(update.Message.Photo)-1]

		log.Printf("Selected latest photo - FileID: %s, FileSize: %d", latestPhoto.FileID, latestPhoto.FileSize)

		// Validate we have a valid photo
		if latestPhoto.FileID == "" {
			log.Printf("No valid photo found")
			c.JSON(200, gin.H{"status": "ok"})
			return
		}

		// Download image from Telegram
		log.Printf("Downloading image with FileID: %s", latestPhoto.FileID)
		imageURL, err := downloadImage(latestPhoto.FileID)
		if err != nil {
			log.Printf("Error downloading image: %v", err)
			sendTelegramMessage(update.Message.Chat.ID, "Sorry, I couldn't download the image. Please try again.")
			c.JSON(200, gin.H{"status": "ok"})
			return
		}

		log.Printf("Image downloaded successfully: %s", imageURL)

		// Extract text using OpenAI Vision API
		log.Printf("Sending image to OpenAI for text extraction...")
		extractedData, err := extractTextFromImage(imageURL)
		if err != nil {
			log.Printf("Error extracting text: %v", err)
			sendTelegramMessage(update.Message.Chat.ID, "Sorry, I couldn't extract any text from this image. Please try with a clearer image.")
			c.JSON(200, gin.H{"status": "ok"})
			return
		}

		log.Printf("Text extracted successfully: %s", extractedData)

		// Send response back to Telegram
		responseText := fmt.Sprintf("🔍 **Extracted data from image:**\n\n```json\n%s\n```", extractedData)
		log.Printf("Sending response to Telegram chat %d", update.Message.Chat.ID)
		sendTelegramMessage(update.Message.Chat.ID, responseText)
		c.JSON(200, gin.H{"status": "ok"})
		return
	}

	// No photos in message
	log.Println("No photos in message")
	c.JSON(200, gin.H{"status": "ok"})
}

// Handle local image testing endpoint
func handleTestImage(c *gin.Context) {
	// Get the uploaded image file
	file, err := c.FormFile("image")
	if err != nil {
		log.Printf("Error getting uploaded file: %v", err)
		c.JSON(400, gin.H{"error": "No image file uploaded"})
		return
	}

	// Check if it's a supported image format
	contentType := file.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/jpg" && contentType != "image/png" {
		c.JSON(400, gin.H{"error": "Only JPEG, JPG, and PNG images are supported"})
		return
	}

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		log.Printf("Error opening uploaded file: %v", err)
		c.JSON(500, gin.H{"error": "Failed to open uploaded file"})
		return
	}
	defer src.Close()

	// Read image content into memory
	imageContent, err := io.ReadAll(src)
	if err != nil {
		log.Printf("Error reading image content: %v", err)
		c.JSON(500, gin.H{"error": "Failed to read image content"})
		return
	}

	// Convert to base64 and send to OpenAI
	base64Image := fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(imageContent))
	extractedData, err := extractTextFromImageBase64(base64Image)
	if err != nil {
		log.Printf("Error extracting text from image: %v", err)
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to extract text from image: %v", err)})
		return
	}

	// Get chat ID from environment
	chatIDStr := os.Getenv("TELEGRAM_CHAT_ID")
	if chatIDStr == "" {
		log.Println("Warning: TELEGRAM_CHAT_ID not set, skipping Telegram notification")
		c.JSON(200, gin.H{
			"success":        true,
			"message":        "Image processed successfully!",
			"extracted_data": extractedData,
			"filename":       file.Filename,
			"size":           len(imageContent),
		})
		return
	}

	// Parse chat ID
	var chatID int64
	if _, err := fmt.Sscanf(chatIDStr, "%d", &chatID); err != nil {
		log.Printf("Error parsing chat ID: %v", err)
		c.JSON(500, gin.H{"error": "Invalid TELEGRAM_CHAT_ID format"})
		return
	}

	// Send the original image to Telegram
	err = sendImageToTelegram(chatID, imageContent, fmt.Sprintf("Original Image: %s", file.Filename))
	if err != nil {
		log.Printf("Error sending image to Telegram: %v", err)
	}

	// Send extracted data to Telegram
	responseText := fmt.Sprintf("🔍 **Extracted data from image (%s):**\n\n```json\n%s\n```", file.Filename, extractedData)
	err = sendTelegramMessage(chatID, responseText)
	if err != nil {
		log.Printf("Error sending message to Telegram: %v", err)
	}

	// Return success response
	c.JSON(200, gin.H{
		"success":        true,
		"message":        "Image processed and sent to Telegram!",
		"extracted_data": extractedData,
		"filename":       file.Filename,
		"size":           len(imageContent),
		"chat_id":        chatID,
	})
}

func downloadImage(fileID string) (string, error) {
	// Get file info from Telegram
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", telegramBotToken, fileID)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	var fileResponse TelegramGetFileResponse
	if err := json.Unmarshal(body, &fileResponse); err != nil {
		return "", fmt.Errorf("failed to parse file response: %v", err)
	}

	if !fileResponse.OK {
		return "", fmt.Errorf("telegram API error: file not found")
	}

	// Construct the public URL for the image
	imageURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", telegramBotToken, fileResponse.Result.FilePath)

	return imageURL, nil
}

func extractTextFromImage(imageURL string) (string, error) {
	// Prepare OpenAI request
	request := OpenAIRequest{
		Model: "gpt-4o-mini",
		Messages: []Message{
			{
				Role: "user",
				Content: []Content{
					{
						Type: "text",
						Text: "Extract all visible text from this image and return it as a structured JSON object. Look for VIN numbers, license plates, vehicle information, addresses, names, or any other readable text. Return the data in this exact JSON format: {\"vin\": \"extracted_vin_or_null\", \"license_plate\": \"extracted_plate_or_null\", \"vehicle_info\": \"any_vehicle_details\", \"address\": \"any_address_found\", \"other_text\": \"any_other_readable_text\"}. If a field is not found, use null. Only return valid JSON, no other text.",
					},
					{
						Type: "image_url",
						ImageURL: &ImageURL{
							URL: imageURL,
						},
					},
				},
			},
		},
	}

	// Convert request to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Make request to OpenAI
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openAIAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	var openAIResponse OpenAIResponse
	if err := json.Unmarshal(body, &openAIResponse); err != nil {
		return "", fmt.Errorf("failed to parse OpenAI response: %v", err)
	}

	if len(openAIResponse.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return openAIResponse.Choices[0].Message.Content, nil
}

func extractTextFromImageBase64(base64Image string) (string, error) {
	request := OpenAIRequest{
		Model: "gpt-4o-mini",
		Messages: []Message{
			{
				Role: "user",
				Content: []Content{
					{
						Type: "text",
						Text: "Extract all visible text from this image and return it as a structured JSON object. Look for VIN numbers, license plates, vehicle information, addresses, names, or any other readable text. Return the data in this exact JSON format: {\"vin\": \"extracted_vin_or_null\", \"license_plate\": \"extracted_plate_or_null\", \"vehicle_info\": \"any_vehicle_details\", \"address\": \"any_address_found\", \"other_text\": \"any_other_readable_text\"}. If a field is not found, use null. Only return valid JSON, no other text.",
					},
					{
						Type: "image_url",
						ImageURL: &ImageURL{
							URL: base64Image,
						},
					},
				},
			},
		},
	}

	// Convert request to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Make request to OpenAI
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openAIAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	var openAIResponse OpenAIResponse
	if err := json.Unmarshal(body, &openAIResponse); err != nil {
		return "", fmt.Errorf("failed to parse OpenAI response: %v", err)
	}

	if len(openAIResponse.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return openAIResponse.Choices[0].Message.Content, nil
}

func sendTelegramMessage(chatID int64, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramBotToken)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	defer resp.Body.Close()

	return nil
}

func sendImageToTelegram(chatID int64, imageData []byte, caption string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", telegramBotToken)

	// Create multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add chat_id
	writer.WriteField("chat_id", fmt.Sprintf("%d", chatID))
	writer.WriteField("caption", caption)

	// Add photo
	part, err := writer.CreateFormFile("photo", "image.jpg")
	if err != nil {
		return fmt.Errorf("failed to create form file: %v", err)
	}
	part.Write(imageData)

	writer.Close()

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send image: %v", err)
	}
	defer resp.Body.Close()

	return nil
}
