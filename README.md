# tfvarenv: Terraform Environment and Variables Management CLI

## Overview

`tfvarenv` is a powerful CLI tool designed to simplify the management of Terraform environments and `tfvars` files. It provides a comprehensive solution for versioning, tracking, and deploying Terraform configurations across multiple environments.

## Features

- **Environment Management**
  - Initialize and manage multiple Terraform environments
  - Store and version tfvars files in S3 with versioning
  - Track deployment history
  - Manage AWS-based backend configurations

- **Version Control**
  - Upload and download tfvars files
  - List and track version history
  - Filter and search versions
  - Store version metadata

- **Deployment Workflow**
  - Plan and apply Terraform configurations
  - Automatic backup of tfvars files
  - Deployment approval mechanisms
  - Tracking of deployment status

## Prerequisites

- Terraform installed
- AWS CLI configured
- AWS S3 bucket with versioning enabled

## Installation

You can install `tfvarenv` using Homebrew:

```bash
brew install marcy326/tap/tfvarenv
```

## Configuration

### Initialize Project

```bash
# Initialize tfvarenv in your project
tfvarenv init
```

### Add Environment

```bash
# Add a new environment
tfvarenv add
```

Interactive prompts will guide you through:
- Environment name and description
- S3 bucket configuration
- AWS region and account details
- Local tfvars file path
- Deployment settings

## Commands

### Environment Management
- `tfvarenv init`: Initialize tfvarenv configuration
- `tfvarenv list`: List all environments
- `tfvarenv add`: Add a new environment
- `tfvarenv use [environment]`: Switch to a specific environment

### Version Management
- `tfvarenv versions [environment]`: List available versions
- `tfvarenv upload [environment]`: Upload local tfvars to S3
- `tfvarenv download [environment]`: Download tfvars from S3

### Terraform Workflow
- `tfvarenv plan [environment]`: Run terraform plan
- `tfvarenv apply [environment]`: Run terraform apply
- `tfvarenv history [environment]`: View deployment history

## Advanced Usage

### Version Filtering
```bash
# List versions since a specific date
tfvarenv versions prod --since 2024-01-01

# Show only deployed versions
tfvarenv versions dev --deployed-only

# Limit version output
tfvarenv versions staging --limit 5
```

### Deployment Options
```bash
# Plan with a specific remote version
tfvarenv plan prod --remote --version-id abc123

# Apply with additional Terraform options
tfvarenv apply dev --options "-refresh=false"
```

## Environment Configuration

The `.tfvarenv.json` file contains:
- Default AWS region
- Environment details
- S3 bucket configurations
- Local and remote tfvars paths

## Security Considerations

- Requires AWS credentials with appropriate S3 and STS permissions
- Supports environment-specific approval workflows
- Automatic backup of tfvars files

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

MIT License

Distributed under the MIT License. See `LICENSE` file for more information.

## Support

For issues, feature requests, or questions, please [open an issue](https://github.com/marcy326/tfvarenv/issues) on GitHub.
