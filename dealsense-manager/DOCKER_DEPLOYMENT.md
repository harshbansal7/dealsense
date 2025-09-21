# DealSense - Docker Deployment Guide

This guide explains how to deploy the complete Joinly ecosystem using Docker Compose.

## 🏗️ Architecture Overview

The Docker Compose setup orchestrates three main services:

1. **Joinly Core** (`joinly-core`) - Main AI service on port 8000
2. **Manager Backend** (`joinly-manager-backend`) - Go backend on port 8001  
3. **Manager Frontend** (`joinly-manager-frontend`) - Next.js frontend on port 3000

## 🚀 Quick Start

### 1. Environment Setup

```bash
# Copy the example environment file
cp .env.example .env

# Edit the .env file with your API keys and configuration
nano .env
```

### 2. Required API Keys

Before starting, ensure you have these API keys in your `.env` file:

```bash
# At minimum, configure one LLM provider
OPENAI_API_KEY=sk-your-openai-key
# OR
ANTHROPIC_API_KEY=sk-ant-your-anthropic-key
# OR  
GOOGLE_API_KEY=your-google-api-key

# And one voice service
ELEVENLABS_API_KEY=your-elevenlabs-key
# OR
DEEPGRAM_API_KEY=your-deepgram-key
```

### 3. Start All Services

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Check service status
docker-compose ps
```

### 4. Access the Application

- **Frontend Dashboard**: http://localhost:3000
- **Backend API**: http://localhost:8001
- **Joinly Core**: http://localhost:8000

## 🔧 Service Configuration

### Joinly Core Service

The main AI service that handles meeting participation and analysis.

**Ports**: 8000  
**Health Check**: `/health`  
**Data Volume**: `joinly_data`

### Manager Backend (Go)

RESTful API service for managing AI agents.

**Ports**: 8001  
**Health Check**: `/`  
**Volumes**: 
- `manager_data` - Agent data persistence
- `manager_logs` - Application logs

### Manager Frontend (Next.js)

Web dashboard for creating and monitoring agents.

**Ports**: 3000  
**Health Check**: `/api/health`

## 🔍 Monitoring & Health Checks

All services include health checks:

```bash
# Check all service health
docker-compose ps

# View specific service logs
docker-compose logs joinly-core
docker-compose logs joinly-manager-backend  
docker-compose logs joinly-manager-frontend

# Follow logs in real-time
docker-compose logs -f --tail=100
```

## 🛠️ Development Mode

For development with hot reload:

```bash
# Start only Joinly Core in Docker
docker-compose up joinly-core -d

# Run backend locally
cd backend_v2
go run cmd/server/main.go

# Run frontend locally (separate terminal)
cd frontend
npm run dev
```

## 📊 Volume Management

### Data Persistence

- `joinly_data`: Core service data
- `manager_data`: Agent configurations and analysis
- `manager_logs`: Application logs

### Backup Volumes

```bash
# Backup all data
docker run --rm -v joinly-manager_joinly_data:/data -v $(pwd)/backup:/backup alpine tar czf /backup/joinly_data.tar.gz -C /data .
docker run --rm -v joinly-manager_manager_data:/data -v $(pwd)/backup:/backup alpine tar czf /backup/manager_data.tar.gz -C /data .

# Restore data
docker run --rm -v joinly-manager_joinly_data:/data -v $(pwd)/backup:/backup alpine tar xzf /backup/joinly_data.tar.gz -C /data
```

## 🔒 Security Configuration

### Production Security

Update your `.env` for production:

```bash
# Use strong secrets
JWT_SECRET=your-strong-jwt-secret
NEXTAUTH_SECRET=your-strong-nextauth-secret

# Enable HTTPS
DOMAIN=your-domain.com
SSL_CERT_PATH=/etc/ssl/certs/cert.pem
SSL_KEY_PATH=/etc/ssl/private/key.pem

# Restrict CORS
CORS_ALLOWED_ORIGINS=https://your-domain.com
```

### Network Security

The compose file creates an isolated network (`joinly-network`) with subnet `172.20.0.0/16`.

## 🚨 Troubleshooting

### Common Issues

1. **Services won't start**
   ```bash
   # Check Docker and Docker Compose versions
   docker --version
   docker-compose --version
   
   # Ensure ports aren't in use
   netstat -tulpn | grep -E ':(3000|8000|8001)'
   ```

2. **API Key errors**
   ```bash
   # Verify environment variables are loaded
   docker-compose exec joinly-manager-backend env | grep API_KEY
   ```

3. **Network connectivity issues**
   ```bash
   # Check internal network connectivity
   docker-compose exec joinly-manager-backend curl http://joinly-core:8000/health
   ```

4. **Volume permission issues**
   ```bash
   # Fix volume permissions
   sudo chown -R $USER:$USER ./data ./logs
   ```

### Debug Mode

Enable debug logging:

```bash
# Update .env
LOG_LEVEL=debug
DEBUG=true

# Restart services
docker-compose restart
```

## 🔄 Updates & Maintenance

### Update Images

```bash
# Pull latest images
docker-compose pull

# Restart with new images
docker-compose up -d
```

### Clean Up

```bash
# Stop and remove containers
docker-compose down

# Remove volumes (⚠️ deletes data)
docker-compose down -v

# Remove images
docker-compose down --rmi all
```

## 📈 Scaling

### Horizontal Scaling

For production load, scale backend instances:

```bash
# Scale backend to 3 instances
docker-compose up -d --scale joinly-manager-backend=3

# Add load balancer (nginx example)
# See production-docker-compose.yml for full config
```

### Resource Limits

Add resource limits to docker-compose.yml:

```yaml
services:
  joinly-core:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 4G
        reservations:
          cpus: '1.0'
          memory: 2G
```

## 🔗 External Integrations

### Discord Webhooks

Enable Discord notifications:

```bash
# In .env
DISCORD_LOGGING_ENABLED=true
DISCORD_INFO_WEBHOOK=https://discord.com/api/webhooks/...
DISCORD_ERROR_WEBHOOK=https://discord.com/api/webhooks/...
```

### External Database

For production, use external PostgreSQL:

```bash
# In .env  
DATABASE_TYPE=postgres
DATABASE_URL=postgresql://user:password@postgres:5432/joinly
```

Add to docker-compose.yml:

```yaml
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: joinly
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
    volumes:
      - postgres_data:/var/lib/postgresql/data
```

## 📝 Environment Variables Reference

See `.env.example` for complete configuration options including:

- **AI Provider Keys**: OpenAI, Anthropic, Google, Ollama
- **Voice Services**: ElevenLabs, Deepgram  
- **Discord Integration**: Webhook URLs for logging
- **Performance Tuning**: Resource limits and timeouts
- **Security Settings**: CORS, JWT secrets, rate limiting
- **Feature Flags**: Debug modes and experimental features

---

For more detailed configuration options, see the main [README.md](./README.md) and [BACKEND_ARCHITECTURE.md](./BACKEND_ARCHITECTURE.md).