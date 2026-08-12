package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"lat-project-generator/internal"
)

const (
	reset = "\033[0m"
	bold  = "\033[1m"

	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	gray   = "\033[90m"
)

func Run(ctx context.Context) {
	g := internal.NewGenerator(
		bufio.NewReader(os.Stdin),
	)

	printBanner()

	projectName := g.ReadProjectName(ctx)
	if projectName == "" {
		fmt.Println()
		fmt.Printf("%s✗ Operation cancelled.%s\n", red, reset)
		return
	}

	fileCount := g.ReadFileCount(ctx)
	if fileCount == 0 {
		fmt.Println()
		fmt.Printf("%s✗ Operation cancelled.%s\n", red, reset)
		return
	}

	files := g.ReadFileNames(ctx, fileCount)
	if files == nil {
		fmt.Println()
		fmt.Printf("%s✗ Operation cancelled.%s\n", red, reset)
		return
	}

	projectPath, err := filepath.Abs(projectName)
	if err != nil {
		fmt.Printf(
			"%s✗ Could not resolve project path: %v%s\n",
			red,
			err,
			reset,
		)
		return
	}

	fmt.Println()
	internal.PrintPreview(projectName, files)

	if !g.Confirm(
		ctx,
		"Continue and create project? [Y/n]: ",
	) {
		fmt.Printf(
			"%s✗ Operation cancelled.%s\n",
			yellow,
			reset,
		)
		return
	}

	fmt.Println()

	exists, err := internal.PathExists(projectPath)
	if err != nil {
		fmt.Printf(
			"%s✗ Could not inspect project path: %v%s\n",
			red,
			err,
			reset,
		)
		return
	}

	if exists {
		info, err := os.Lstat(projectPath)
		if err != nil {
			fmt.Printf(
				"%s✗ Could not inspect project directory: %v%s\n",
				red,
				err,
				reset,
			)
			return
		}

		if info.Mode()&os.ModeSymlink != 0 {
			fmt.Printf(
				"%s✗ Refusing to use a symbolic link as the project directory.%s\n",
				red,
				reset,
			)
			return
		}

		if !info.IsDir() {
			fmt.Printf(
				"%s✗ Project path exists but is not a directory.%s\n",
				red,
				reset,
			)
			return
		}

		fmt.Printf(
			"%s! Project directory already exists:%s %s\n",
			yellow,
			reset,
			projectPath,
		)
	} else {
		if err := os.MkdirAll(projectPath, 0755); err != nil {
			fmt.Printf(
				"%s✗ Could not create project directory: %v%s\n",
				red,
				err,
				reset,
			)
			return
		}
	}

	results := internal.CreateFiles(
		ctx,
		projectPath,
		files,
		g,
	)

	internal.PrintSummary(
		projectName,
		projectPath,
		results,
	)

	fmt.Println()

	internal.PrintTree(
		projectName,
		results,
	)

	fmt.Println()
	fmt.Printf(
		"%s%sDone.%s\n",
		bold,
		green,
		reset,
	)
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
