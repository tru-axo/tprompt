package main

import (
	"flag"
	"fmt"
	"io"
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
	r := flag.Bool("right", false, "prints right prompt")
	flag.Parse()

	if *r {
		if err := right(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Right prompt err: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := left(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Left prompt err: %v\n", err)
			os.Exit(1)
		}
	}
}

func right(out io.Writer) error {
	if !isRemote() {
		return nil
	}

	usr, err := user.Current()
	if err != nil {
		return err
	}
	out.Write([]byte(usr.Username))

	host, err := os.Hostname()
	if err != nil {
		return err
	}
	out.Write([]byte("@" + host))

	return nil
}

func left(out io.Writer) error {
	// wd, err := os.Getwd()
	// if err != nil {
	// 	return err
	// }

	// home, err := os.UserHomeDir()
	// if err != nil {
	// 	return err
	// }

	// dir := strings.ReplaceAll(wd, home, "~")
	// out.Write([]byte(dir))

	if isRepo() {
		out.Write([]byte("@"))

		status, err := repoStatus()
		if err != nil {
			return err
		}

		if status.head != "main" && status.head != "master" {
			out.Write([]byte("*"))
		}

		if status.dirty {
			out.Write([]byte("!"))
		}

		if status.behind > 0 {
			out.Write([]byte("<"))
		}

		if status.ahead > 0 {
			out.Write([]byte(">"))
		}

		if status.stash {
			out.Write([]byte("$"))
		}
	}

	out.Write([]byte("> "))

	return nil
}

func isRemote() bool {
	return os.Getenv("SSH_CONNECTION") != ""
}

func isRepo() bool {
	cmd := exec.Command("git", "--no-optional-locks", "rev-parse", "--is-inside-work-tree")
	err := cmd.Run()
	return err == nil
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
