@echo off
echo Starting Pokemon Card Scraper with Tracker System (Development Mode)

REM Set development environment variables
set TRACKER_ENABLED=true
set TRACKER_DEVELOPMENT=true
set TRACKER_SCAN_INTERVAL=5m
set TRACKER_MAX_WORKERS=2
set TRACKER_TIMEOUT=30s
set TRACKER_ENABLE_NOTIFICATIONS=false

REM Scraper settings
set TRACKER_HEADLESS=true
set TRACKER_SCRAPER_TIMEOUT=30s
set TRACKER_USER_AGENT=Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36

REM Discord webhook (disabled for development)
REM set DISCORD_WEBHOOK_URL=
set DISCORD_USERNAME=Pokemon Card Tracker (Dev)
set DISCORD_TIMEOUT=10s

REM Database settings
set BADGER_PATH=./data/badger-dev
set BADGER_IN_MEMORY=false

REM Server settings
set PORT=8080
set DEBUG=true

echo Development Configuration:
echo - Tracker Enabled: %TRACKER_ENABLED%
echo - Development Mode: %TRACKER_DEVELOPMENT%
echo - Scan Interval: %TRACKER_SCAN_INTERVAL%
echo - Max Workers: %TRACKER_MAX_WORKERS%
echo - Discord Notifications: %TRACKER_ENABLE_NOTIFICATIONS%
echo - Database Path: %BADGER_PATH%
echo - Debug Mode: %DEBUG%
echo.

echo Building backend...
cd backend
go mod download
go build -o ../bin/server.exe ./cmd/server

echo Starting development server...
..\bin\server.exe

pause 