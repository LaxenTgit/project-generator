package main

import (
        "bufio"
        "context"
        "errors"
        "fmt"
        "os"
        "os/signal"
        "path/filepath"
        "sort"
        "strconv"
        "strings"
        "unicode"
)

const (
        maxFiles = 10000

        reset = "\033[0m"
        bold  = "\033[1m"

        red    = "\033[31m"
        green  = "\033[32m"
        yellow = "\033[33m"
        cyan   = "\033[36m"
        gray   = "\033[90m"
)

type FileResult struct {
        Path   string
        Status string
}

type Generator struct {
        reader *bufio.Reader
}

func main() {
        ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
        defer stop()

        g := &Generator{
                reader: bufio.NewReader(os.Stdin),
        }

        printBanner()

        projectName := g.readProjectName(ctx)
        if projectName == "" {
                fmt.Println()
                fmt.Printf("%s✗ Operation cancelled.%s\n", red, reset)
                return
        }

        fileCount := g.readFileCount(ctx)
        if fileCount == 0 {
                fmt.Println()
                fmt.Printf("%s✗ Operation cancelled.%s\n", red, reset)
                return
        }

        files := g.readFileNames(ctx, fileCount)
        if files == nil {
                fmt.Println()
                fmt.Printf("%s✗ Operation cancelled.%s\n", red, reset)
                return
        }

        projectPath, err := filepath.Abs(projectName)
        if err != nil {
                fmt.Printf("%s✗ Could not resolve project path: %v%s\n", red, err, reset)
                return
        }

        fmt.Println()
        printPreview(projectName, files)

        if !g.confirm(ctx, "Continue and create project? [Y/n]: ") {
                fmt.Printf("%s✗ Operation cancelled.%s\n", yellow, reset)
                return
        }

        fmt.Println()

        exists, err := pathExists(projectPath)
        if err != nil {
                fmt.Printf("%s✗ Could not inspect project path: %v%s\n", red, err, reset)
                return
        }

        if exists {
                info, err := os.Lstat(projectPath)
                if err != nil {
                        fmt.Printf("%s✗ Could not inspect project directory: %v%s\n", red, err, reset)
                        return
                }

                if info.Mode()&os.ModeSymlink != 0 {
                        fmt.Printf("%s✗ Refusing to use a symbolic link as the project directory.%s\n", red, reset)
                        return
                }

                if !info.IsDir() {
                        fmt.Printf("%s✗ Project path exists but is not a directory.%s\n", red, reset)
                        return
                }

                fmt.Printf("%s! Project directory already exists:%s %s\n", yellow, reset, projectPath)
        } else {
                if err := os.MkdirAll(projectPath, 0755); err != nil {
                        fmt.Printf("%s✗ Could not create project directory: %v%s\n", red, err, reset)
                        return
                }
        }

        results := createFiles(ctx, projectPath, files, g)

        printSummary(projectName, projectPath, results)

        fmt.Println()
        printTree(projectName, results)

        fmt.Println()
        fmt.Printf("%s%sDone.%s\n", bold, green, reset)
}

func printBanner() {
        fmt.Println()
        fmt.Printf("%s%s", bold, cyan)
        fmt.Println("╔══════════════════════════════════════╗")
        fmt.Println("║                 LAT                  ║")
        fmt.Println("║                 :3                   ║")
        fmt.Println("╚══════════════════════════════════════╝")
        fmt.Printf("%s", reset)
        fmt.Println()
}

func (g *Generator) readProjectName(ctx context.Context) string {
        for {
                if ctx.Err() != nil {
                        return ""
                }

                fmt.Printf("%sProject name%s: ", bold, reset)

                input, ok := g.readLine(ctx)
                if !ok {
                        return ""
                }

                name := strings.TrimSpace(input)

                if err := validateProjectName(name); err != nil {
                        fmt.Printf("%s✗ %v%s\n", red, err, reset)
                        continue
                }

                return name
        }
}

func (g *Generator) readFileCount(ctx context.Context) int {
        for {
                if ctx.Err() != nil {
                        return 0
                }

                fmt.Printf("%sHow many files?%s ", bold, reset)

                input, ok := g.readLine(ctx)
                if !ok {
                        return 0
                }

                input = strings.TrimSpace(input)

                count, err := strconv.Atoi(input)
                if err != nil || count <= 0 {
                        fmt.Printf("%s✗ Enter a positive number.%s\n", red, reset)
                        continue
                }

                if count > maxFiles {
                        fmt.Printf(
                                "%s✗ Too many files. Maximum allowed: %d%s\n",
                                red,
                                maxFiles,
                                reset,
                        )
                        continue
                }

                return count
        }
}

