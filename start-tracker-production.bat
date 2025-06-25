@echo off
echo Starting Pokemon Card Scraper with Tracker System (Production Mode)

REM Set production environment variables
set TRACKER_ENABLED=true
set TRACKER_DEVELOPMENT=false
set TRACKER_SCAN_INTERVAL=1h
set TRACKER_MAX_WORKERS=3
set TRACKER_TIMEOUT=45s
set TRACKER_ENABLE_NOTIFICATIONS=true

REM Scraper settings
set TRACKER_HEADLESS=true
set TRACKER_SCRAPER_TIMEOUT=60s
set TRACKER_USER_AGENT=Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36

REM Discord webhook (uncomment and set your webhook URL)
REM set DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/YOUR_WEBHOOK_URL_HERE
set DISCORD_USERNAME=Pokemon Card Tracker
set DISCORD_TIMEOUT=15s

REM Database settings
set BADGER_PATH=./data/badger
set BADGER_IN_MEMORY=false

REM Server settings
set PORT=8080
set DEBUG=false

echo Configuration:
echo - Tracker Enabled: %TRACKER_ENABLED%
echo - Scan Interval: %TRACKER_SCAN_INTERVAL%
echo - Max Workers: %TRACKER_MAX_WORKERS%
echo - Discord Notifications: %TRACKER_ENABLE_NOTIFICATIONS%
echo - Database Path: %BADGER_PATH%
echo - Server Port: %PORT%
echo.

echo Building backend...
cd backend
go mod download
go build -o ../bin/server.exe ./cmd/server

echo Starting server...
..\bin\server.exe

pause 