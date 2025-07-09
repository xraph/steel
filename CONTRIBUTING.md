# CONTRIBUTING.md
# Contributing to ForgeRouter

Thank you for your interest in contributing to ForgeRouter! This document provides guidelines and information about contributing to the project.

## Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the issue list to see if the bug has already been reported. When creating a bug report, please include as many details as possible:

- Use a clear and descriptive title
- Describe the exact steps to reproduce the problem
- Provide specific examples to demonstrate the steps
- Describe the behavior you observed and what behavior you expected
- Include details about your configuration and environment

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, please:

- Use a clear and descriptive title
- Provide a detailed description of the suggested enhancement
- Provide specific examples to illustrate the enhancement
- Explain why this enhancement would be useful

### Pull Requests

1. Fork the repository
2. Create a new branch from `main` for your feature or bug fix
3. Make your changes
4. Add or update tests as necessary
5. Ensure all tests pass
6. Update documentation as needed
7. Submit a pull request

## Development Setup

### Prerequisites

- Go 1.20 or later
- Git
- Make (optional, for convenience commands)

### Setup

  ```bash
  # Clone your fork
  git clone https://github.com/yourusername/forge-router.git
  cd forge-router
  
  # Add upstream remote
  git remote add upstream https://github.com/xraph/forgerouter.git
  
  # Install dependencies
  go mod download
  
  # Run tests
  go test ./...
  
  # Run linter
  golangci-lint run
  ```

## Development Guidelines

### Code Style

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` to format your code
- Use `golangci-lint` to check for common issues
- Write clear, self-documenting code
- Add comments for complex logic

### Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/) specification:

  ```
<type>[optional scope]: <description>
  
  [optional body]
  
  [optional footer(s)]
  ```

Types:
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `perf`: A code change that improves performance
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to the build process or auxiliary tools

Examples:
  ```
feat(router): add support for middleware groups
fix(params): handle empty parameter values correctly
docs: update API documentation for new features
  ```

### Testing

- Write unit tests for new functionality
- Maintain or improve test coverage
- Include integration tests for complex features
- Add benchmark tests for performance-critical code

### Documentation

- Update README.md if you add new features
- Add inline code documentation for public APIs
- Update examples if the API changes
- Consider adding tutorials for complex features

## Project Structure

  ```
  forge-router/
  ├── cmd/                 # Command-line applications
  ├── internal/           # Private application code
  ├── pkg/               # Public library code
  ├── examples/          # Example applications
  ├── docs/             # Documentation
  ├── .github/          # GitHub workflows and templates
  ├── benchmark/        # Benchmark tests
  └── scripts/          # Build and utility scripts
  ```

## Performance Considerations

ForgeRouter is designed for high performance. When contributing:

- Avoid unnecessary allocations
- Use object pooling where appropriate
- Benchmark performance-critical code
- Consider memory usage and garbage collection impact
- Profile code when making performance claims

## Security

- Never commit sensitive information
- Follow secure coding practices
- Report security vulnerabilities privately
- Consider security implications of changes

## Release Process

Releases are automated through GitHub Actions:

1. Commits to `main` trigger automatic versioning based on conventional commits
2. Semantic versioning is used (MAJOR.MINOR.PATCH)
3. Releases include:
- Binary artifacts for multiple platforms
- Docker images
- GitHub release with changelog
- Updated documentation

## Getting Help

- Create a [GitHub Discussion](https://github.com/xraph/forgerouter/discussions) for questions
- Check existing [issues](https://github.com/xraph/forgerouter/issues) and [pull requests](https://github.com/xraph/forgerouter/pulls)
- Read the [documentation](https://github.com/xraph/forgerouter/wiki)

## Recognition

Contributors will be recognized in:
- CONTRIBUTORS.md file
- GitHub contributors page
- Release notes for significant contributions

Thank you for contributing to ForgeRouter! 🚀