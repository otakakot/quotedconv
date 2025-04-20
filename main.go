package main

import (
	"context"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	root := getTargetPath()
	if err := processPath(ctx, root); err != nil && err != context.Canceled {
		println("Error:", err.Error())
		os.Exit(1)
	}
}

func getTargetPath() string {
	if len(os.Args) > 1 {
		return os.Args[1]
	}

	cwd, err := os.Getwd()
	if err != nil {
		println("Failed to get current directory:", err.Error())
		os.Exit(1)
	}
	return cwd
}

func processPath(ctx context.Context, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") {
				return err
			}
			if isCancelled(ctx) {
				return ctx.Err()
			}
			return fixFile(ctx, p)
		})
	}

	if strings.HasSuffix(path, ".go") {
		return fixFile(ctx, path)
	}

	println("Not a .go file or directory.")
	os.Exit(1)
	return nil
}

func fixFile(ctx context.Context, filename string) error {
	if isCancelled(ctx) {
		return ctx.Err()
	}

	src, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return err
	}

	changed := false

	ast.Inspect(file, func(n ast.Node) bool {
		if isCancelled(ctx) {
			return false
		}

		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}

		if shouldConvertLiteral(lit.Value) {
			content := lit.Value[1 : len(lit.Value)-1]
			lit.Value = strconv.Quote(content)
			changed = true
		}

		return true
	})

	if !changed {
		return nil
	}

	var buf strings.Builder
	if err := printer.Fprint(&buf, fset, file); err != nil {
		return err
	}

	formatted, err := format.Source([]byte(buf.String()))
	if err != nil {
		return err
	}

	if err := os.WriteFile(filename, formatted, 0644); err != nil {
		return err
	}

	println("Fixed:", filename)
	return nil
}

func shouldConvertLiteral(value string) bool {
	if !strings.HasPrefix(value, "`") || !strings.HasSuffix(value, "`") {
		return false
	}

	content := value[1 : len(value)-1]
	return !strings.ContainsAny(content, "\n`\\")
}

func isCancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
