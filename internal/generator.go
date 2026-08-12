package internal

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type FileResult struct {
	Path   string
	Status string
}

type Generator struct {
	reader *bufio.Reader
}

func NewGenerator(reader *bufio.Reader) *Generator {
	return &Generator{
		reader: reader,
	}
}

func (g *Generator) ReadProjectName(ctx context.Context) string {
	for {
		if ctx.Err() != nil {
			return ""
		}

		fmt.Printf("%sProject name%s: ", Bold, Reset)

		input, ok := g.readLine(ctx)
		if !ok {
			return ""
		}

		name := strings.TrimSpace(input)

		if err := ValidateProjectName(name); err != nil {
			fmt.Printf("%s✗ %v%s\n", Red, err, Reset)
			continue
		}

		return name
	}
}

func (g *Generator) ReadFileCount(ctx context.Context) int {
	for {
		if ctx.Err() != nil {
			return 0
		}

		fmt.Printf("%sHow many files?%s ", Bold, Reset)

		input, ok := g.readLine(ctx)
		if !ok {
			return 0
		}

		input = strings.TrimSpace(input)

		count := 0
		_, err := fmt.Sscanf(input, "%d", &count)

		if err != nil || count <= 0 {
			fmt.Printf(
				"%s✗ Enter a positive number.%s\n",
				Red,
				Reset,
			)
			continue
		}

		if count > MaxFiles {
			fmt.Printf(
				"%s✗ Too many files. Maximum allowed: %d%s\n",
				Red,
				MaxFiles,
				Reset,
			)
			continue
		}

		return count
	}
}

func (g *Generator) ReadFileNames(
	ctx context.Context,
	count int,
) []string {
	files := make([]string, 0, count)
	seen := make(map[string]struct{}, count)

	for i := 1; i <= count; i++ {
		for {
			if ctx.Err() != nil {
				return nil
			}

			fmt.Printf(
				"%s%d.%s File name: ",
				Bold,
				i,
				Reset,
			)

			input, ok := g.readLine(ctx)
			if !ok {
				return nil
			}

			name := strings.TrimSpace(input)

			normalized, err := ValidateFilePath(name)
			if err != nil {
				fmt.Printf(
					"%s✗ %v%s\n",
					Red,
					err,
					Reset,
				)
				continue
			}

			key := normalized

			if os.PathSeparator == '\\' {
				key = strings.ToLower(key)
			}

			if _, exists := seen[key]; exists {
				fmt.Printf(
					"%s✗ This file was already added.%s\n",
					Red,
					Reset,
				)
				continue
			}

			seen[key] = struct{}{}
			files = append(files, normalized)

			break
		}
	}

	return files
}

func (g *Generator) readLine(
	ctx context.Context,
) (string, bool) {
	if ctx.Err() != nil {
		return "", false
	}

	line, err := g.reader.ReadString('\n')

	if err != nil {
		if len(line) == 0 {
			return "", false
		}
	}

	return strings.TrimRight(
		line,
		"\r\n",
	), true
}

func (g *Generator) Confirm(
	ctx context.Context,
	message string,
) bool {
	for {
		if ctx.Err() != nil {
			return false
		}

		fmt.Print(message)

		input, ok := g.readLine(ctx)
		if !ok {
			return false
		}

		switch strings.ToLower(
			strings.TrimSpace(input),
		) {
		case "", "y", "yes":
			return true

		case "n", "no":
			return false

		default:
			fmt.Printf(
				"%s✗ Enter Y or N.%s\n",
				Red,
				Reset,
			)
		}
	}
}

func (g *Generator) AskExistingFile(
	ctx context.Context,
	path string,
) string {
	for {
		if ctx.Err() != nil {
			return "cancel"
		}

		fmt.Println()

		fmt.Printf(
			"%sFile already exists:%s %s\n",
			Yellow,
			Reset,
			path,
		)

		fmt.Println("  [1] Overwrite")
		fmt.Println("  [2] Skip")
		fmt.Println("  [3] Cancel")
		fmt.Print("  Choice [2]: ")

		input, ok := g.readLine(ctx)
		if !ok {
			return "cancel"
		}

		switch strings.TrimSpace(input) {
		case "", "2":
			return "skip"

		case "1":
			return "overwrite"

		case "3":
			return "cancel"

		default:
			fmt.Printf(
				"%s✗ Choose 1, 2 or 3.%s\n",
				Red,
				Reset,
			)
		}
	}
}

