# Formance Control CLI (fctl)

Command-line interface for managing and interacting with Formance services.

## Overview

`fctl` is the official CLI tool for Formance, providing a comprehensive set of commands to interact with Formance services, including:

- Ledger management
- Payments processing
- Wallets
- Reconciliation
- Orchestration
- Authentication
- Stack management
- Cloud resources
- Search functionality
- Webhooks configuration

## Installation

### Using Homebrew (macOS/Linux)

```bash
brew install formancehq/tap/fctl
```

### Manual Installation

Download the latest binary for your platform from the [Releases page](https://github.com/formancehq/fctl/releases).

## Getting Started

### Authentication

```bash
# Login to Formance
fctl login

# Configure profiles
fctl profiles list
fctl profiles create <name> --endpoint <endpoint>
```

### Basic Usage

```bash
# Get version information
fctl version

# Get help for any command
fctl --help
fctl <command> --help

# Use the interactive mode
fctl prompt
```

## Features

- **Multiple Output Formats**: Support for plain text and JSON output
- **Profile Management**: Create and switch between different configuration profiles
- **Interactive Mode**: Use the prompt mode for interactive command execution
- **Comprehensive API Coverage**: Access to all Formance services and features

## Configuration

Configuration is stored in `~/.formance/fctl.config` by default. You can specify a different configuration file using the `-c` flag.

## Options

- `--profile, -p`: Configuration profile to use
- `--config, -c`: Path to configuration file
- `--debug, -d`: Enable debug mode
- `--output, -o`: Output format (plain, json)
- `--insecure-tls`: Allow insecure TLS connections
- `--telemetry`: Enable telemetry

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Links

- [Formance Documentation](https://docs.formance.com)
- [Formance GitHub](https://github.com/formancehq) 