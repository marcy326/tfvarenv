# tfvarenv

`tfvarenv` is a powerful CLI tool designed to simplify and enhance Terraform environment management, focusing on version control and secure handling of tfvars files.

## Features

- **Secure Version Management**: Manage Terraform variables (tfvars) as versioned files in S3
- **Environment Configuration**: Create and manage multiple Terraform environments
- **AWS Integration**: Seamless AWS credentials and account management
- **Deployment Tracking**: Track deployment history and version changes
- **Flexible Workflows**: Support for local and remote tfvars files

## Prerequisites

- Terraform installed
- AWS CLI configured
- AWS S3 bucket with versioning enabled

## Installation

You can install `tfvarenv` using Homebrew:

```bash
brew install marcy326/tap/tfvarenv
```

## Quick Start

### 1. Initialize tfvarenv

```bash
tfvarenv init
```

This creates a `.tfvarenv.json` configuration file and sets up your default AWS region.

### 2. Add an Environment

```bash
tfvarenv add
```

Follow the interactive prompts to:
- Define environment name
- Configure S3 bucket
- Set up local tfvars path
- Configure deployment settings

### 3. Upload Local tfvars

```bash
tfvarenv upload dev
```

### 4. Plan and Apply Terraform Changes

```bash
# Local plan
tfvarenv plan dev

# Remote plan (using S3 version)
tfvarenv plan dev --remote

# Apply changes
tfvarenv apply dev
```

## Key Commands

- `tfvarenv init`: Initialize project configuration
- `tfvarenv add`: Add a new environment
- `tfvarenv list`: List all environments
- `tfvarenv upload`: Upload local tfvars to S3
- `tfvarenv download`: Download tfvars from S3
- `tfvarenv plan`: Run Terraform plan
- `tfvarenv apply`: Apply Terraform changes
- `tfvarenv versions`: List version history
- `tfvarenv history`: Show deployment history

## Configuration

### .tfvarenv.json Structure

```json
{
  "version": "1.0",
  "default_region": "us-east-1",
  "environments": {
    "dev": {
      "name": "dev",
      "description": "Development environment",
      "s3": {
        "bucket": "my-terraform-bucket",
        "prefix": "terraform/dev",
        "tfvars_key": "terraform.tfvars"
      },
      "aws": {
        "account_id": "123456789012",
        "region": "us-east-1"
      },
      "local": {
        "tfvars_path": "envs/dev/terraform.tfvars"
      },
      "deployment": {
        "auto_backup": true,
        "require_approval": false
      }
    }
  }
}
```

## Security Considerations

- S3 bucket must have versioning enabled
- Supports AWS credentials via environment variables
- Optional deployment approval workflow
- Automatic backup of tfvars files

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

[Specify your license, e.g., MIT]

## Troubleshooting

- Ensure AWS CLI is configured
- Check IAM permissions for S3 and STS
- Verify Terraform is installed
- Review `.tfvarenv.json` configuration

## Todo

- [ ] Add support for multiple cloud providers
- [ ] Implement more advanced version filtering
- [ ] Create comprehensive test suite

## Contact

[Your contact information or project maintainer details]