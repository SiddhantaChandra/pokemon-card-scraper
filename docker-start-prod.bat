@echo off
echo Starting Pokemon Card Scraper with Tracker in Docker (Production Mode)
echo.

echo Configuration:
echo - Tracker Enabled: Yes
echo - Scan Interval: 1 hour
echo - Max Workers: 3
echo - Notifications: Enabled (set DISCORD_WEBHOOK_URL in docker.env)
echo - Chrome: Headless mode
echo - Database: Persistent storage in ./data
echo.

echo Building and starting containers...
docker-compose down
docker-compose build --no-cache
docker-compose up -d

echo.
echo Containers started in background. 
echo Frontend: http://localhost:3000
echo Backend API: http://localhost:8080
echo.
echo To view logs: docker-compose logs -f
echo To stop: docker-compose down
echo.
pause 