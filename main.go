package main

import (
	"flag"
	"fmt"
	"github.com/howeyc/fsnotify"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"
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

	fmt.Println("Watching", dirs, "for", cmd)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		subfolders := Subfolders(dir)
		for _, f := range subfolders {
			err = watcher.WatchFlags(f, fsnotify.FSN_ALL)
			if err != nil {
				return err
			}
		}
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
		}
	}

	var fired bool = false
	for {
		select {
		case e := <-watcher.Event:
			included, err := regexp.MatchString(*include, e.Name)
			if err != nil {
				log.Println(err)
			}

			if !included {
				continue
			}
			log.Println(e.Name)

			if !fired {
				fired = true
				go func(dir string) {
					// duration to avoid to run commands frequency at once
					select {
					case <-time.After(200 * time.Millisecond):
						fired = false
						if task != nil && task.ProcessState != nil && !task.ProcessState.Exited() {
							fmt.Println("Stopping Task...")
							err := task.Process.Kill()
							if err != nil {
								log.Println(err)
							}
						}
						fmt.Println("Running Task:", cmd)
						task = exec.Command(cmd[0], cmd[1:]...)
						task.Stdout = os.Stdout
						task.Stderr = os.Stderr
						if options.Bool("chdir") {
							task.Dir = dir
						}
						runCommand(task)
					}
				}(filepath.Dir(e.Name))
			}

		case err := <-watcher.Error:
			log.Println("Error:", err)
		}
	}

	watcher.Close()
	return nil
}

var failed = fmt.Println
var success = fmt.Println
