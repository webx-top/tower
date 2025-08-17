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
	"github.com/admpub/rundelay"
)

const (
	DefaultWatchedFiles = "go"
	DefaultIngoredPaths = `(\/\.\w+)|(^\.)|(\.\w+$)`
)

type Watcher struct {
	WatchedDir         string
	OnChanged          func()
	Watcher            *fsnotify.Watcher
	FilePattern        string
	IgnoredPathPattern string
	OnlyWatchBin       bool
	FileNameSuffix     string
	Paused             bool
	compiling          atomic.Bool
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

	delay := time.Second * 2
	dr := rundelay.New(delay, func(_ string) error {
		w.OnChanged()
		w.compiling.Store(false)
		return nil
	})
	defer dr.Close()

	expectedFileReg := regexp.MustCompile(filePattern)
	defer w.Watcher.Close()
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
			fi, err := os.Stat(file.Name)
			if err == nil && fi.IsDir() && file.Op == fsnotify.Create {
				w.Watcher.Add(file.Name)
				continue
			}
			log.Infof("== [EVEN] %s", file)
			log.Warn("== Change detected: ", file.Name)
			w.compiling.Store(true)
			dr.Run(file.Name)
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

func (w *Watcher) Reset() {
}

// checkTMPFile returns true if the event was for TMP files.
func checkTMPFile(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".tmp")
}
