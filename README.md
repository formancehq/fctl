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
# Create or authenticate a named Cloud profile
fctl --profile <name> login
# Select an existing profile
fctl profiles use <name>
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
- **Service Commands**: Commands for the services listed above

## Configuration

Configuration is stored in `~/.config/formance/fctl` by default. You can specify a
different directory using `--config-dir` or `-c`. The directory contains
`config.yml` and `profiles/<name>/profile.json`.

Service access currently uses Cloud profiles authenticated through Membership.
Direct endpoints and unauthenticated local services are not supported yet.

## Options

- `--profile, -p`: Configuration profile to use
- `--config-dir, -c`: Path to configuration directory
- `--debug, -d`: Enable debug mode
- `--output, -o`: Output format (plain, json)
- `--insecure-tls`: Allow insecure TLS connections
- `--telemetry`: Enable telemetry

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Links

- [Repository technical documentation](docs/README.md)
- [Formance Documentation](https://docs.formance.com)
- [Formance GitHub](https://github.com/formancehq)
