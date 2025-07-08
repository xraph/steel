#!/bin/bash
# setup-cicd.sh - Automated CI/CD setup script

set -e

echo "🚀 Setting up FastRouter CI/CD..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if we're in a git repository
if [ ! -d ".git" ]; then
    print_error "This script must be run from the root of a git repository"
    exit 1
fi

# Check if GitHub CLI is installed
if ! command -v gh &> /dev/null; then
    print_warning "GitHub CLI (gh) not found. Some setup steps may require manual configuration."
    GITHUB_CLI_AVAILABLE=false
else
    GITHUB_CLI_AVAILABLE=true
fi

# Create directory structure
print_status "Creating directory structure..."
mkdir -p .github/workflows
mkdir -p .github/ISSUE_TEMPLATE
mkdir -p cmd
mkdir -p internal
mkdir -p pkg
mkdir -p examples
mkdir -p docs
mkdir -p scripts

# Create basic Go project structure if not exists
if [ ! -f "go.mod" ]; then
    print_status "Initializing Go module..."
    read -p "Enter module name (e.g., github.com/yourorg/forge-router): " MODULE_NAME
    go mod init "$MODULE_NAME"
fi

# Create basic main.go if not exists
if [ ! -f "cmd/main.go" ]; then
    print_status "Creating basic main.go..."
    cat > cmd/main.go << 'EOF'
package main

import (
    "fmt"
    "os"
)

var (
    version   = "dev"
    buildTime = "unknown"
    gitCommit = "unknown"
)

func main() {
    if len(os.Args) > 1 && os.Args[1] == "version" {
        fmt.Printf("FastRouter %s\n", version)
        fmt.Printf("Build time: %s\n", buildTime)
        fmt.Printf("Git commit: %s\n", gitCommit)
        return
    }

    if len(os.Args) > 1 && os.Args[1] == "health" {
        // Health check for Docker
        fmt.Println("OK")
        return
    }

    fmt.Println("FastRouter - High Performance HTTP Router")
    fmt.Println("Use 'version' to see version info")
}
EOF
fi

# Create .gitignore if not exists
if [ ! -f ".gitignore" ]; then
    print_status "Creating .gitignore..."
    cat > .gitignore << 'EOF'
# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with `go test -c`
*.test

# Output of the go coverage tool
*.out
coverage.html

# Dependency directories
vendor/

# Go workspace file
go.work

# IDE files
.vscode/
.idea/
*.swp
*.swo
*~

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db

# Build artifacts
dist/
build/
bin/

# Logs
*.log

# Environment files
.env
.env.local
.env.*.local

# Docker
.dockerignore

# Temporary files
tmp/
temp/
EOF
fi

# Set up GitHub repository settings if GitHub CLI is available
if [ "$GITHUB_CLI_AVAILABLE" = true ]; then
    print_status "Configuring GitHub repository settings..."

    # Check if we're connected to GitHub
    if gh auth status &> /dev/null; then
        # Enable vulnerability alerts
        gh api repos/:owner/:repo --method PATCH --field has_vulnerability_alerts=true || print_warning "Could not enable vulnerability alerts"

        # Enable automated security fixes
        gh api repos/:owner/:repo/automated-security-fixes --method PUT || print_warning "Could not enable automated security fixes"

        # Set up branch protection (optional)
        read -p "Do you want to set up branch protection for main branch? (y/N): " SETUP_PROTECTION
        if [[ $SETUP_PROTECTION =~ ^[Yy]$ ]]; then
            gh api repos/:owner/:repo/branches/main/protection --method PUT --input - << 'EOF'
{
  "required_status_checks": {
    "strict": true,
    "contexts": ["Status Check"]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": {
    "required_approving_review_count": 1,
    "dismiss_stale_reviews": true
  },
  "restrictions": null
}
EOF
        fi

        print_success "GitHub repository configured"
    else
        print_warning "Not authenticated with GitHub CLI. Run 'gh auth login' to set up repository settings."
    fi
fi

# Create initial commit if this is a new repository
if [ -z "$(git log --oneline 2>/dev/null)" ]; then
    print_status "Creating initial commit..."
    git add .
    git commit -m "feat: initial FastRouter setup with CI/CD

- Add automated CI/CD workflows
- Add security scanning and quality gates
- Add automated releases with semantic versioning
- Add Docker support
- Add comprehensive documentation"
fi

# Create development branch
print_status "Setting up Git branches..."
git checkout -b develop 2>/dev/null || git checkout develop
git push -u origin develop 2>/dev/null || print_warning "Could not push develop branch to origin"
git checkout main

print_success "CI/CD setup completed!"

echo ""
echo "📋 Next Steps:"
echo "1. Update the module name in workflow files (replace 'github.com/yourorg/forge-router')"
echo "2. Update GitHub username in dependabot.yml and workflow files"
echo "3. Add any required secrets to GitHub repository settings:"
echo "   - SLACK_WEBHOOK_URL (optional, for release notifications)"
echo "4. Enable GitHub Pages if you want documentation hosting"
echo "5. Review and customize .golangci.yml linter configuration"
echo "6. Add your actual router implementation"
echo ""
echo "🔄 To trigger your first release:"
echo "1. Make changes and commit with conventional commit messages:"
echo "   - feat: add new feature (minor version bump)"
echo "   - fix: fix bug (patch version bump)"
echo "   - feat!: breaking change (major version bump)"
echo "2. Push to main branch"
echo "3. GitHub Actions will automatically create a release"
echo ""
echo "📚 Conventional Commit Examples:"
echo "   feat(router): add middleware support"
echo "   fix(params): handle edge case in parameter parsing"
echo "   docs: update README with new examples"
echo "   perf: optimize route matching algorithm"
echo "   refactor: simplify handler registration"
echo ""
print_success "Happy coding! 🚀"