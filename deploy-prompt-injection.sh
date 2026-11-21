#!/bin/bash
set -e

echo "========================================="
echo "Sub2API Prompt Injection Test Deployment"
echo "========================================="
echo ""

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if .env.pi exists
if [ ! -f ".env.pi" ]; then
    echo -e "${RED}Error: .env.pi not found${NC}"
    echo "Please copy .env.pi.example to .env.pi and configure it:"
    echo "  cp .env.pi.example .env.pi"
    echo "  nano .env.pi  # or vim/vi"
    exit 1
fi

echo -e "${GREEN}Step 1: Creating required directories...${NC}"
mkdir -p data_pi postgres_data_pi redis_data_pi
echo "✓ Directories created: data_pi, postgres_data_pi, redis_data_pi"
echo ""

echo -e "${GREEN}Step 2: Checking Docker network...${NC}"
if ! docker network ls | grep -q sub2api-network; then
    echo "Creating sub2api-network..."
    docker network create sub2api-network
    echo "✓ Network created"
else
    echo "✓ Network sub2api-network already exists"
fi
echo ""

echo -e "${GREEN}Step 3: Building Docker image...${NC}"
echo "This may take several minutes..."
docker build -t sub2api:prompt-injection .
echo "✓ Image built: sub2api:prompt-injection"
echo ""

echo -e "${GREEN}Step 4: Starting services...${NC}"
docker-compose -f docker-compose.prompt-injection.yml --env-file .env.pi up -d
echo "✓ Services started"
echo ""

echo -e "${GREEN}Step 5: Waiting for services to be healthy...${NC}"
sleep 5
echo ""

echo -e "${YELLOW}Service Status:${NC}"
docker-compose -f docker-compose.prompt-injection.yml ps
echo ""

echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}Deployment Complete!${NC}"
echo -e "${GREEN}=========================================${NC}"
echo ""
echo "Test Stack URLs:"
echo "  - API: http://localhost:8081"
echo "  - Health: http://localhost:8081/health"
echo ""
echo "Production Stack (should still be running):"
echo "  - API: http://localhost:8080"
echo ""
echo "Useful commands:"
echo "  - View logs: docker-compose -f docker-compose.prompt-injection.yml logs -f"
echo "  - Stop: docker-compose -f docker-compose.prompt-injection.yml down"
echo "  - Restart: docker-compose -f docker-compose.prompt-injection.yml restart"
echo "  - Remove all: docker-compose -f docker-compose.prompt-injection.yml down -v"
echo ""
