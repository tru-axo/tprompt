package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
)

// This is really just a shell agnostic way of implementing my
// favorite prompt config. Once installed, it should be just called
// in whatever config you might want.
func main() {
	flag := flag.NewFlagSet("tprompt", flag.ExitOnError)

	l := flag.Bool("left", true, "prints left prompt")
	r := flag.Bool("right", false, "prints right prompt")

	if err := flag.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(flag.Output(), "Could not parse flags: %v\n", err)
		os.Exit(1)
	}

	if *r {
		out, err := right()
		if err != nil {
			fmt.Fprintf(flag.Output(), "Right prompt err: %v\n", err)
			os.Exit(1)
		}
		os.Stdout.WriteString(out)
		os.Exit(0)
	}

	if *l {
		out, err := left()
		if err != nil {
			fmt.Fprintf(flag.Output(), "Left prompt err: %v\n", err)
			os.Exit(1)
		}
		os.Stdout.WriteString(out)
		os.Exit(0)
	}

	flag.Usage()
	os.Exit(1)
}

// right generates the right prompt.
// If executed over ssh, it prints <username>@<hostname>, otherwise
// it is just an empty string.
func right() (string, error) {
	if !isRemote() {
		return "", nil
	}

	usr, err := user.Current()
	if err != nil {
		return "", err
	}

	host, err := os.Hostname()
	if err != nil {
		return "", err
	}

	return usr.Username + "@" + host, nil
}

func left() (string, error) {
	var prompt strings.Builder

	wd, err := os.Getwd()
	if err != nil {
		return prompt.String(), err
	}

	usr, err := user.Current()
	if err != nil {
		return prompt.String(), err
	}

	if strings.HasPrefix(wd, usr.HomeDir) {
		prompt.WriteRune('~')
		prompt.WriteString(strings.TrimPrefix(wd, usr.HomeDir))
	} else {
		prompt.WriteString(wd)
	}

	if isRepo() {
		var flags string

		status, err := repoStatus()
		if err != nil {
			return prompt.String(), err
		}

		if status.head != "main" && status.head != "master" {
			flags += "c"
		}

		if status.dirty {
			flags += "d"
		}

		if status.behind > 0 {
			flags += "b"
		}

		if status.ahead > 0 {
			flags += "a"
		}

		if status.stash {
			flags += "s"
		}

		if len(flags) > 0 {
			prompt.WriteString("|" + flags)
		}
	}

	prompt.WriteString("> ")

	return fmt.Sprintf("%s", prompt.String()), nil
}

func isRemote() bool {
	return os.Getenv("SSH_CONNECTION") != ""
}

func isRepo() bool {
	cmd := exec.Command("git", "--no-optional-locks", "rev-parse", "--is-inside-work-tree")
	err := cmd.Run()
	return err == nil
}

func repoPath() (string, error) {
	cmd := exec.Command("git", "--no-optional-locks", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	return string(out), err
}

func repoStatus() (status, error) {
	cmd := exec.Command("git", "--no-optional-locks", "status", "--show-stash", "--branch", "--porcelain=v2")
	out, err := cmd.Output()
	if err != nil {
		return status{}, err
	}

	return parseRepoStatus(out)
}

func parseRepoStatus(in []byte) (status, error) {
	var st status
	for _, ln := range strings.Split(string(in), "\n") {
		info := strings.Split(ln, " ")

		if len(info) < 2 {
			continue
		}

		switch info[0] {
		case "#":
			switch info[1] {
			case "branch.head":
				st.head = info[2]

			case "branch.ab":
				st.ahead, _ = strconv.Atoi(info[2][1:])
				st.behind, _ = strconv.Atoi(info[3][1:])

			case "stash":
				n, err := strconv.Atoi(info[2])
				if err != nil {
					continue
				}
				st.stash = n > 0
			}

		case "!", "?", "1", "2", "u":
			st.dirty = true
		}
	}

	return st, nil
}

type status struct {
	head   string
	ahead  int
	behind int
	dirty  bool
	stash  bool
}
