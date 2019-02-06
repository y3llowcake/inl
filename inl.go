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
	excludeDir  = flag.String("excludeDir", `^\.`, "regular expression of directory basenames to exclude from watching")
	excludeFile = flag.String("exclude", `^.*\.swp$`, "regular expression of files to exclude from watching")
	throttle    = flag.Duration("throttle", time.Millisecond*10, "a duration of time to wait between a filesystem event and triggering the action")
	wait        = flag.Bool("wait", true, "wait for the action to run to completion")
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	flag.Parse()
	if len(flag.Args()) < 1 {
		fmt.Println("usage: inl COMMAND")
		os.Exit(1)
	}
	log.Clear()
	for {
		watchLoop("./")
	}
}

type Log struct{}

func (_ Log) Infof(f string, i ...interface{}) {
	s := fmt.Sprintf(f, i...)
	tm.Println(tm.Color(tm.Bold("[INL] "), tm.GREEN) + s)
	tm.Flush()
}

func (_ Log) Errorf(f string, i ...interface{}) {
	s := fmt.Sprintf(f, i...)
	tm.Println(tm.Color(tm.Bold("[INL] "), tm.RED) + s)
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

// Recursively watch a filesystem path.
func watchLoop(path string) {
	log.Debugf("creating recursive watches for '%s'", path)
	watcher, err := fsnotify.NewWatcher()
	check(err)
	defer watcher.Close()
	check(watcher.Add(path))
	dircount := 1
	dirskip := 0
	excludeDirRegexp := regexp.MustCompile(*excludeDir)
	excludeFileRegexp := regexp.MustCompile(*excludeFile)
	err = filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			return nil
		}
		log.Debugf("watch directory candidate '%s'", path)
		if excludeDirRegexp.MatchString(f.Name()) {
			// Do not watch this directory.
			dirskip++
			return filepath.SkipDir
		}
		log.Debugf("watching '%s'", path)
		check(watcher.Add(path))
		dircount++
		return nil
	})
	check(err)
	log.Debugf("found %d directories to watch, skipped %d", dircount, dirskip)
	log.Debugf("waiting for events...")
	t := time.NewTimer(time.Hour /*arbitrary*/)
	var trigger *fsnotify.Event
	eventCount := 0

	cmd := invoke()

	for {
		select {
		case event := <-watcher.Events:
			log.Debugf("event: %+v", event)
			if excludeFileRegexp.MatchString(event.Name) {
				log.Debugf("ignoring event: %+v", event)
				break
			}
			if trigger == nil {
				trigger = &event
			}
			eventCount++
			// Reset the timer.
			t.Reset(*throttle)
		case err := <-watcher.Errors:
			check(err)
		case <-t.C:
			// Ticker fired. There was a lull in events and we can now process.
			// Stop any already running command.
			if cmd != nil {
				log.Infof("sending sigkill to %d", cmd.Process.Pid)
				cmd.Process.Signal(os.Kill)
				cmd.Wait()
			}

			log.Ln()
			log.Ln()
			log.Ln()
			log.Clear()
			log.Infof("trigger: %+v (+%d)", trigger, eventCount)
			cmd = invoke()

			eventCount = 0
			trigger = nil
		}
	}
}

func invoke() *exec.Cmd {
	args := []string{"/bin/bash", "-c", "--", strings.Join(flag.Args(), " ")}
	cmd := exec.Command(args[0], args[1:]...)
	log.Infof("command: %+q", args)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Ln()
	check(cmd.Start())

	if *wait {
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
