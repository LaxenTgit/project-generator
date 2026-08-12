package internal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const MaxFiles = 10000

func ValidateProjectName(name string) error {
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
		return errors.New(
			"project name contains characters unsupported by Windows",
		)
	}

	if strings.HasSuffix(name, " ") ||
		strings.HasSuffix(name, ".") {
		return errors.New(
			"project name cannot end with a space or dot",
		)
	}

	if IsWindowsReservedName(name) {
		return errors.New(
			"project name is reserved by Windows",
		)
	}

	for _, r := range name {
		if unicode.IsControl(r) {
			return errors.New(
				"project name contains a control character",
			)
		}
	}

	return nil
}

func ValidateFilePath(input string) (string, error) {
	if input == "" {
		return "", errors.New(
			"file name cannot be empty",
		)
	}

	if strings.ContainsRune(input, '\x00') {
		return "", errors.New(
			"file path contains a null byte",
		)
	}

	if filepath.IsAbs(input) {
		return "", errors.New(
			"absolute paths are not allowed",
		)
	}

	if filepath.VolumeName(input) != "" {
		return "", errors.New(
			"volume-qualified paths are not allowed",
		)
	}

	clean := filepath.Clean(input)

	if clean == "." || clean == ".." {
		return "", errors.New(
			"invalid file path",
		)
	}

	rel, err := filepath.Rel(".", clean)
	if err != nil {
		return "", errors.New(
			"could not normalize file path",
		)
	}

	if rel == ".." ||
		strings.HasPrefix(
			rel,
			".."+string(os.PathSeparator),
		) {
		return "", errors.New(
			"path traversal is not allowed",
		)
	}

	parts := strings.FieldsFunc(
		clean,
		func(r rune) bool {
			return r == '/' || r == '\\'
		},
	)

	for _, part := range parts {
		if part == "." || part == ".." {
			return "", errors.New(
				"path traversal is not allowed",
			)
		}

		if strings.ContainsAny(
			part,
			`<>:"|?*`,
		) {
			return "",
				fmt.Errorf(
					"invalid path component: %q",
					part,
				)
		}

		if strings.HasSuffix(part, " ") ||
			strings.HasSuffix(part, ".") {
			return "",
				fmt.Errorf(
					"path component cannot end with space or dot: %q",
					part,
				)
		}

		if IsWindowsReservedName(part) {
			return "",
				fmt.Errorf(
					"reserved Windows path component: %q",
					part,
				)
		}
	}

	return clean, nil
}

func IsWindowsReservedName(name string) bool {
	base := strings.TrimSpace(name)

	if index := strings.IndexByte(base, '.'); index >= 0 {
		base = base[:index]
	}

	switch strings.ToUpper(base) {
	case "CON", "PRN", "AUX", "NUL":
		return true

	case "COM1", "COM2", "COM3",
		"COM4", "COM5", "COM6",
		"COM7", "COM8", "COM9":
		return true

	case "LPT1", "LPT2", "LPT3",
		"LPT4", "LPT5", "LPT6",
		"LPT7", "LPT8", "LPT9":
		return true

	default:
		return false
	}
}