func CreateFiles(
	ctx context.Context,
	projectPath string,
	files []string,
	g *Generator,
) []FileResult {
	results := make([]FileResult, 0, len(files))

	for index, relativePath := range files {
		if ctx.Err() != nil {
			results = append(
				results,
				FileResult{
					Path:   relativePath,
					Status: "skipped",
				},
			)

			for _, remaining := range files[index+1:] {
				results = append(
					results,
					FileResult{
						Path:   remaining,
						Status: "skipped",
					},
				)
			}

			break
		}

		fmt.Printf(
			"%s[%d/%d]%s %s",
			Gray,
			index+1,
			len(files),
			Reset,
			relativePath,
		)

		fullPath, err := SecureJoin(
			projectPath,
			relativePath,
		)

		if err != nil {
			fmt.Printf(
				" %s✗ %v%s\n",
				Red,
				err,
				Reset,
			)

			results = append(
				results,
				FileResult{
					Path:   relativePath,
					Status: "failed",
				},
			)

			continue
		}

		existing, err := PathExists(fullPath)

		if err != nil {
			fmt.Printf(
				" %s✗ %v%s\n",
				Red,
				err,
				Reset,
			)

			results = append(
				results,
				FileResult{
					Path:   relativePath,
					Status: "failed",
				},
			)

			continue
		}

		if existing {
			info, err := os.Lstat(fullPath)

			if err != nil {
				fmt.Printf(
					" %s✗ %v%s\n",
					Red,
					err,
					Reset,
				)

				results = append(
					results,
					FileResult{
						Path:   relativePath,
						Status: "failed",
					},
				)

				continue
			}

			if info.Mode()&os.ModeSymlink != 0 {
				fmt.Printf(
					" %s✗ refusing to modify symbolic link%s\n",
					Red,
					Reset,
				)

				results = append(
					results,
					FileResult{
						Path:   relativePath,
						Status: "failed",
					},
				)

				continue
			}

			if info.IsDir() {
				fmt.Printf(
					" %s✗ path is already a directory%s\n",
					Red,
					Reset,
				)

				results = append(
					results,
					FileResult{
						Path:   relativePath,
						Status: "failed",
					},
				)

				continue
			}

			choice := g.AskExistingFile(
				ctx,
				relativePath,
			)

			switch choice {
			case "overwrite":
				if err := OverwriteFile(fullPath); err != nil {
					fmt.Printf(
						" %s✗ %v%s\n",
						Red,
						err,
						Reset,
					)

					results = append(
						results,
						FileResult{
							Path:   relativePath,
							Status: "failed",
						},
					)

					continue
				}

				fmt.Printf(
					" %s✓ overwritten%s\n",
					Green,
					Reset,
				)

				results = append(
					results,
					FileResult{
						Path:   relativePath,
						Status: "created",
					},
				)

			case "skip":
				fmt.Printf(
					" %s→ skipped%s\n",
					Yellow,
					Reset,
				)

				results = append(
					results,
					FileResult{
						Path:   relativePath,
						Status: "skipped",
					},
				)

			default:
				fmt.Printf(
					" %s→ cancelled%s\n",
					Yellow,
					Reset,
				)

				results = append(
					results,
					FileResult{
						Path:   relativePath,
						Status: "skipped",
					},
				)

				for _, remaining := range files[index+1:] {
					results = append(
						results,
						FileResult{
							Path:   remaining,
							Status: "skipped",
						},
					)
				}

				return results
			}

			continue
		}

		parent := filepath.Dir(fullPath)

		if err := EnsureSafeDirectories(
			projectPath,
			parent,
		); err != nil {
			fmt.Printf(
				" %s✗ %v%s\n",
				Red,
				err,
				Reset,
			)

			results = append(
				results,
				FileResult{
					Path:   relativePath,
					Status: "failed",
				},
			)

			continue
		}

		if err := os.MkdirAll(
			parent,
			0755,
		); err != nil {
			fmt.Printf(
				" %s✗ %v%s\n",
				Red,
				err,
				Reset,
			)

			results = append(
				results,
				FileResult{
					Path:   relativePath,
					Status: "failed",
				},
			)

			continue
		}

		if err := CreateEmptyFile(fullPath); err != nil {
			fmt.Printf(
				" %s✗ %v%s\n",
				Red,
				err,
				Reset,
			)

			results = append(
				results,
				FileResult{
					Path:   relativePath,
					Status: "failed",
				},
			)

			continue
		}

		fmt.Printf(
			" %s✓ created%s\n",
			Green,
			Reset,
		)

		results = append(
			results,
			FileResult{
				Path:   relativePath,
				Status: "created",
			},
		)
	}

	return results
}

