# QuotedConv

QuotedConv is a command-line tool written in Go for processing Go source files. It automatically converts eligible raw string literals (backtick-quoted strings) into interpreted string literals (double-quoted strings) to enforce consistent formatting and simplify code maintenance.

## Features

- **AST-Based Processing:**  
  Uses Go's parser and abstract syntax tree (AST) to reliably locate and transform raw string literals in Go source code.

- **Selective Conversion:**  
  Converts raw string literals only if they meet specific criteria: the string must be single-line and must not contain newline characters, backticks, or escape sequences.

- **Recursive Directory Traversal:**  
  Can process a single Go file or recursively traverse directories to update all `.go` files.

- **Graceful Interruption:**  
  Implements context cancellation (via `os.Interrupt`) to allow the operation to be stopped cleanly during long-running tasks.

- **Automatic Code Formatting:**  
  After modifying a file, the tool formats the code using Go's `go/format` package to ensure adherence to Go coding standards.

## Getting Started

### Prerequisites

- Go 1.16 or later

### Installation

- **Install the Tool:**

   ```bash
   go install github.com/otakakot/quotedconv@latest
   ```

## Usage

- **Process a Single File:**

  ```bash
  quotedconv /path/to/file.go
  ```

- **Process a Directory:**

  ```bash
  quotedconv /path/to/directory
  ```

If no target path is provided, the tool defaults to the current directory.

## How It Works

1. **File Detection:**  
   The tool determines whether the provided path is a file or a directory. If a directory, it recursively inspects all subdirectories for `.go` files.

2. **Parsing and Transformation:**  
   Each Go file is parsed into an AST. The tool then inspects the AST for raw string literals (`\``...`\``) and checks if they should be converted. Eligible literals are replaced with their properly quoted equivalent using Go’s standard library functions.

3. **Formatting and Saving:**  
   Once transformations are applied, the modified code is formatted using `go/format` and written back to the original file.

4. **Interruption Handling:**  
   The tool listens for interrupt signals (e.g., Ctrl+C) and cancels ongoing operations gracefully.

## Error Handling

- On encountering critical errors (such as file read or parse failures), the tool will panic with an error message.
- If the process is interrupted while running, it stops processing further files without leaving partially updated files.

## Contributing

Contributions are welcome! To contribute:

1. Fork the repository.
2. Create a feature branch.
3. Commit your changes with clear, descriptive messages.
4. Submit a pull request outlining your changes.

## License

This project is licensed under the terms of the [MIT License](LICENSE).

## Acknowledgements

Thanks to the Go community and all contributors for their support and inspiration. This tool builds upon best practices in Go development and leverages Go’s powerful AST capabilities to simplify code maintenance.
