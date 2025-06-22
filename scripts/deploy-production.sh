#!/bin/bash

# Production deployment script for Pokemon Card Scraper

set -e

echo "🚀 Starting production deployment..."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker and try again."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "❌ docker-compose is not installed. Please install docker-compose and try again."
    exit 1
fi

# Clean up any existing containers and volumes
echo "🧹 Cleaning up existing containers and volumes..."
docker-compose down --volumes --remove-orphans || true

# Remove any dangling images
echo "🗑️ Removing dangling images..."
docker image prune -f || true

# Create data directory if it doesn't exist
echo "📁 Creating data directory..."
mkdir -p ./data/badger

# Set proper permissions for data directory
chmod 755 ./data
chmod 755 ./data/badger

# Build and start services
echo "🔨 Building and starting services..."
docker-compose build --no-cache
docker-compose up -d

# Wait for services to be healthy
echo "⏳ Waiting for services to be healthy..."
timeout=300  # 5 minutes timeout
elapsed=0
interval=5

while [ $elapsed -lt $timeout ]; do
    if docker-compose ps | grep -q "healthy"; then
        backend_healthy=$(docker-compose ps backend | grep -q "healthy" && echo "true" || echo "false")
        frontend_healthy=$(docker-compose ps frontend | grep -q "healthy" && echo "true" || echo "false")
        
        if [ "$backend_healthy" = "true" ] && [ "$frontend_healthy" = "true" ]; then
            echo "✅ All services are healthy!"
            break
        fi
    fi
    
    echo "⏳ Waiting for services to be healthy... (${elapsed}s/${timeout}s)"
    sleep $interval
    elapsed=$((elapsed + interval))
done

if [ $elapsed -ge $timeout ]; then
    echo "❌ Services failed to become healthy within ${timeout} seconds"
    echo "📋 Service status:"
    docker-compose ps
    echo "📋 Backend logs:"
    docker-compose logs backend | tail -20
    echo "📋 Frontend logs:"
    docker-compose logs frontend | tail -20
    exit 1
fi

# Test API endpoints
echo "🧪 Testing API endpoints..."
sleep 5  # Give services a moment to fully start

# Test backend health
if curl -f -s http://localhost:8080/health > /dev/null; then
    echo "✅ Backend health check passed"
else
    echo "❌ Backend health check failed"
    docker-compose logs backend | tail -10
    exit 1
fi

# Test frontend
if curl -f -s http://localhost:3000 > /dev/null; then
    echo "✅ Frontend health check passed"
else
    echo "❌ Frontend health check failed"
    docker-compose logs frontend | tail -10
    exit 1
fi

# Test API proxy
if curl -f -s http://localhost:3000/api/health > /dev/null; then
    echo "✅ API proxy health check passed"
else
    echo "❌ API proxy health check failed"
    echo "📋 Frontend logs:"
    docker-compose logs frontend | tail -10
    exit 1
fi

echo ""
echo "🎉 Production deployment successful!"
echo ""
echo "📋 Service Information:"
echo "   Frontend: http://localhost:3000"
echo "   Backend API: http://localhost:8080"
echo "   Health Check: http://localhost:8080/health"
echo ""
echo "📋 Available API endpoints:"
echo "   GET  /api/health              - Health check"
echo "   GET  /api/cards               - List all cards"
echo "   GET  /api/cards/search        - Search cards"
echo "   POST /api/scrape/start        - Start scraping"
echo "   GET  /api/scrape/status       - Get scrape status"
echo "   DELETE /api/database/reset    - Reset database"
echo ""
echo "📋 Management commands:"
echo "   docker-compose logs           - View logs"
echo "   docker-compose ps             - View service status"
echo "   docker-compose down           - Stop services"
echo "   docker-compose up -d          - Start services"
echo ""
echo "✅ Deployment complete!" 