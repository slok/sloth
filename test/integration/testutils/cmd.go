package testutils

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var multiSpaceRegex = regexp.MustCompile(" +")

// RunSloth executes sloth command.
func RunSloth(ctx context.Context, env []string, cmdApp, cmdArgs string, nolog bool) (stdout, stderr []byte, err error) {
	// Sanitize command.
	cmdArgs = strings.TrimSpace(cmdArgs)
	cmdArgs = multiSpaceRegex.ReplaceAllString(cmdArgs, " ")

	// Split into args.
	args := strings.Split(cmdArgs, " ")

	// Create command.
	var outData, errData bytes.Buffer
	cmd := exec.CommandContext(ctx, cmdApp, args...)
	cmd.Stdout = &outData
	cmd.Stderr = &errData

	// Set env.
	newEnv := append([]string{}, env...)
	newEnv = append(newEnv, os.Environ()...)
	if nolog {
		newEnv = append(newEnv,
			"SLOTH_NO_LOG=true",
			"SLOTH_NO_COLOR=true",
		)
	}
	cmd.Env = newEnv

	// Run.
	err = cmd.Run()

	return outData.Bytes(), errData.Bytes(), err
}

func SlothVersion(ctx context.Context, slothBinary string) (string, error) {
	stdout, stderr, err := RunSloth(ctx, []string{}, slothBinary, "version", false)
	if err != nil {
		return "", fmt.Errorf("could not obtain versions: %s: %w", stderr, err)
	}

	version := string(stdout)
	version = strings.TrimSpace(version)

	return version, nil
}
