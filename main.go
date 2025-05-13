package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
)

func main() {
	out, err := run()
	if err != nil {
		msg := strings.TrimSpace(err.Error())
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, out)
}

func run() (string, error) {
	c, err := parseCLI()
	if err != nil {
		return "", err
	}

	url, err := getRemoteURL(c.path, c.lineNumber, c.blame)
	if err != nil {
		return "", err
	}

	if err = copyToClipboard(url); err != nil {
		return "", err
	}

	if c.open {
		if err = openBrowser(url); err != nil {
			return "", err
		}
	}

	return url, nil
}

type config struct {
	path       string
	lineNumber string
	blame      bool
	open       bool
}

func parseCLI() (config, error) {
	c := config{
		blame: os.Getenv("BLAME") == "true",
		open:  os.Getenv("OPEN") == "true",
	}

	var args []string
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--help", "-h":
			printAndExit(helpString)
		case "--version", "-V":
			printAndExit(getVersion())
		case "--blame":
			c.blame = true
		case "--open":
			c.open = true
		default:
			if strings.HasPrefix(arg, "-") {
				return c, invalidFlagErr(arg)
			}
			args = append(args, arg)
		}
	}

	if len(args) != 2 {
		return c, errors.New("unexpected arguments\n\nTry 'gitlink --help'")
	}
	c.path = args[0]
	c.lineNumber = args[1]

	absPath, err := filepath.Abs(c.path)
	if err != nil {
		return c, err
	}
	c.path = absPath

	return c, nil
}

func getRemoteURL(path, lineNumber string, blame bool) (string, error) {
	repoRoot, err := runGit("-C", filepath.Dir(path), "rev-parse", "--show-toplevel")
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
	relPath, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return "", err
	}

	kind := "blob"
	if blame {
		kind = "blame"
	}

	return fmt.Sprintf("%s/%s/%s/%s#L%s", url, kind, commit, relPath, lineNumber), nil
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

func invalidFlagErr(arg string) error {
	flag, _, _ := strings.Cut(arg, "=")
	return fmt.Errorf("invalid flag provided: '%s'", flag)
}

func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return "dev"
	}
	return info.Main.Version
}

func printAndExit(s string) {
	fmt.Fprintln(os.Stdout, s)
	os.Exit(0)
}

const helpString = `gitlink

Usage: gitlink [OPTIONS] <FILEPATH> <LINE_NUM>

Arguments:
  <FILEPATH>  Path to the file
  <LINE_NUM>  Line number to link to

Options:
  --blame  Link to the git blame view
  --open   Open link in the default browser`
