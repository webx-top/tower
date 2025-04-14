package main

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/admpub/fsnotify"
	"github.com/admpub/log"
)

const (
	DefaultWatchedFiles = "go"
	DefaultIngoredPaths = `(\/\.\w+)|(^\.)|(\.\w+$)`
)

var (
	scheduleTime atomic.Int64
	eventTime    = make(map[string]time.Time)
)

type Watcher struct {
	WatchedDir         string
	changed            chan struct{}
	OnChanged          func()
	Watcher            *fsnotify.Watcher
	FilePattern        string
	IgnoredPathPattern string
	OnlyWatchBin       bool
	FileNameSuffix     string
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
	w.changed = make(chan struct{})
	return
}

func (w *Watcher) Watch(ctx context.Context) (err error) {
	for _, dir := range w.dirsToWatch() {
		err = w.Watcher.Add(dir)
		if err != nil {
			return
		}
	}
	filePattern := `\.(` + w.FilePattern + `)$`
	if w.OnlyWatchBin {
		filePattern = regexp.QuoteMeta(BinPrefix) + `[\d]+(\.exe)?$`
	}
	expectedFileReg := regexp.MustCompile(filePattern)
	defer w.Watcher.Close()
	go func() {
		for {
			select {
			case <-w.changed:
				if time.Now().Unix()-scheduleTime.Load() >= 2 {
					w.OnChanged()
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	for {
		select {
		case file := <-w.Watcher.Events:
			if w.Paused {
				log.Info(`== Pause monitoring file changes.`)
				continue
			}
			// Skip TMP files for Sublime Text.
			if checkTMPFile(file.Name) {
				continue
			}
			if !expectedFileReg.MatchString(file.Name) {
				if w.OnlyWatchBin {
					log.Info("== [IGNORE]", file.Name)
				}
				continue
			}
			fileName := filepath.Base(file.Name)
			if w.OnlyWatchBin {
				if !strings.HasPrefix(fileName, BinPrefix) {
					log.Info(`忽略非`, BinPrefix, `前缀文件更改`)
					return
				}
				if len(w.FileNameSuffix) > 0 {
					fileName = strings.TrimSuffix(fileName, w.FileNameSuffix)
				}
				newAppBin := fileName
				fileName = strings.TrimPrefix(fileName, BinPrefix)
				newFileTs, err := strconv.ParseInt(fileName, 10, 64)
				if err != nil {
					log.Error(err)
					continue
				}
				fileName = strings.TrimPrefix(AppBin, BinPrefix)
				oldFileTs, err := strconv.ParseInt(fileName, 10, 64)
				if err != nil {
					log.Error(err)
					continue
				}
				if newFileTs <= oldFileTs {
					log.Info(`新文件时间戳小于旧文件，忽略`)
					continue
				}
				AppBin = newAppBin
			} else {
				if strings.HasPrefix(fileName, BinPrefix) {
					log.Info(`忽略`, fileName, `更改`)
					continue
				}
			}
			mt, isDir := getFileModTime(file.Name)
			if file.Op == fsnotify.Create && isDir {
				w.Watcher.Add(file.Name)
			}
			if t := eventTime[file.Name]; mt.Unix() == t.Unix() {
				log.Debugf("== [SKIP] # %s #", file.String())
				continue
			}
			log.Infof("== [EVEN] %s", file)
			eventTime[file.Name] = mt
			if scheduleTime.Load() < time.Now().Unix() {
				scheduleTime.Store(time.Now().Add(time.Second).Unix())
				log.Warn("== Change detected: ", file.Name)
				select {
				case w.changed <- struct{}{}:
				default:
				}
			}
		case err := <-w.Watcher.Errors:
			log.Warn(err) // No need to exit here
		case <-ctx.Done():
			return nil
		}
	}
}

func (w *Watcher) dirsToWatch() (dirs []string) {
	ignoredPathReg := regexp.MustCompile(w.IgnoredPathPattern)
	matchedDirs := make(map[string]bool)
	dir, _ := filepath.Abs("./")
	matchedDirs[dir] = true
	for _, dir := range strings.Split(w.WatchedDir, `|`) {
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
