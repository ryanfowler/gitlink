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
	url, err := run()
	if err != nil {
		fmt.Printf("Error: %s", err)
		os.Exit(1)
	}
	io.WriteString(os.Stdout, url)
}

func run() (string, error) {
	if len(os.Args) != 3 {
		return "", errors.New("usage: gitlink <filepath> <linenumber>")
	}
	path := os.Args[1]
	lineNumber := os.Args[2]

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
	if os.Getenv("BLAME") == "true" {
		kind = "blame"
	}
	url = fmt.Sprintf("%s/%s/%s/%s#L%s", url, kind, commit, relPath, lineNumber)

	if err = copyToClipboard(url); err != nil {
		return "", err
	}

	if os.Getenv("OPEN") == "true" {
		if err = openBrowser(url); err != nil {
			return "", err
		}
	}
	return url, nil
}

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
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
