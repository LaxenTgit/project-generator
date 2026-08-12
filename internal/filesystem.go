package internal

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func PathExists(path string) (bool, error) {
	_, err := os.Lstat(path)

	if err == nil {
		return true, nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, err
}

func SecureJoin(
	projectPath string,
	relativePath string,
) (string, error) {
	projectAbs, err := filepath.Abs(projectPath)
	if err != nil {
		return "", err
	}

	target := filepath.Join(
		projectAbs,
		relativePath,
	)

	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(
		projectAbs,
		targetAbs,
	)
	if err != nil {
		return "", err
	}

	if rel == ".." ||
		strings.HasPrefix(
			rel,
			".."+string(os.PathSeparator),
		) {
		return "",
			errors.New(
				"path escapes project directory",
			)
	}

	return targetAbs, nil
}

func EnsureSafeDirectories(
	projectPath string,
	targetDir string,
) error {
	projectAbs, err := filepath.Abs(projectPath)
	if err != nil {
		return err
	}

	targetAbs, err := filepath.Abs(targetDir)
	if err != nil {
		return err
	}

	rel, err := filepath.Rel(
		projectAbs,
		targetAbs,
	)
	if err != nil {
		return err
	}

	if rel == ".." ||
		strings.HasPrefix(
			rel,
			".."+string(os.PathSeparator),
		) {
		return errors.New(
			"directory escapes project directory",
		)
	}

	current := projectAbs

	if rel == "." {
		return nil
	}

	parts := strings.Split(
		rel,
		string(os.PathSeparator),
	)

	for _, part := range parts {
		current = filepath.Join(
			current,
			part,
		)

		info, err := os.Lstat(current)

		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New(
				"symbolic link in directory path: " +
					part,
			)
		}

		if !info.IsDir() {
			return errors.New(
				"path component is not a directory: " +
					part,
			)
		}
	}

	return nil
}

func OverwriteFile(path string) error {
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

func CreateEmptyFile(path string) error {
	file, err := os.OpenFile(
		path,
		os.O_WRONLY|
			os.O_CREATE|
			os.O_EXCL,
		0644,
	)

	if err != nil {
		return err
	}

	return file.Close()
}