func (g *Generator) readFileNames(ctx context.Context, count int) []string {
        files := make([]string, 0, count)
        seen := make(map[string]struct{}, count)

        for i := 1; i <= count; i++ {
                for {
                        if ctx.Err() != nil {
                                return nil
                        }

                        fmt.Printf("%s%d.%s File name: ", bold, i, reset)

                        input, ok := g.readLine(ctx)
                        if !ok {
                                return nil
                        }

                        name := strings.TrimSpace(input)

                        normalized, err := validateFilePath(name)
                        if err != nil {
                                fmt.Printf("%s✗ %v%s\n", red, err, reset)
                                continue
                        }

                        key := normalized
                        if os.PathSeparator == '\\' {
                                key = strings.ToLower(key)
                        }

                        if _, exists := seen[key]; exists {
                                fmt.Printf("%s✗ This file was already added.%s\n", red, reset)
                                continue
                        }

                        seen[key] = struct{}{}
                        files = append(files, normalized)
                        break
                }
        }

        return files
}

func (g *Generator) readLine(ctx context.Context) (string, bool) {
        if ctx.Err() != nil {
                return "", false
        }

        line, err := g.reader.ReadString('\n')
        if err != nil {
                if errors.Is(err, context.Canceled) {
                        return "", false
                }

                if len(line) == 0 {
                        return "", false
                }
        }

        return strings.TrimRight(line, "\r\n"), true
}

func validateProjectName(name string) error {
        if name == "" {
                return errors.New("project name cannot be empty")
        }

        if name == "." || name == ".." {
                return errors.New("invalid project name")
        }

        if filepath.IsAbs(name) {
                return errors.New("project name must be relative")
        }

        if filepath.VolumeName(name) != "" {
                return errors.New("volume-qualified paths are not allowed")
        }

        if strings.ContainsAny(name, `/\`) {
                return errors.New("project name must be a single directory name")
        }

        if strings.ContainsRune(name, '\x00') {
                return errors.New("project name contains an invalid character")
        }

        if strings.ContainsAny(name, `<>:"|?*`) {
                return errors.New("project name contains characters unsupported by Windows")
        }

        if strings.HasSuffix(name, " ") || strings.HasSuffix(name, ".") {
                return errors.New("project name cannot end with a space or dot")
        }

        if isWindowsReservedName(name) {
                return errors.New("project name is reserved by Windows")
        }

        for _, r := range name {
                if unicode.IsControl(r) {
                        return errors.New("project name contains a control character")
                }
        }

        return nil
}

func validateFilePath(input string) (string, error) {
        if input == "" {
                return "", errors.New("file name cannot be empty")
        }

        if strings.ContainsRune(input, '\x00') {
                return "", errors.New("file path contains a null byte")
        }

        if filepath.IsAbs(input) {
                return "", errors.New("absolute paths are not allowed")
        }

        if filepath.VolumeName(input) != "" {
                return "", errors.New("volume-qualified paths are not allowed")
        }

        clean := filepath.Clean(input)

        if clean == "." || clean == ".." {
                return "", errors.New("invalid file path")
        }

        rel, err := filepath.Rel(".", clean)
        if err != nil {
                return "", errors.New("could not normalize file path")
        }

        if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
                return "", errors.New("path traversal is not allowed")
        }

        parts := strings.FieldsFunc(clean, func(r rune) bool {
                return r == '/' || r == '\\'
        })

        for _, part := range parts {
                if part == "." || part == ".." {
                        return "", errors.New("path traversal is not allowed")
                }

                if strings.ContainsAny(part, `<>:"|?*`) {
                        return "", fmt.Errorf("invalid path component: %q", part)
                }

                if strings.HasSuffix(part, " ") || strings.HasSuffix(part, ".") {
                        return "", fmt.Errorf("path component cannot end with space or dot: %q", part)
                }

                if isWindowsReservedName(part) {
                        return "", fmt.Errorf("reserved Windows path component: %q", part)
                }
        }

        return clean, nil
}

func isWindowsReservedName(name string) bool {
        base := strings.TrimSpace(name)

        if index := strings.IndexByte(base, '.'); index >= 0 {
                base = base[:index]
        }

        switch strings.ToUpper(base) {
        case "CON", "PRN", "AUX", "NUL":
                return true
        case "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9":
                return true
        case "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
                return true
        default:
                return false
        }
}

func pathExists(path string) (bool, error) {
        _, err := os.Lstat(path)

        if err == nil {
                return true, nil
        }

        if errors.Is(err, os.ErrNotExist) {
                return false, nil
        }

        return false, err
}

