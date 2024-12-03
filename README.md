# tfvarenv

`tfvarenv` is a command-line tool designed to simplify the management of Terraform environments and tfvars files. It provides a set of commands to initialize, add, list, use, and apply Terraform environments.

## Installation

You can install `tfvarenv` using Homebrew:

```bash
brew install marcy326/tap/tfvarenv
```

## Usage

Once installed, you can use the following commands:

### Initialize Configuration

Initialize the configuration directory and file.

```bash
tfvarenv init
```

### Add Environment

Add a new environment with specific details.

```bash
tfvarenv add
```

You will be prompted to enter the environment name, S3 key, account ID, and local file path.

### List Environments

List all available environments.

```bash
tfvarenv list
```

### Use Environment

Switch to a specific environment.

```bash
tfvarenv use [environment_name]
```

### Apply Environment

Run `terraform apply` for the current environment.

```bash
tfvarenv apply
```

### Plan Environment

Run `terraform plan` for the current environment.

```bash
tfvarenv plan
```

## Configuration

The configuration is stored in a YAML file located at `.tfvarenv/config.yaml`. It contains details about the environments and the current environment in use.

## Dependencies

This tool relies on the following Go packages:

- `github.com/spf13/cobra` for command-line interface management.
- `gopkg.in/yaml.v3` for YAML parsing.
- `github.com/aws/aws-sdk-go-v2` for AWS SDK integration.

## License

This project is licensed under the MIT License. See the LICENSE file for details.

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request.

## Contact

For any questions or issues, please open an issue on the GitHub repository.
