@echo off
setlocal enabledelayedexpansion

REM Production deployment script for Pokemon Card Scraper (Windows)

echo 🚀 Starting production deployment...

REM Check if Docker is running
docker info >nul 2>&1
if errorlevel 1 (
    echo ❌ Docker is not running. Please start Docker and try again.
    exit /b 1
)

REM Check if docker-compose is available
docker-compose --version >nul 2>&1
if errorlevel 1 (
    echo ❌ docker-compose is not installed. Please install docker-compose and try again.
    exit /b 1
)

REM Clean up any existing containers and volumes
echo 🧹 Cleaning up existing containers and volumes...
docker-compose down --volumes --remove-orphans

REM Remove any dangling images
echo 🗑️ Removing dangling images...
docker image prune -f

REM Create data directory if it doesn't exist
echo 📁 Creating data directory...
if not exist ".\data" mkdir ".\data"
if not exist ".\data\badger" mkdir ".\data\badger"

REM Build and start services
echo 🔨 Building and starting services...
docker-compose build --no-cache
if errorlevel 1 (
    echo ❌ Failed to build services
    exit /b 1
)

docker-compose up -d
if errorlevel 1 (
    echo ❌ Failed to start services
    exit /b 1
)

REM Wait for services to be healthy
echo ⏳ Waiting for services to be healthy...
set timeout=300
set elapsed=0
set interval=5

:wait_loop
if !elapsed! geq !timeout! (
    echo ❌ Services failed to become healthy within !timeout! seconds
    echo 📋 Service status:
    docker-compose ps
    echo 📋 Backend logs:
    docker-compose logs backend
    echo 📋 Frontend logs:
    docker-compose logs frontend
    exit /b 1
)

REM Check if services are healthy
docker-compose ps | findstr "healthy" >nul 2>&1
if not errorlevel 1 (
    echo ✅ Services are healthy!
    goto test_endpoints
)

echo ⏳ Waiting for services to be healthy... (!elapsed!s/!timeout!s)
timeout /t !interval! /nobreak >nul
set /a elapsed=!elapsed! + !interval!
goto wait_loop

:test_endpoints
REM Test API endpoints
echo 🧪 Testing API endpoints...
timeout /t 5 /nobreak >nul

REM Test backend health
curl -f -s http://localhost:8080/health >nul 2>&1
if errorlevel 1 (
    echo ❌ Backend health check failed
    docker-compose logs backend
    exit /b 1
) else (
    echo ✅ Backend health check passed
)

REM Test frontend
curl -f -s http://localhost:3000 >nul 2>&1
if errorlevel 1 (
    echo ❌ Frontend health check failed
    docker-compose logs frontend
    exit /b 1
) else (
    echo ✅ Frontend health check passed
)

REM Test API proxy
curl -f -s http://localhost:3000/api/health >nul 2>&1
if errorlevel 1 (
    echo ❌ API proxy health check failed
    echo 📋 Frontend logs:
    docker-compose logs frontend
    exit /b 1
) else (
    echo ✅ API proxy health check passed
)

echo.
echo 🎉 Production deployment successful!
echo.
echo 📋 Service Information:
echo    Frontend: http://localhost:3000
echo    Backend API: http://localhost:8080
echo    Health Check: http://localhost:8080/health
echo.
echo 📋 Available API endpoints:
echo    GET  /api/health              - Health check
echo    GET  /api/cards               - List all cards
echo    GET  /api/cards/search        - Search cards
echo    POST /api/scrape/start        - Start scraping
echo    GET  /api/scrape/status       - Get scrape status
echo    DELETE /api/database/reset    - Reset database
echo.
echo 📋 Management commands:
echo    docker-compose logs           - View logs
echo    docker-compose ps             - View service status
echo    docker-compose down           - Stop services
echo    docker-compose up -d          - Start services
echo.
echo ✅ Deployment complete!

pause 