func createFiles(
        ctx context.Context,
        projectPath string,
        files []string,
        g *Generator,
) []FileResult {
        results := make([]FileResult, 0, len(files))

        for index, relativePath := range files {
                if ctx.Err() != nil {
                        results = append(results, FileResult{
                                Path:   relativePath,
                                Status: "skipped",
                        })

                        for _, remaining := range files[index+1:] {
                                results = append(results, FileResult{
                                        Path:   remaining,
                                        Status: "skipped",
                                })
                        }

                        break
                }

                fmt.Printf(
                        "%s[%d/%d]%s %s",
                        gray,
                        index+1,
                        len(files),
                        reset,
                        relativePath,
                )

                fullPath, err := secureJoin(projectPath, relativePath)
                if err != nil {
                        fmt.Printf(" %s✗ %v%s\n", red, err, reset)

                        results = append(results, FileResult{
                                Path:   relativePath,
                                Status: "failed",
                        })
                        continue
                }

                existing, err := pathExists(fullPath)
                if err != nil {
                        fmt.Printf(" %s✗ %v%s\n", red, err, reset)

                        results = append(results, FileResult{
                                Path:   relativePath,
                                Status: "failed",
                        })
                        continue
                }

                if existing {
                        info, err := os.Lstat(fullPath)
                        if err != nil {
                                fmt.Printf(" %s✗ %v%s\n", red, err, reset)

                                results = append(results, FileResult{
                                        Path:   relativePath,
                                        Status: "failed",
                                })
                                continue
                        }

                        if info.Mode()&os.ModeSymlink != 0 {
                                fmt.Printf(" %s✗ refusing to modify symbolic link%s\n", red, reset)

                                results = append(results, FileResult{
                                        Path:   relativePath,
                                        Status: "failed",
                                })
                                continue
                        }

                        if info.IsDir() {
                                fmt.Printf(" %s✗ path is already a directory%s\n", red, reset)

                                results = append(results, FileResult{
                                        Path:   relativePath,
                                        Status: "failed",
                                })
                                continue
                        }

                        choice := g.askExistingFile(ctx, relativePath)

                        switch choice {
                        case "overwrite":
                                if err := overwriteFile(fullPath); err != nil {
                                        fmt.Printf(" %s✗ %v%s\n", red, err, reset)

                                        results = append(results, FileResult{
                                                Path:   relativePath,
                                                Status: "failed",
                                        })
                                        continue
                                }

                                fmt.Printf(" %s✓ overwritten%s\n", green, reset)

                                results = append(results, FileResult{
                                        Path:   relativePath,
                                        Status: "created",
                                })

                        case "skip":
                                fmt.Printf(" %s→ skipped%s\n", yellow, reset)

                                results = append(results, FileResult{
                                        Path:   relativePath,
                                        Status: "skipped",
                                })

                        default:
                                fmt.Printf(" %s→ cancelled%s\n", yellow, reset)

                                results = append(results, FileResult{
                                        Path:   relativePath,
                                        Status: "skipped",
                                })

                                for _, remaining := range files[index+1:] {
                                        results = append(results, FileResult{
                                                Path:   remaining,
                                                Status: "skipped",
                                        })
                                }

                                return results
                        }

                        continue
                }

                parent := filepath.Dir(fullPath)

                if err := ensureSafeDirectories(projectPath, parent); err != nil {
                        fmt.Printf(" %s✗ %v%s\n", red, err, reset)

                        results = append(results, FileResult{
                                Path:   relativePath,
                                Status: "failed",
                        })
                        continue
                }

                if err := os.MkdirAll(parent, 0755); err != nil {
                        fmt.Printf(" %s✗ %v%s\n", red, err, reset)

                        results = append(results, FileResult{
                                Path:   relativePath,
                                Status: "failed",
                        })
                        continue
                }

                file, err := os.OpenFile(
                        fullPath,
                        os.O_WRONLY|os.O_CREATE|os.O_EXCL,
                        0644,
                )

                if err != nil {
                        fmt.Printf(" %s✗ %v%s\n", red, err, reset)

                        results = append(results, FileResult{
                                Path:   relativePath,
                                Status: "failed",
                        })
                        continue
                }

                closeErr := file.Close()
                if closeErr != nil {
                        fmt.Printf(" %s✗ %v%s\n", red, closeErr, reset)

                        results = append(results, FileResult{
                                Path:   relativePath,
                                Status: "failed",
                        })
                        continue
                }

                fmt.Printf(" %s✓ created%s\n", green, reset)

                results = append(results, FileResult{
                        Path:   relativePath,
                        Status: "created",
                })
        }

        return results
}

