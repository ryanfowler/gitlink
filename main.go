package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	out, err := run()
	if err != nil {
		io.WriteString(os.Stderr, fmt.Sprintf("Error: %s\n", err))
		os.Exit(1)
	}
	io.WriteString(os.Stdout, out)
}

func run() (string, error) {
	blame := os.Getenv("BLAME") == "true"
	open := os.Getenv("OPEN") == "true"

	var args []string
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--help", "-h":
			return helpString, nil
		case "--blame":
			blame = true
		case "--open":
			open = true
		default:
			args = append(args, arg)
		}
	}
	if len(args) != 2 {
		return "", errors.New("unexpected arguments\n\nTry 'gitlink --help'")
	}
	path := args[0]
	lineNumber := args[1]

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	repoRoot, err := runGit("-C", filepath.Dir(absPath), "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	relPath, err := filepath.Rel(repoRoot, absPath)
	if err != nil {
		return "", err
	}

	commit, err := runGit("-C", repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	remote, err := runGit("-C", repoRoot, "remote", "get-url", "origin")
	if err != nil {
		return "", err
	}

	url := remoteToURL(remote)

	kind := "blob"
	if blame {
		kind = "blame"
	}
	url = fmt.Sprintf("%s/%s/%s/%s#L%s", url, kind, commit, relPath, lineNumber)

	if err = copyToClipboard(url); err != nil {
		return "", err
	}

	if open {
		if err = openBrowser(url); err != nil {
			return "", err
		}
	}
	return url, nil
}

func runGit(args ...string) (string, error) {
	var stderr, stdout strings.Builder
	cmd := exec.Command("git", args...)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", errors.New(stderr.String())
		}
		if stdout.Len() > 0 {
			return "", errors.New(stdout.String())
		}
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}

func remoteToURL(remote string) string {
	var ok bool
	if remote, ok = strings.CutPrefix(remote, "git@"); ok {
		if host, path, ok := strings.Cut(remote, ":"); ok {
			remote = host + "/" + path
		}
		remote = "https://" + remote
	}
	return strings.TrimSuffix(remote, ".git")
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Run()
}

func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "windows":
		cmd = exec.Command("clip")
	default:
		cmd = exec.Command("xclip", "-selection", "clipboard")
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

const helpString = `gitlink

Usage: gitlink [OPTIONS] <FILEPATH> <LINE_NUM>

Arguments:
  <FILEPATH>  Path to the file
  <LINE_NUM>  Line number to link to

Options:
  --blame  Link to the git blame view
  --open   Open link in the default browser
`
