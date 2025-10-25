# Telegram AI Bot with Go & Gin

A Telegram bot that automatically extracts text from images using OpenAI's Vision API. When someone uploads an image in a group where the bot is present, it will analyze the image and reply with any readable text found (VIN numbers, license plates, etc.).

## 🚀 Features

- **Automatic Image Processing**: Detects uploaded images in Telegram groups
- **AI-Powered Text Extraction**: Uses OpenAI GPT-4o Vision to extract text from images
- **Smart Error Handling**: Provides user-friendly error messages
- **Cloud-Ready**: Designed for easy deployment on Render
- **Webhook-Based**: Efficient real-time processing

## 🛠️ Tech Stack

- **Language**: Go 1.22
- **Framework**: Gin
- **AI**: OpenAI GPT-4o (Vision)
- **Deployment**: Render (auto HTTPS domain)
- **Configuration**: Environment variables

## 📦 Dependencies

- `github.com/gin-gonic/gin` - Web framework
- `github.com/joho/godotenv` - Environment variable loading

## 🏗️ Project Structure

```
telegram-ai-bot-go/
├── main.go          # Main application file
├── go.mod           # Go module file
├── go.sum           # Go dependencies checksum
├── .env             # Environment variables (not in git)
├── .gitignore       # Git ignore rules
└── README.md        # This file
```

## ⚙️ Setup Instructions

### 1. Prerequisites

- Go 1.22 or later
- Telegram Bot Token (from [@BotFather](https://t.me/botfather))
- OpenAI API Key

### 2. Clone and Setup

```bash
git clone <your-repo-url>
cd telegram-ai-bot-go
```

### 3. Install Dependencies

```bash
go mod tidy
```

### 4. Environment Configuration

Create a `.env` file in the project root:

```env
# Telegram Bot Configuration
TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here

# OpenAI API Configuration
OPENAI_API_KEY=your_openai_api_key_here
```

**Getting your tokens:**

1. **Telegram Bot Token**: 
   - Message [@BotFather](https://t.me/botfather) on Telegram
   - Use `/newbot` command
   - Follow instructions to create your bot
   - Copy the token provided

2. **OpenAI API Key**:
   - Go to [OpenAI Platform](https://platform.openai.com/)
   - Create an account or sign in
   - Navigate to API Keys section
   - Create a new secret key

### 5. Local Testing

#### Option A: Using ngrok (Recommended)

1. Install [ngrok](https://ngrok.com/)
2. Start your bot:
   ```bash
   go run main.go
   ```
3. In another terminal, expose your local server:
   ```bash
   ngrok http 8080
   ```
4. Copy the HTTPS URL (e.g., `https://abc123.ngrok.io`)
5. Set the webhook:
   ```bash
   curl "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook?url=https://abc123.ngrok.io/webhook"
   ```

#### Option B: Direct Testing

1. Start your bot:
   ```bash
   go run main.go
   ```
2. Test the health endpoint:
   ```bash
   curl http://localhost:8080/
   ```
3. You should see: `{"message":"Bot is live 🚀","status":"healthy"}`

### 6. Add Bot to Group

1. Add your bot to a Telegram group
2. Make sure the bot has permission to read messages
3. Upload an image with text to test the functionality

## 🌍 Deployment to Render

### 1. Push to GitHub

```bash
git add .
git commit -m "Initial commit"
git push origin main
```

### 2. Deploy on Render

1. Go to [Render Dashboard](https://dashboard.render.com/)
2. Click "New +" → "Web Service"
3. Connect your GitHub repository
4. Configure the service:
   - **Name**: `aliasauto-bot` (or your preferred name)
   - **Runtime**: Go
   - **Build Command**: `go build -o main`
   - **Start Command**: `./main`
5. Add environment variables:
   - `TELEGRAM_BOT_TOKEN`: Your bot token
   - `OPENAI_API_KEY`: Your OpenAI API key
6. Click "Create Web Service"

### 3. Set Webhook

Once deployed, Render will provide a URL like `https://aliasauto-bot.onrender.com`. Set the webhook:

```bash
curl "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook?url=https://aliasauto-bot.onrender.com/webhook"
```

### 4. Verify Deployment

1. Check the health endpoint: `https://aliasauto-bot.onrender.com/`
2. Add the bot to a group and test with an image

## 🔧 API Endpoints

### GET `/`
Health check endpoint
- **Response**: `{"message":"Bot is live 🚀","status":"healthy"}`

### POST `/webhook`
Telegram webhook endpoint
- **Purpose**: Receives updates from Telegram
- **Content-Type**: `application/json`
- **Body**: Telegram Update object

## 🧪 Testing

### Test Cases

1. **Valid Image with Text**: Upload an image with clear text
2. **Image without Text**: Upload an image with no readable text
3. **Blurry Image**: Test with low-quality images
4. **Multiple Images**: Test with images containing multiple text elements
5. **Error Handling**: Test with invalid images or API failures

### Expected Behaviors

- ✅ Extracts and returns readable text from clear images
- ✅ Handles images with no text gracefully
- ✅ Provides user-friendly error messages
- ✅ Logs detailed errors for debugging

## 🐛 Troubleshooting

### Common Issues

1. **"Missing required environment variables"**
   - Ensure `.env` file exists with correct tokens
   - Check token validity

2. **"Failed to download image"**
   - Verify bot token is correct
   - Check if bot has proper permissions

3. **"OpenAI API error"**
   - Verify OpenAI API key is valid
   - Check if you have sufficient API credits
   - Ensure the image URL is publicly accessible

4. **Webhook not receiving updates**
   - Verify webhook URL is set correctly
   - Check if the bot is added to the group
   - Ensure bot has "Read Messages" permission

### Debug Mode

To enable debug logging, set the `GIN_MODE` environment variable:

```bash
export GIN_MODE=debug
go run main.go
```

## 📝 Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `TELEGRAM_BOT_TOKEN` | Bot token from BotFather | Yes |
| `OPENAI_API_KEY` | OpenAI API key for Vision API | Yes |
| `PORT` | Server port (Render sets this automatically) | No |

## 🔒 Security Notes

- Never commit `.env` file to version control
- Use environment variables in production
- Regularly rotate API keys
- Monitor API usage and costs

## 📈 Monitoring

- Check Render dashboard for deployment status
- Monitor OpenAI API usage in their dashboard
- Review application logs for errors
- Set up alerts for high API usage

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## 📄 License

This project is open source and available under the [MIT License](LICENSE).

## 🆘 Support

If you encounter issues:

1. Check the troubleshooting section above
2. Review the logs in Render dashboard
3. Verify your API keys are correct
4. Test with a simple image first

---

**Happy coding! 🚀**
