// Package main provides a tool for processing Go source files to convert raw string literals
// (backtick-quoted strings) into interpreted string literals (double-quoted strings) if they
// meet specific criteria. It traverses directories or processes individual files, making
// modifications in place while ensuring proper formatting and syntax.
package main

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type collectorError struct {
	mu     sync.Mutex
	errors []error
}

func (ec *collectorError) Add(err error) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.errors = append(ec.errors, err)
}

func (ec *collectorError) HasErrors() bool {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	return len(ec.errors) > 0
}

func (ec *collectorError) Error() string {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	errStrings := make([]string, 0, len(ec.errors))

	for _, err := range ec.errors {
		errStrings = append(errStrings, err.Error())
	}

	return strings.Join(errStrings, "\n")
}

type workerPool struct {
	wg             sync.WaitGroup
	jobChan        chan string
	numWorkers     int
	ctx            context.Context
	collectorError *collectorError
	processedFiles int32
}

func newWorkerPool(ctx context.Context, numWorkers int) *workerPool {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	const chanSize = 2

	return &workerPool{
		wg:         sync.WaitGroup{},
		jobChan:    make(chan string, numWorkers*chanSize),
		numWorkers: numWorkers,
		ctx:        ctx,
		collectorError: &collectorError{
			mu:     sync.Mutex{},
			errors: []error{},
		},
		processedFiles: 0,
	}
}

func (wp *workerPool) Start() {
	for range wp.numWorkers {
		wp.wg.Add(1)

		go func() {
			defer wp.wg.Done()

			for filePath := range wp.jobChan {
				if isCancelled(wp.ctx) {
					return
				}

				err := fixFile(wp.ctx, filePath)
				if err != nil && !errors.Is(err, context.Canceled) {
					wp.collectorError.Add(fmt.Errorf("error processing file %s: %w", filePath, err))
				} else if err == nil {
					atomic.AddInt32(&wp.processedFiles, 1)
				}
			}
		}()
	}
}

func (wp *workerPool) AddJob(filePath string) {
	wp.jobChan <- filePath
}

func (wp *workerPool) Wait() {
	close(wp.jobChan)
	wp.wg.Wait()
}

func (wp *workerPool) GetProcessedCount() int {
	return int(atomic.LoadInt32(&wp.processedFiles))
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	root := getTargetPath()

	numWorkers := runtime.NumCPU()

	if err := processPath(ctx, root, numWorkers); err != nil && !errors.Is(err, context.Canceled) {
		panic("Error: " + err.Error())
	}
}

func getTargetPath() string {
	if len(os.Args) > 1 {
		return os.Args[1]
	}

	cwd, err := os.Getwd()
	if err != nil {
		panic("Failed to get current directory. Error: " + err.Error())
	}

	return cwd
}

func processPath(ctx context.Context, path string, numWorkers int) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat path: %w", err)
	}

	if info.IsDir() {
		files := []string{}

		if err = filepath.WalkDir(path, func(pathStr string, dir fs.DirEntry, err error) error {
			if err != nil {
				return fmt.Errorf("walking directory: %w", err)
			}

			if dir.IsDir() || !strings.HasSuffix(pathStr, ".go") {
				return nil
			}

			if isCancelled(ctx) {
				return fmt.Errorf("context error: %w", ctx.Err())
			}

			files = append(files, pathStr)

			return nil
		}); err != nil {
			return fmt.Errorf("walking directory: %w", err)
		}

		pool := newWorkerPool(ctx, numWorkers)

		pool.Start()

		for _, file := range files {
			if isCancelled(ctx) {
				break
			}

			pool.AddJob(file)
		}

		pool.Wait()

		log.Printf("Successfully processed %d files", pool.GetProcessedCount())

		if pool.collectorError.HasErrors() {
			return fmt.Errorf("errors occurred during processing: %w", pool.collectorError)
		}

		return nil
	}

	if strings.HasSuffix(path, ".go") {
		return fixFile(ctx, path)
	}

	log.Println("Not a .go file or directory.")

	os.Exit(1)

	return nil
}

func fixFile(ctx context.Context, filename string) error {
	if isCancelled(ctx) {
		return fmt.Errorf("context error: %w", ctx.Err())
	}

	src, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	file, fset, err := parseGoFile(filename, src)
	if err != nil {
		return err
	}

	changed := processAST(ctx, file)
	if !changed {
		return nil
	}

	return writeFormattedFile(filename, fset, file)
}

func parseGoFile(filename string, src []byte) (*ast.File, *token.FileSet, error) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("parse file: %w", err)
	}

	return file, fset, nil
}

func processAST(ctx context.Context, file *ast.File) bool {
	changed := false

	tagPositions := make(map[token.Pos]bool)

	ast.Inspect(file, func(n ast.Node) bool {
		if field, ok := n.(*ast.Field); ok && field.Tag != nil {
			tagPositions[field.Tag.Pos()] = true
		}

		return true
	})

	ast.Inspect(file, func(n ast.Node) bool {
		if isCancelled(ctx) {
			return false
		}

		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}

		if tagPositions[lit.Pos()] {
			return true
		}

		if shouldConvertLiteral(lit.Value) {
			content := lit.Value[1 : len(lit.Value)-1]
			lit.Value = strconv.Quote(content)
			changed = true
		}

		return true
	})

	return changed
}

func writeFormattedFile(filename string, fset *token.FileSet, file *ast.File) error {
	var buf strings.Builder
	if err := printer.Fprint(&buf, fset, file); err != nil {
		return fmt.Errorf("print file: %w", err)
	}

	formatted, err := format.Source([]byte(buf.String()))
	if err != nil {
		return fmt.Errorf("format source: %w", err)
	}

	if err := os.WriteFile(filename, formatted, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	log.Printf("Fixed: %s", filename)

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
