@echo off
echo Starting Pokemon Card Scraper with Tracker in Docker (Development Mode)
echo.

echo Building and starting containers...
docker-compose -f docker-compose.dev.yml down
docker-compose -f docker-compose.dev.yml build --no-cache
docker-compose -f docker-compose.dev.yml up

echo.
echo Docker containers stopped.
pause 