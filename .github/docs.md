# Steel CI/CD Setup Guide

This guide provides a complete CI/CD setup with automatic semantic versioning for the Steel project.

## 🚀 Features

- **Automated Testing** - Multi-platform testing across Go versions
- **Security Scanning** - CodeQL, Trivy, Gosec, Nancy vulnerability scanning
- **Quality Gates** - Linting, formatting, coverage reporting
- **Automatic Releases** - Semantic versioning based on conventional commits
- **Multi-Platform Builds** - Binaries for Linux, macOS, Windows, FreeBSD
- **Docker Images** - Automated container builds and publishing
- **Documentation** - Automated docs building and link checking
- **Dependency Management** - Automated dependency updates

## 📁 File Structure

```
.github/
├── workflows/
│   ├── ci.yml              # Main CI pipeline
│   ├── release.yml         # Release automation
│   ├── security.yml        # Security scanning
│   ├── dependencies.yml    # Dependency updates
│   └── docs.yml           # Documentation
├── ISSUE_TEMPLATE/
│   ├── bug_report.yml
│   ├── feature_request.yml
│   ├── performance.yml
│   ├── documentation.yml
│   └── config.yml
├── pull_request_template.md
├── release.yml             # Release notes config
├── dependabot.yml          # Dependabot config
├── mlc_config.json        # Link checker config
└── typos.toml             # Spell checker config

# Project root
├── .golangci.yml          # Linter configuration
├── Dockerfile             # Container definition
├── CONTRIBUTING.md        # Contribution guidelines
└── setup-cicd.sh         # Setup script
```

## 🛠️ Quick Setup

### 1. Run the Setup Script

```bash
#!/bin/bash
# setup-cicd.sh - Automated CI/CD setup script

set -e

echo "🚀 Setting up Steel CI/CD..."

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
    read -p "Enter module name (e.g., github.com/xraph/steel): " MODULE_NAME
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
        fmt.Printf("Steel %s\n", version)
        fmt.Printf("Build time: %s\n", buildTime)
        fmt.Printf("Git commit: %s\n", gitCommit)
        return
    }
    
    if len(os.Args) > 1 && os.Args[1] == "health" {
        // Health check for Docker
        fmt.Println("OK")
        return
    }
    
    fmt.Println("Steel - High Performance HTTP Router")
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
    git commit -m "feat: initial Steel setup with CI/CD

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
echo "1. Update the module name in workflow files (replace 'github.com/yourorg/steel')"
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
```

### 2. Repository Configuration

#### Required Repository Settings

1. **Enable GitHub Actions**
    - Go to Settings → Actions → General
    - Allow all actions and reusable workflows

2. **Enable Dependency Graph**
    - Go to Settings → Security & analysis
    - Enable dependency graph

3. **Enable Vulnerability Alerts**
    - Enable Dependabot alerts
    - Enable Dependabot security updates

4. **Branch Protection** (Recommended)
   ```bash
   # Using GitHub CLI
   gh api repos/:owner/:repo/branches/main/protection --method PUT --input protection.json
   ```

#### Optional Secrets

Add these in Settings → Secrets → Actions:

- `SLACK_WEBHOOK_URL` - For release notifications
- `CODECOV_TOKEN` - For enhanced coverage reporting
- Custom registry tokens if using private registries

### 3. Workflow Triggers

#### Automatic Triggers

- **CI Workflow**: Every push and PR to main/develop
- **Release Workflow**: Push to main with conventional commits
- **Security Workflow**: Weekly + every push to main
- **Dependencies Workflow**: Weekly on Mondays

#### Manual Triggers

- **Release Workflow**: Manual trigger with version override
- **Dependencies Workflow**: Manual dependency updates

## 📈 Semantic Versioning

### Conventional Commit Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Version Bumping Rules

| Commit Type | Version Bump | Example |
|-------------|--------------|---------|
| `feat` | Minor (0.1.0 → 0.2.0) | `feat: add middleware support` |
| `fix` | Patch (0.1.0 → 0.1.1) | `fix: handle nil parameters` |
| `feat!` or `BREAKING CHANGE` | Major (0.1.0 → 1.0.0) | `feat!: change API structure` |
| `docs`, `style`, `refactor`, `test`, `chore` | No release | Documentation updates |

### Examples

```bash
# Feature addition (minor bump)
git commit -m "feat(router): add WebSocket support

Add support for WebSocket connections with automatic upgrading
and message handling capabilities."

# Bug fix (patch bump)
git commit -m "fix(params): handle empty parameter values

Fixed issue where empty path parameters caused panic.
Now returns empty string for missing parameters.

Fixes #123"

# Breaking change (major bump)
git commit -m "feat!: redesign middleware API

BREAKING CHANGE: Middleware signature changed from
func(http.Handler) http.Handler to func(Context) error.

Migration guide available in MIGRATION.md"

# Performance improvement (patch bump)
git commit -m "perf(routing): optimize route matching

Improved route lookup performance by 25% through
better tree traversal algorithm."
```

## 🔒 Security Features

### Automated Security Scanning

- **CodeQL**: Static analysis for vulnerabilities
- **Trivy**: Container and filesystem vulnerability scanning
- **Gosec**: Go-specific security issues
- **Nancy**: Dependency vulnerability checking

### Security Best Practices

- Regular dependency updates via Dependabot
- SARIF upload for security findings
- Branch protection with required status checks
- Signed commits (recommended)

## 🐳 Docker Support

### Multi-stage Build

The included Dockerfile provides:
- Multi-stage build for minimal image size
- Non-root user for security
- Health check endpoint
- Version information injection

### Usage

```bash
# Build image
docker build -t steel:latest .

# Run container
docker run -p 8080:8080 steel:latest

# Health check
docker run steel:latest health
```

## 📊 Monitoring and Observability

### Metrics Included

- **Build Metrics**: Build time, success rate
- **Test Metrics**: Coverage, test results
- **Security Metrics**: Vulnerability counts
- **Performance Metrics**: Benchmark results

### GitHub Insights

- **Actions**: Workflow run history and performance
- **Security**: Vulnerability alerts and patches
- **Dependencies**: Dependency graph and updates
- **Code Quality**: Coverage reports and trends

## 🚀 Release Process

### Automatic Releases

1. **Commit Analysis**: Scan commits since last tag
2. **Version Calculation**: Determine version bump type
3. **Artifact Building**: Multi-platform binaries and Docker images
4. **Release Creation**: GitHub release with changelog
5. **Notification**: Optional Slack/Discord notifications

### Manual Releases

```bash
# Trigger manual release
gh workflow run release.yml -f release_type=patch
gh workflow run release.yml -f release_type=minor
gh workflow run release.yml -f release_type=major
```

## 🛠️ Development Workflow

### Recommended Workflow

1. **Create Feature Branch**
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make Changes**
    - Follow conventional commit messages
    - Add tests for new functionality
    - Update documentation

3. **Create Pull Request**
    - CI automatically runs tests and checks
    - Security scanning performed
    - Code coverage reported

4. **Review and Merge**
    - Required approvals (if configured)
    - All checks must pass
    - Automatic merge to main triggers release

### Quality Gates

All PRs must pass:
- ✅ Unit tests across Go versions and platforms
- ✅ Linting and formatting checks
- ✅ Security vulnerability scanning
- ✅ Code coverage requirements
- ✅ Dependency vulnerability checks

## 📚 Additional Resources

- [Conventional Commits](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [golangci-lint](https://golangci-lint.run/)

## 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed contribution guidelines.

## 📄 License

This CI/CD setup is provided under the same license as Steel.