func secureJoin(projectPath, relativePath string) (string, error) {
        projectAbs, err := filepath.Abs(projectPath)
        if err != nil {
                return "", err
        }

        target := filepath.Join(projectAbs, relativePath)

        targetAbs, err := filepath.Abs(target)
        if err != nil {
                return "", err
        }

        rel, err := filepath.Rel(projectAbs, targetAbs)
        if err != nil {
                return "", err
        }

        if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
                return "", errors.New("path escapes project directory")
        }

        return targetAbs, nil
}

func ensureSafeDirectories(projectPath, targetDir string) error {
        projectAbs, err := filepath.Abs(projectPath)
        if err != nil {
                return err
        }

        targetAbs, err := filepath.Abs(targetDir)
        if err != nil {
                return err
        }

        rel, err := filepath.Rel(projectAbs, targetAbs)
        if err != nil {
                return err
        }

        if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
                return errors.New("directory escapes project directory")
        }

        current := projectAbs

        if rel == "." {
                return nil
        }

        parts := strings.Split(rel, string(os.PathSeparator))

        for _, part := range parts {
                current = filepath.Join(current, part)

                info, err := os.Lstat(current)
                if err != nil {
                        if errors.Is(err, os.ErrNotExist) {
                                continue
                        }

                        return err
                }

                if info.Mode()&os.ModeSymlink != 0 {
                        return fmt.Errorf("symbolic link in directory path: %s", part)
                }

                if !info.IsDir() {
                        return fmt.Errorf("path component is not a directory: %s", part)
                }
        }

        return nil
}

func overwriteFile(path string) error {
        file, err := os.OpenFile(
                path,
                os.O_WRONLY|os.O_TRUNC,
                0644,
        )
        if err != nil {
                return err
        }

        return file.Close()
}

func (g *Generator) askExistingFile(
        ctx context.Context,
        path string,
) string {
        for {
                if ctx.Err() != nil {
                        return "cancel"
                }

                fmt.Println()
                fmt.Printf("%sFile already exists:%s %s\n", yellow, reset, path)
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
                        fmt.Printf("%s✗ Choose 1, 2 or 3.%s\n", red, reset)
                }
        }
}

func (g *Generator) confirm(
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

                switch strings.ToLower(strings.TrimSpace(input)) {
                case "", "y", "yes":
                        return true
                case "n", "no":
                        return false
                default:
                        fmt.Printf("%s✗ Enter Y or N.%s\n", red, reset)
                }
        }
}

func printPreview(projectName string, files []string) {
        fmt.Printf("%s%sProject preview%s\n", bold, cyan, reset)
        fmt.Printf("%s%s/%s\n", bold, projectName, reset)

        for i, file := range files {
                last := i == len(files)-1

                prefix := "├── "
                if last {
                        prefix = "└── "
                }

                fmt.Printf("%s%s\n", prefix, file)
        }

        fmt.Println()
}

func printSummary(
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
        fmt.Printf("%s%sProject created successfully!%s\n\n", bold, green, reset)

        fmt.Printf("%sProject:%s\n", bold, reset)
        fmt.Printf("%s\n\n", projectName)

        fmt.Printf("%sLocation:%s\n", bold, reset)
        fmt.Printf("%s\n\n", projectPath)

        fmt.Printf("%sFiles:%s\n", bold, reset)

        for _, result := range results {
                switch result.Status {
                case "created":
                        fmt.Printf("  %s✓%s %s\n", green, reset, result.Path)
                case "skipped":
                        fmt.Printf(
                                "  %s→%s %s %s(skipped)%s\n",
                                yellow,
                                reset,
                                result.Path,
                                gray,
                                reset,
                        )
                case "failed":
                        fmt.Printf(
                                "  %s✗%s %s %s(failed)%s\n",
                                red,
                                reset,
                                result.Path,
                                gray,
                                reset,
                        )
                }
        }

        fmt.Println()
        fmt.Printf("%sCreated:%s %d\n", bold, reset, created)
        fmt.Printf("%sSkipped:%s %d\n", bold, reset, skipped)
        fmt.Printf("%sFailed:%s  %d\n", bold, reset, failed)
}

type treeNode struct {
        name     string
        children map[string]*treeNode
        file     bool
}

func printTree(projectName string, results []FileResult) {
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

        fmt.Println()
        fmt.Printf("%s%sProject tree%s\n", bold, cyan, reset)
        fmt.Printf("%s/%s\n", projectName, "")

        printTreeChildren(root, "")
}

func printTreeChildren(node *treeNode, prefix string) {
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
                        fmt.Printf("%s%s%s\n", prefix, branch, name)
                } else {
                        fmt.Printf(
                                "%s%s%s/\n",
                                prefix,
                                branch,
                                name,
                        )

                        printTreeChildren(child, nextPrefix)
                }
        }
}
