package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tm "github.com/buger/goterm"
	"github.com/fsnotify/fsnotify"
)

var (
	verbose     = flag.Bool("v", false, "verbose logging")
	excludeDir  = flag.String("ed", `^\.`, "regular expression of directory basenames to exclude from watching")
	excludeFile = flag.String("e", `(^.*\.sw[px]$)|(/4913$)`, "regular expression of files to exclude when watching")
	includeFile = flag.String("i", `.*`, "regular expression of files to include when watching")
	throttle    = flag.Duration("t", time.Millisecond*100, "a duration of time to wait between a filesystem event and triggering the action")
	noWait      = flag.Bool("n", false, "do not wait for the action to run to completion, use sigkill on the next filesystem change")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: inl SHELL-COMMAND\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
	flag.Parse()
	if len(flag.Args()) < 1 {
		flag.Usage()
	}
	log.Clear()
	cmd := invoke()
	path, err := filepath.Abs("./")
	check(err)
	for {
		time.Sleep(*throttle)
		watchLoop(path)
		if cmd != nil {
			log.Warningf("killing process %d", cmd.Process.Pid)
			cmd.Process.Signal(os.Kill)
			cmd.Wait()
		}
		cmd = invoke()
	}
}

// Recursively watch a filesystem path.
func watchLoop(path string) {
	log.Debugf("establishing watches...")
	watcher, err := fsnotify.NewWatcher()
	check(err)
	defer watcher.Close()
	check(watcher.Add(path))
	dircount := 1
	excludeDirRegexp := regexp.MustCompile(*excludeDir)
	excludeFileRegexp := regexp.MustCompile(*excludeFile)
	includeFileRegexp := regexp.MustCompile(*includeFile)
	err = filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			return nil
		}
		log.Debugf("watch directory candidate '%s'", path)
		if excludeDirRegexp.MatchString(f.Name()) {
			return filepath.SkipDir
		}
		log.Debugf("watching '%s'", path)
		check(watcher.Add(path))
		dircount++
		return nil
	})
	check(err)
	log.Infof("watching '%s' +%d", path, dircount)
	for {
		select {
		case event := <-watcher.Events:
			log.Debugf("event: %+v", event)
			if excludeFileRegexp.MatchString(event.Name) || !includeFileRegexp.MatchString(event.Name) {
				log.Debugf("ignoring event: %+v", event)
				break
			}
			// Ticker fired. There was a lull in events and we can now process.

			log.Ln()
			log.Ln()
			log.Ln()
			log.Infof(strings.Repeat("=", 80))
			log.Clear()
			log.Infof("change detected: %+v", event)
			return
		case err := <-watcher.Errors:
			check(err)
		}
	}
}

func invoke() *exec.Cmd {
	args := []string{"/bin/bash", "-c", "--", strings.Join(flag.Args(), " ")}
	cmd := exec.Command(args[0], args[1:]...)
	log.Infof("executing command: %+q", args)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Ln()
	check(cmd.Start())

	if !*noWait {
		cmd.Wait()
		log.Ln()
		if cmd.ProcessState.Success() {
			log.Infof(cmd.ProcessState.String())
		} else {
			log.Errorf(cmd.ProcessState.String())
		}
		cmd = nil
	}

	return cmd
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

type Log struct{}

func (l Log) Infof(f string, i ...interface{}) {
	l.Sayf(tm.GREEN, f, i...)
}

func (l Log) Errorf(f string, i ...interface{}) {
	l.Sayf(tm.RED, f, i...)
}

func (l Log) Warningf(f string, i ...interface{}) {
	l.Sayf(tm.YELLOW, f, i...)
}

func (_ Log) Sayf(color int, f string, i ...interface{}) {
	s := fmt.Sprintf(f, i...)
	tm.Println(tm.Color(tm.Bold("[INL] "), color) + s)
	tm.Flush()
}

func (l Log) Debugf(f string, i ...interface{}) {
	if !*verbose {
		return
	}
	l.Infof(f, i...)
}

func (_ Log) Clear() {
	tm.Clear()
	tm.MoveCursor(1, 1)
	tm.Flush()
}

func (_ Log) Ln() {
	tm.Println()
	tm.Flush()
}

var log Log
