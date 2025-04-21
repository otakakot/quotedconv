# QuotedConv

This project provides a tool for converting backtick-quoted raw string literals into interpreted string literals (double-quoted strings) based on specific criteria.

## Conversion Rules

The tool converts a raw string literal (i.e. a backtick-quoted string) to an interpreted string literal (double-quoted) only if all of the following conditions are met:

- **No Newlines:** The string literal does not contain any newline characters.
- **No Backticks:** The literal does not contain any additional backtick characters.
- **No Backslashes:** The literal does not include any backslashes.
- **No Double Quotes:** The literal does not contain any double quote characters.
- **Not a Struct Tag:** The literal is not part of a struct tag (this is determined via syntactic analysis of the Go AST).

String literals that do not satisfy these conditions remain unchanged.

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
