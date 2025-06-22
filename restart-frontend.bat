@echo off
echo Restarting Pokemon Card Scraper with fixed frontend...

echo Stopping existing services...
docker-compose down

echo Rebuilding frontend service...
docker-compose build frontend --no-cache

echo Starting services...
docker-compose up -d

echo Waiting for services to start...
timeout /t 10 /nobreak

echo Checking service status...
docker-compose ps

echo.
echo Frontend should now be available at: http://localhost:3000
echo Backend is available at: http://localhost:8080
echo.
echo Press any key to exit...
pause 