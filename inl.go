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
	excludeDir  = flag.String("exclude_dir", `^\.`, "regular expression of directory basenames to exclude from watching")
	excludeFile = flag.String("exclude", `^.*\.swp$`, "regular expression of files to exclude from watching")
	throttle    = flag.Duration("t", time.Millisecond*100, "a duration of time to wait between a filesystem event and triggering the action")
	noWait      = flag.Bool("n", false, "do not wait for the action to run to completion, use sigkill on the next filesystem change")
)

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

func main() {
	flag.Parse()
	if len(flag.Args()) < 1 {
		fmt.Println("usage: inl COMMAND")
		os.Exit(1)
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
	log.Infof("watching '%s'", path)
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
			log.Ln()
			log.Ln()
			log.Ln()
			log.Clear()
			log.Infof("trigger: %+v (+%d)", trigger, eventCount)

			return
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
