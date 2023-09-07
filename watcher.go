package main

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/admpub/fsnotify"
	"github.com/admpub/log"
)

const (
	DefaultWatchedFiles = "go"
	DefaultIngoredPaths = `(\/\.\w+)|(^\.)|(\.\w+$)`
)

var (
	eventTime    = make(map[string]time.Time)
	scheduleTime time.Time
)

type Watcher struct {
	WatchedDir         string
	Changed            bool
	OnChanged          func(string)
	Watcher            *fsnotify.Watcher
	FilePattern        string
	IgnoredPathPattern string
	OnlyWatchBin       bool
	Paused             bool
}

func NewWatcher(dir, filePattern, ignoredPathPattern string) (w Watcher) {
	w.WatchedDir = dir
	w.FilePattern = DefaultWatchedFiles
	w.IgnoredPathPattern = DefaultIngoredPaths
	if len(filePattern) != 0 {
		w.FilePattern = filePattern
	}
	if len(ignoredPathPattern) != 0 {
		w.IgnoredPathPattern = ignoredPathPattern
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	w.Watcher = watcher

	return
}

func (this *Watcher) Watch(ctx context.Context) (err error) {
	for _, dir := range this.dirsToWatch() {
		err = this.Watcher.Add(dir)
		if err != nil {
			return
		}
	}
	filePattern := `\.(` + this.FilePattern + `)$`
	if this.OnlyWatchBin {
		filePattern = regexp.QuoteMeta(BinPrefix) + `[\d]+(\.exe)?$`
	}
	expectedFileReg := regexp.MustCompile(filePattern)
	defer this.Watcher.Close()
	ch := make(chan string, 1)
	go func() {
		for {
			select {
			case filePath, ok := <-ch:
				if !ok {
					return
				}
				scheduleTime = time.Now().Add(time.Second)
				log.Warn("== Change detected: ", filePath)
				this.Changed = true
				if this.OnChanged != nil {
					this.OnChanged(filePath)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	for {
		select {
		case file := <-this.Watcher.Events:
			if this.Paused {
				log.Info(`== Pause monitoring file changes.`)
				continue
			}
			// Skip TMP files for Sublime Text.
			if checkTMPFile(file.Name) {
				continue
			}
			if !expectedFileReg.MatchString(file.Name) {
				if this.OnlyWatchBin {
					log.Info("== [IGNORE]", file.Name)
				}
				continue
			}
			mt, isDir := getFileModTime(file.Name)
			if file.Op == fsnotify.Create && isDir {
				this.Watcher.Add(file.Name)
			}
			if t := eventTime[file.Name]; mt.Unix() == t.Unix() {
				log.Debugf("== [SKIP] # %s #", file.String())
				continue
			}
			log.Infof("== [EVEN] %s", file)
			eventTime[file.Name] = mt
			if scheduleTime.Before(time.Now()) {
				ch <- file.Name
			}
		case err := <-this.Watcher.Errors:
			log.Warn(err) // No need to exit here
		case <-ctx.Done():
			return nil
		}
	}
}

func (this *Watcher) dirsToWatch() (dirs []string) {
	ignoredPathReg := regexp.MustCompile(this.IgnoredPathPattern)
	matchedDirs := make(map[string]bool)
	dir, _ := filepath.Abs("./")
	matchedDirs[dir] = true
	for _, dir := range strings.Split(this.WatchedDir, `|`) {
		if dir == "" {
			continue
		}
		dir, _ := filepath.Abs(dir)
		f, err := os.Open(dir)
		if err != nil {
			continue
		}
		fi, err := f.Stat()
		f.Close()
		if err != nil {
			log.Errorf("Fail to get file information[ %s ]", err)
			continue
		}
		if !fi.IsDir() {
			continue
		}
		log.Debug("")
		log.Debug("")
		log.Debug("Watch directory: ", dir)
		log.Debug("==================================================================")
		filepath.Walk(dir, func(filePath string, info os.FileInfo, e error) (err error) {
			if e != nil {
				return e
			}
			filePath = strings.Replace(filePath, "\\", "/", -1)
			if !info.IsDir() || ignoredPathReg.MatchString(filePath) || ignoredPathReg.MatchString(filePath+`/`) {
				return
			}
			if matchedDirs[filePath] {
				return
			}
			log.Debug("    ->", filePath)
			matchedDirs[filePath] = true
			return
		})
		log.Debug("")
		log.Debug("")
	}
	for dir := range matchedDirs {
		dirs = append(dirs, dir)
	}
	return
}

func (this *Watcher) Reset() {
	this.Changed = false
}

// checkTMPFile returns true if the event was for TMP files.
func checkTMPFile(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".tmp")
}

// getFileModTime retuens unix timestamp of `os.File.ModTime` by given path.
func getFileModTime(path string) (time.Time, bool) {
	path = strings.Replace(path, "\\", "/", -1)
	f, err := os.Open(path)
	if err != nil {
		log.Errorf("Fail to open file[ %s ]", err)
		return time.Now(), false
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		log.Errorf("Fail to get file information[ %s ]", err)
		return time.Now(), false
	}

	return fi.ModTime(), fi.IsDir()
}
