# Token Counter

A command-line utility for counting tokens in text files using various tokenization models. This tool helps estimate token usage for large language models (LLMs) like GPT-3.5 and GPT-4.

## Features

- Count tokens in files and entire directories
- Support for different tokenization models
- Respect .gitignore rules to skip ignored files
- Detailed reports with token counts by directory and file
- Skip binary files and common non-text formats
- Filter files by minimum token count

## Installation

### From Source

1. Ensure Go is installed on your system (version 1.16 or later recommended)
2. Clone this repository
3. Build the binary:

```bash
go build -o token-counter
```

## Usage

Basic usage:

```bash
./token-counter [options] [path]
```

If no path is provided, the current directory will be analyzed.

### Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-path` | current directory | Path to the directory to analyze |
| `-model` | cl100k_base | Token counting model to use (e.g., cl100k_base for GPT-4) |
| `-gitignore` | true | Whether to respect .gitignore rules |
| `-files` | true | Whether to show individual file details |
| `-min` | 0 | Minimum token count for a file to be included |
| `-no-hidden` | true | Whether to ignore hidden files and directories (starting with .) |

### Examples

Count tokens in the current directory:

```bash
./token-counter
```

Count tokens in a specific directory:

```bash
./token-counter /path/to/project
```

Count tokens using a different model:

```bash
./token-counter -model r50k_base
```

Only show files with at least 100 tokens:

```bash
./token-counter -min 100
```

Show summary without file details:

```bash
./token-counter -files=false
```

Include hidden files and directories:

```bash
./token-counter -no-hidden=false
```

## Supported Models

- `cl100k_base` - Used by GPT-4 and GPT-3.5-Turbo
- `p50k_base` - Used by GPT-3 models like text-davinci-003
- `r50k_base` - Used by older GPT-3 models

## Output Format

The tool provides a summary of token usage including:
- Total token count for the repository
- Token count by directory (sorted by token count)
- Token count by file within each directory (if -files=true)

## License

[Add your license information here]