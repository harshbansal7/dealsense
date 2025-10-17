#!/bin/bash
# DealSense Deployment Script
# This script helps deploy both Joinly Core and DealSense Backend using Docker Compose

set -e

echo "🚀 DealSense Deployment Script"
echo "================================"

# Check if required .env files exist
if [ ! -f ".env" ]; then
    echo "❌ Error: Root .env file not found!"
    echo "Please create .env in the project root for Joinly Core."
    echo "See README.md for configuration details."
    exit 1
fi

if [ ! -f "dealsense-manager/backend_v2/.env" ]; then
    echo "❌ Error: Backend .env file not found!"
    echo "Please create dealsense-manager/backend_v2/.env for DealSense backend."
    echo "See README.md for configuration details."
    exit 1
fi

# Check if docker and docker compose are available
if ! command -v docker &> /dev/null; then
    echo "❌ Error: Docker is not installed or not in PATH"
    exit 1
fi

if ! command -v docker compose &> /dev/null; then
    echo "❌ Error: Docker Compose is not installed or not in PATH"
    exit 1
fi

echo "✅ Prerequisites check passed"

# Function to show usage
show_usage() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  start     - Start all services (default)"
    echo "  stop      - Stop all services"
    echo "  restart   - Restart all services"
    echo "  logs      - Show logs from all services"
    echo "  status    - Show status of all services"
    echo "  clean     - Stop services and remove containers/volumes"
    echo ""
    echo "Examples:"
    echo "  ./deploy.sh              # Start services"
    echo "  ./deploy.sh logs         # View logs"
    echo "  ./deploy.sh stop         # Stop services"
}

# Default command is start
COMMAND=${1:-start}

case $COMMAND in
    start)
        echo "🏗️  Building and starting services..."
        docker compose up -d --build
        echo ""
        echo "✅ Services started!"
        echo ""
        echo "🌐 Service URLs:"
        echo "   Joinly Core:     http://localhost:8000"
        echo "   DealSense API:   http://localhost:8001"
        echo "   Health Check:    http://localhost:8001/health"
        echo ""
        echo "📁 Environment Files:"
        echo "   Root .env:       Used by Joinly Core"
        echo "   Backend .env:    dealsense-manager/backend_v2/.env"
        echo ""
        echo "📊 Check status with: ./deploy.sh status"
        echo "📋 View logs with:    ./deploy.sh logs"
        ;;

    stop)
        echo "🛑 Stopping services..."
        docker compose down
        echo "✅ Services stopped"
        ;;

    restart)
        echo "🔄 Restarting services..."
        docker compose restart
        echo "✅ Services restarted"
        ;;

    logs)
        echo "📋 Showing service logs..."
        docker compose logs -f
        ;;

    status)
        echo "📊 Service Status:"
        docker compose ps
        echo ""
        echo "🏥 Health Checks:"
        echo "Joinly Core:"
        curl -s http://localhost:8000/health || echo "  ❌ Not responding"
        echo "DealSense Backend:"
        curl -s http://localhost:8001/health || echo "  ❌ Not responding"
        ;;

    clean)
        echo "🧹 Cleaning up services and volumes..."
        docker compose down -v --remove-orphans
        echo "✅ Cleanup complete"
        ;;

    *)
        echo "❌ Unknown command: $COMMAND"
        echo ""
        show_usage
        exit 1
        ;;
esac
