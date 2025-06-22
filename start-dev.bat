@echo off
echo Starting Pokemon Card Scraper in development mode...

echo Stopping any existing services...
docker-compose down
docker-compose -f docker-compose.dev.yml down

echo Building and starting development services...
docker-compose -f docker-compose.dev.yml build --no-cache
docker-compose -f docker-compose.dev.yml up -d

echo Waiting for services to start...
timeout /t 15 /nobreak

echo Checking service status...
docker-compose -f docker-compose.dev.yml ps

echo.
echo Services should be available at:
echo Frontend (Development): http://localhost:3000
echo Backend API: http://localhost:8080
echo.
echo Press any key to exit...
pause 