func PrintPreview(
	projectName string,
	files []string,
) {
	fmt.Printf(
		"%s%sProject preview%s\n",
		Bold,
		Cyan,
		Reset,
	)

	fmt.Printf(
		"%s%s/%s\n",
		Bold,
		projectName,
		Reset,
	)

	for i, file := range files {
		prefix := "├── "

		if i == len(files)-1 {
			prefix = "└── "
		}

		fmt.Printf(
			"%s%s\n",
			prefix,
			file,
		)
	}

	fmt.Println()
}

func PrintSummary(
	projectName string,
	projectPath string,
	results []FileResult,
) {
	created := 0
	skipped := 0
	failed := 0

	for _, result := range results {
		switch result.Status {
		case "created":
			created++

		case "skipped":
			skipped++

		case "failed":
			failed++
		}
	}

	fmt.Println()

	fmt.Printf(
		"%s%sProject created successfully!%s\n\n",
		Bold,
		Green,
		Reset,
	)

	fmt.Printf(
		"%sProject:%s\n",
		Bold,
		Reset,
	)

	fmt.Printf(
		"%s\n\n",
		projectName,
	)

	fmt.Printf(
		"%sLocation:%s\n",
		Bold,
		Reset,
	)

	fmt.Printf(
		"%s\n\n",
		projectPath,
	)

	fmt.Printf(
		"%sFiles:%s\n",
		Bold,
		Reset,
	)

	for _, result := range results {
		switch result.Status {
		case "created":
			fmt.Printf(
				"  %s✓%s %s\n",
				Green,
				Reset,
				result.Path,
			)

		case "skipped":
			fmt.Printf(
				"  %s→%s %s %s(skipped)%s\n",
				Yellow,
				Reset,
				result.Path,
				Gray,
				Reset,
			)

		case "failed":
			fmt.Printf(
				"  %s✗%s %s %s(failed)%s\n",
				Red,
				Reset,
				result.Path,
				Gray,
				Reset,
			)
		}
	}

	fmt.Println()

	fmt.Printf(
		"%sCreated:%s %d\n",
		Bold,
		Reset,
		created,
	)

	fmt.Printf(
		"%sSkipped:%s %d\n",
		Bold,
		Reset,
		skipped,
	)

	fmt.Printf(
		"%sFailed:%s  %d\n",
		Bold,
		Reset,
		failed,
	)
}

type treeNode struct {
	name     string
	children map[string]*treeNode
	file     bool
}

func PrintTree(
	projectName string,
	results []FileResult,
) {
	root := &treeNode{
		name:     projectName,
		children: make(map[string]*treeNode),
	}

	for _, result := range results {
		if result.Status != "created" {
			continue
		}

		parts := strings.Split(
			filepath.ToSlash(result.Path),
			"/",
		)

		current := root

		for i, part := range parts {
			child, exists := current.children[part]

			if !exists {
				child = &treeNode{
					name:     part,
					children: make(map[string]*treeNode),
				}

				current.children[part] = child
			}

			if i == len(parts)-1 {
				child.file = true
			}

			current = child
		}
	}

	fmt.Printf(
		"%s%sProject tree%s\n",
		Bold,
		Cyan,
		Reset,
	)

	fmt.Printf(
		"%s/%s\n",
		projectName,
		"",
	)

	printTreeChildren(root, "")
}

func printTreeChildren(
	node *treeNode,
	prefix string,
) {
	names := make([]string, 0, len(node.children))

	for name := range node.children {
		names = append(names, name)
	}

	sort.Strings(names)

	for i, name := range names {
		child := node.children[name]
		last := i == len(names)-1

		branch := "├── "
		nextPrefix := prefix + "│   "

		if last {
			branch = "└── "
			nextPrefix = prefix + "    "
		}

		if child.file {
			fmt.Printf(
				"%s%s%s\n",
				prefix,
				branch,
				name,
			)
		} else {
			fmt.Printf(
				"%s%s%s/\n",
				prefix,
				branch,
				name,
			)

			printTreeChildren(
				child,
				nextPrefix,
			)
		}
	}
}
