@echo off
echo Starting SCRAPER 9000 in development mode...
echo This will enable hot reloading for frontend changes.
echo.
docker-compose -f docker-compose.dev.yml up --build 