package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
)

var versionStr = "akavel/0.1.1"

func main() {
	err := run()
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
	os.Exit(1)
}

func run() error {
	// parse arguments

	opt := flag.NewFlagSet("", flag.ExitOnError)
	opt.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] [DIR] -- COMMAND [WITH ARGS...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Where OPTIONS:\n")
		opt.PrintDefaults()
	}
	include := opt.String("include", "\\.(go|c|h)$", "regular expressions specifying file patterns to watch")

	// after double-dash, we expect a command to call on change, with arguments
	cmd := []string{}
	args := []string{}
	for i := 1; i < len(os.Args)-1; i++ {
		if os.Args[i] != "--" {
			continue
		}
		cmd = os.Args[i+1:]
		args = os.Args[1:i]
	}
	if len(cmd) == 0 {
		fmt.Fprintf(os.Stderr, "no command specified\n")
		opt.Usage()
		os.Exit(1)
	}

	opt.Parse(args)

	dirs := opt.Args()
	if len(dirs) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		dirs = []string{cwd}
	}
	if len(dirs) > 1 {
		return fmt.Errorf("FIXME: can handle max 1 dir for now")
	}

	fmt.Println("Watching", dirs, "for", cmd)

	watcher := Watcher{
		Excluder: func(path string) bool {
			included, err := regexp.MatchString(*include, path)
			if err != nil {
				log.Println(err)
				return false
			}
			return !included
		},
	}

	err := watcher.Start(dirs[0])
	if err != nil {
		return err
	}

	var wasFailed bool = false
	var task *exec.Cmd
	runCommand := func(task *exec.Cmd) {
		err := task.Start()
		if err != nil {
			log.Println(err)
			failed("Failed!")
			wasFailed = true
			return
		}
		err = task.Wait()
		if err != nil {
			log.Println(err)
			failed("Failed!")
			wasFailed = true
			return
		}

		// fixed
		if wasFailed {
			wasFailed = false
			success("Congratulations! It's fixed!")
		} else {
			success("Done.")
		}
	}

	for {
		watcher.Latch.Wait()
		log.Println("Running Task:", cmd)
		task = exec.Command(cmd[0], cmd[1:]...)
		task.Stdout = os.Stdout
		task.Stderr = os.Stderr
		runCommand(task)
	}

	<-watcher.Quit
	return nil
}

var failed = log.Println
var success = log.Println
