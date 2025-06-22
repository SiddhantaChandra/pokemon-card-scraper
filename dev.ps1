Write-Host "Starting SCRAPER 9000 in development mode..." -ForegroundColor Green
Write-Host "This will enable hot reloading for frontend changes." -ForegroundColor Yellow
Write-Host ""
docker-compose -f docker-compose.dev.yml up --build 