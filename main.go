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
	fmt.Fprintf(os.Stderr, "error: %s", err.Error())
	os.Exit(1)
}

func run() error {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [DIR] -- COMMAND [WITH ARGS...]\n", os.Args[0])
		flag.PrintDefaults()
	}

	// parse arguments
	// after double-dash, we expect a command to call on change, with arguments
	cmd := []string{}
	dirs := []string{}
	for i := 1; i < len(os.Args)-1; i++ {
		if os.Args[i] != "--" {
			continue
		}
		cmd = os.Args[i+1:]
		dirs = os.Args[1:i]
	}

	if len(cmd) == 0 {
		flag.Usage()
		return fmt.Errorf("no command to call found")
	}

	/*
		var dirArgs = []string{}
		var cmdArgs = []string{}

		var hasDash bool = false
		var nArgs = len(os.Args)

		for n := 1; n < nArgs; n++ {
			var arg = os.Args[n]
			if arg == "--" {
				hasDash = true
				continue
			}

			var flag, value = "", ""
			if strings.HasPrefix(arg, "-") {
				tokens := strings.SplitN(arg, "=", 2)
				switch len(tokens) {
				case 1:
					flag = tokens[0]
					if n < nArgs-1 && options.Has(flag) && !options.IsBool(flag) {
						value = os.Args[n+1]
						n++
					}
				case 2:
					flag = tokens[0]
					value = tokens[1]
				default:
					continue
				}
			}

			// everything after the dash, should be the command arguments

			// everything before the dash "--"
			if !hasDash {
				if len(flag) > 0 && flag[0] == '-' {
					option := options.Get(flag[1:])
					if option == nil {
						log.Fatalf("Invalid option: '%v'\n", flag)
					} else {
						if _, ok := option.value.(string); ok {
							option.value = value
						} else {
							option.value = true
						}
					}
				} else {
					if exists, _ := FileExists(arg); exists {
						dirArgs = append(dirArgs, arg)
					} else {
						log.Printf("Invalid path: '%v'", arg)
					}
				}
			} else {
				cmdArgs = append(cmdArgs, arg)
			}
		}

		var cmds = CommandSet{}
		var cmd = Command(cmdArgs)

		_ = cmds

		if len(cmd) == 0 {
			if options.Bool("t") {
				cmd = goCommands["test"]
			} else if options.Bool("i") {
				cmd = goCommands["install"]
			} else if options.Bool("f") {
				cmd = goCommands["fmt"]
			} else if options.Bool("r") {
				cmd = goCommands["run"]
			} else if options.Bool("b") {
				cmd = goCommands["build"]
			} else {
				// default behavior
				cmd = goCommands["build"]
			}
			if options.Bool("x") && len(cmd) > 0 {
				cmd = append(cmd, "-x")
			}
		}

		if len(cmd) == 0 {
			fmt.Println("No command specified")
			os.Exit(2)
		}
	*/
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
			matched, err := regexp.MatchString("\\.(go|c|h)$", e.Name)
			if err != nil {
				log.Println(err)
			}

			if !matched {
				if options.Bool("d") {
					log.Println("Ignore:", e)
				}
				continue
			}

			if options.Bool("d") {
				log.Println("Event:", e)
			} else {
				log.Println(e.Name)
			}

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
