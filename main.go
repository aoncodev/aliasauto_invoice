package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	MessageID int64             `json:"message_id"`
	From      TelegramUser      `json:"from"`
	Chat      TelegramChat      `json:"chat"`
	Date      int64             `json:"date"`
	Text      string            `json:"text"`
	Photo     []TelegramPhoto   `json:"photo"`
	Document  *TelegramDocument `json:"document,omitempty"`
}

type TelegramDocument struct {
	FileName     string `json:"file_name"`
	MimeType     string `json:"mime_type"`
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	FileSize     int    `json:"file_size"`
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
		FileSize int    `json:"file_size"`
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

	// Check if message has photos
	if len(update.Message.Photo) > 0 {
		// Get the largest photo (last in the array)
		largestPhoto := update.Message.Photo[len(update.Message.Photo)-1]

		// Download image from Telegram
		imageURL, err := downloadImage(largestPhoto.FileID)
		if err != nil {
			log.Printf("Error downloading image: %v", err)
			sendTelegramMessage(update.Message.Chat.ID, "Sorry, I couldn't download the image. Please try again.")
			c.JSON(200, gin.H{"status": "ok"})
			return
		}

		// Extract text using OpenAI Vision API
		extractedText, err := extractTextFromImage(imageURL)
		if err != nil {
			log.Printf("Error extracting text: %v", err)
			sendTelegramMessage(update.Message.Chat.ID, "Sorry, I couldn't extract any text from this image. Please try with a clearer image.")
			c.JSON(200, gin.H{"status": "ok"})
			return
		}

		// Send response back to Telegram
		responseText := fmt.Sprintf("🔍 **Extracted text from image:**\n\n%s", extractedText)
		sendTelegramMessage(update.Message.Chat.ID, responseText)
		c.JSON(200, gin.H{"status": "ok"})
		return
	}

	// Check if message has a document (PDF)
	if update.Message.Document != nil && isPDF(update.Message.Document.MimeType) {
		// Download PDF from Telegram
		pdfURL, err := downloadDocument(update.Message.Document.FileID)
		if err != nil {
			log.Printf("Error downloading PDF: %v", err)
			sendTelegramMessage(update.Message.Chat.ID, "Sorry, I couldn't download the PDF. Please try again.")
			c.JSON(200, gin.H{"status": "ok"})
			return
		}

		// Extract text from PDF using OpenAI
		extractedText, err := extractTextFromPDF(pdfURL)
		if err != nil {
			log.Printf("Error extracting text from PDF: %v", err)
			sendTelegramMessage(update.Message.Chat.ID, "Sorry, I couldn't extract any text from this PDF. Please try with a different document.")
			c.JSON(200, gin.H{"status": "ok"})
			return
		}

		// Send response back to Telegram
		responseText := fmt.Sprintf("📄 **Extracted text from PDF:**\n\n%s", extractedText)
		sendTelegramMessage(update.Message.Chat.ID, responseText)
		c.JSON(200, gin.H{"status": "ok"})
		return
	}

	// No photos or PDFs in message
	log.Println("No photos or PDFs in message")
	c.JSON(200, gin.H{"status": "ok"})
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

	// Construct the public URL for the file
	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", telegramBotToken, fileResponse.Result.FilePath)

	return fileURL, nil
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
						Text: "Extract any text visible in this image, including VIN numbers, license plates, or any other readable text. If you find multiple pieces of text, list them clearly.",
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

	// Marshal request to JSON
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

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("OpenAI API error: %s", string(body))
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
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: %s", string(body))
	}

	return nil
}

// Helper function to check if a file is a PDF
func isPDF(mimeType string) bool {
	return mimeType == "application/pdf"
}

// Download document from Telegram (works for PDFs and other documents)
func downloadDocument(fileID string) (string, error) {
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

	// Construct the public URL for the file
	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", telegramBotToken, fileResponse.Result.FilePath)

	return fileURL, nil
}

// Extract text from PDF using OpenAI API
func extractTextFromPDF(pdfURL string) (string, error) {
	// For PDFs, we'll use a different approach since OpenAI Vision API doesn't directly support PDFs
	// We'll use the text extraction model instead

	// First, we need to convert the PDF to a format that can be processed
	// For now, we'll use a simple approach with the GPT-4o model

	request := OpenAIRequest{
		Model: "gpt-4o-mini",
		Messages: []Message{
			{
				Role: "user",
				Content: []Content{
					{
						Type: "text",
						Text: fmt.Sprintf("I have a PDF document at this URL: %s. Please extract all the text content from this PDF. If you cannot access the URL directly, please let me know and I'll provide the content in a different way.", pdfURL),
					},
				},
			},
		},
	}

	// Marshal request to JSON
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

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("OpenAI API error: %s", string(body))
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
