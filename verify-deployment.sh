#!/bin/bash

# Pre-deployment Verification Script for Sub2API Prompt Injection
# This script validates all components before server deployment
# Usage: bash verify-deployment.sh

set -e

echo "=================================="
echo "Sub2API Prompt Injection Verification"
echo "=================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

check_status() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ $1${NC}"
        return 0
    else
        echo -e "${RED}✗ $1${NC}"
        return 1
    fi
}

warn() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# 1. Check repository
echo "1. Checking Repository..."
if [ -d ".git" ]; then
    git status > /dev/null 2>&1
    check_status "Git repository found"
    
    git log --oneline -1 | grep -q "4187d8ca9\|8b1467339"
    check_status "Recent prompt injection commits found"
else
    echo -e "${RED}✗ Not in a git repository${NC}"
    exit 1
fi

# 2. Check required files
echo ""
echo "2. Checking Required Files..."
required_files=(
    "backend/ent/schema/api_key.go"
    "backend/internal/service/prompt_templates.go"
    "backend/internal/service/dynamic_prompt_builder.go"
    "backend/internal/service/prompt_injection.go"
    "backend/internal/handler/gateway_handler_chat_completions.go"
    "frontend/src/views/user/KeysView.vue"
    "Dockerfile"
    "docker-compose.prompt-injection.yml"
    ".env.pi.example"
    "deploy-prompt-injection.sh"
    "DEPLOYMENT.md"
)

for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC} $file"
    else
        echo -e "${RED}✗${NC} $file (MISSING)"
    fi
done

# 3. Verify backend code changes
echo ""
echo "3. Verifying Backend Code..."

# Check if prompt_injection_enabled field exists
if grep -q "prompt_injection_enabled" backend/ent/schema/api_key.go; then
    check_status "prompt_injection_enabled field in schema"
else
    echo -e "${RED}✗ prompt_injection_enabled field missing${NC}"
fi

# Check handler integration
if grep -q "BuildDynamicPrompt\|GetPromptInjectionEnabled" backend/internal/handler/gateway_handler_chat_completions.go; then
    check_status "Handler integration for prompt injection"
else
    echo -e "${RED}✗ Handler integration missing${NC}"
fi

# Check prompt templates
line_count=$(wc -l < backend/internal/service/prompt_templates.go)
if [ "$line_count" -gt 2000 ]; then
    check_status "Prompt templates file size ($(($line_count)) lines, expected ~2112)"
else
    echo -e "${YELLOW}⚠ Prompt templates file size might be incorrect ($line_count lines)${NC}"
fi

# 4. Verify frontend changes
echo ""
echo "4. Verifying Frontend Code..."

if grep -q "prompt_injection_enabled\|promptInjectionEnabled" frontend/src/views/user/KeysView.vue; then
    check_status "KeysView.vue prompt injection support"
else
    echo -e "${RED}✗ KeysView.vue missing prompt injection support${NC}"
fi

if grep -q "Prompt Injection" frontend/src/i18n/locales/en/dashboard.ts; then
    check_status "English i18n translations"
else
    echo -e "${YELLOW}⚠ English i18n translations might be incomplete${NC}"
fi

if grep -q "提示词注入\|动态提示词" frontend/src/i18n/locales/zh/dashboard.ts; then
    check_status "Chinese i18n translations"
else
    echo -e "${YELLOW}⚠ Chinese i18n translations might be incomplete${NC}"
fi

# 5. Verify Docker configuration
echo ""
echo "5. Verifying Docker Configuration..."

if grep -q "prompt-injection\|_pi" docker-compose.prompt-injection.yml; then
    check_status "Docker Compose prompt injection configuration"
else
    echo -e "${RED}✗ Docker Compose configuration incomplete${NC}"
fi

if grep -q "port 8081\|8081:" docker-compose.prompt-injection.yml; then
    check_status "Port 8081 configured for test environment"
else
    echo -e "${RED}✗ Port 8081 not configured${NC}"
fi

# 6. Verify deployment script
echo ""
echo "6. Verifying Deployment Script..."

if [ -f "deploy-prompt-injection.sh" ]; then
    if grep -q "docker compose\|docker-compose" deploy-prompt-injection.sh; then
        check_status "Deployment script has Docker commands"
    else
        echo -e "${RED}✗ Deployment script missing Docker commands${NC}"
    fi
    
    if [ -x "deploy-prompt-injection.sh" ]; then
        check_status "Deployment script is executable"
    else
        warn "Deployment script is not executable (will need chmod +x)"
    fi
else
    echo -e "${RED}✗ Deployment script not found${NC}"
fi

# 7. Environment template
echo ""
echo "7. Verifying Environment Template..."

if [ -f ".env.pi.example" ]; then
    check_status ".env.pi.example file exists"
    
    required_env_vars=(
        "POSTGRES_PASSWORD_PI"
        "REDIS_PASSWORD_PI"
        "DATABASE_URL"
        "REDIS_URL"
        "SESSION_SECRET"
        "JWT_SECRET"
    )
    
    for var in "${required_env_vars[@]}"; do
        if grep -q "$var" .env.pi.example; then
            echo -e "${GREEN}✓${NC} $var in template"
        else
            echo -e "${YELLOW}⚠${NC} $var might be missing from template"
        fi
    done
else
    echo -e "${RED}✗ .env.pi.example not found${NC}"
fi

# 8. Summary
echo ""
echo "=================================="
echo "Verification Summary"
echo "=================================="
echo ""
echo "Pre-deployment checks completed. Next steps:"
echo ""
echo "1. On your server, run:"
echo "   git clone git@github.com:JarJiang/sub2api-jrw.git"
echo "   cd sub2api-jrw"
echo ""
echo "2. Configure environment:"
echo "   cp .env.pi.example .env.pi"
echo "   nano .env.pi  # Edit with secure passwords"
echo ""
echo "3. Deploy:"
echo "   chmod +x deploy-prompt-injection.sh"
echo "   ./deploy-prompt-injection.sh"
echo ""
echo "4. Verify deployment using SERVER_DEPLOYMENT_CHECKLIST.md"
echo ""
