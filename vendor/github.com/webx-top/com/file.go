// Copyright 2013 com authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package com

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Storage unit constants.
const (
	Byte  = 1
	KByte = Byte * 1024
	MByte = KByte * 1024
	GByte = MByte * 1024
	TByte = GByte * 1024
	PByte = TByte * 1024
	EByte = PByte * 1024
)

func logn(n, b float64) float64 {
	return math.Log(n) / math.Log(b)
}

func humanateBytes(s uint64, base float64, sizes []string) string {
	if s < 10 {
		return fmt.Sprintf("%dB", s)
	}
	e := math.Floor(logn(float64(s), base))
	suffix := sizes[int(e)]
	val := float64(s) / math.Pow(base, math.Floor(e))
	f := "%.0f"
	if val < 10 {
		f = "%.1f"
	}

	return fmt.Sprintf(f+"%s", val, suffix)
}

// HumaneFileSize calculates the file size and generate user-friendly string.
func HumaneFileSize(s uint64) string {
	sizes := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	return humanateBytes(s, 1024, sizes)
}

// FileMTime returns file modified time and possible error.
func FileMTime(file string) (int64, error) {
	f, err := os.Stat(file)
	if err != nil {
		return 0, err
	}
	return f.ModTime().Unix(), nil
}

// TrimFileName trim the file name
func TrimFileName(ppath string) string {
	if len(ppath) == 0 {
		return ppath
	}
	for i := len(ppath) - 1; i >= 0; i-- {
		if ppath[i] == '/' || ppath[i] == '\\' {
			if i+1 < len(ppath) {
				return ppath[0 : i+1]
			}
			return ppath
		}
	}
	return ``
}

func BaseFileName(ppath string) string {
	if len(ppath) == 0 {
		return ppath
	}
	for i := len(ppath) - 1; i >= 0; i-- {
		if ppath[i] == '/' || ppath[i] == '\\' {
			if i+1 < len(ppath) {
				return ppath[i+1:]
			}
			return ``
		}
	}
	return ppath
}

func HasPathSeperatorPrefix(ppath string) bool {
	return strings.HasPrefix(ppath, `/`) || strings.HasPrefix(ppath, `\`)
}

func HasPathSeperatorSuffix(ppath string) bool {
	return strings.HasSuffix(ppath, `/`) || strings.HasSuffix(ppath, `\`)
}

var pathSeperatorRegex = regexp.MustCompile(`(\\|/)`)
var winPathSeperatorRegex = regexp.MustCompile(`[\\]+`)

func GetPathSeperator(ppath string) string {
	matches := pathSeperatorRegex.FindAllStringSubmatch(ppath, 1)
	if len(matches) > 0 && len(matches[0]) > 1 {
		return matches[0][1]
	}
	return ``
}

func RealPath(fullPath string) string {
	if len(fullPath) == 0 {
		return fullPath
	}
	var root string
	var pathSeperator string
	if strings.HasPrefix(fullPath, `/`) {
		pathSeperator = `/`
	} else {
		pathSeperator = GetPathSeperator(fullPath)
		if pathSeperator == `/` {
			fullPath = `/` + fullPath
		} else {
			cleanedFullPath := winPathSeperatorRegex.ReplaceAllString(fullPath, `\`)
			parts := strings.SplitN(cleanedFullPath, `\`, 2)
			if !strings.HasSuffix(parts[0], `:`) {
				if len(parts[0]) > 0 {
					parts[0] = `c:\` + parts[0]
				} else {
					parts[0] = `c:`
				}
			}
			root = parts[0] + `\`
			if len(parts) != 2 {
				return fullPath
			}
			fullPath = parts[1]
		}
	}
	parts := pathSeperatorRegex.Split(fullPath, -1)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == `.` {
			continue
		}
		if part == `..` {
			if len(result) <= 1 {
				continue
			}
			result = result[0 : len(result)-1]
			continue
		}
		result = append(result, part)
	}
	return root + strings.Join(result, pathSeperator)
}

func SplitFileDirAndName(ppath string) (dir string, name string) {
	if len(ppath) == 0 {
		return
	}
	for i := len(ppath) - 1; i >= 0; i-- {
		if ppath[i] == '/' || ppath[i] == '\\' {
			if i+1 < len(ppath) {
				return ppath[0:i], ppath[i+1:]
			}
			dir = ppath[0:i]
			return
		}
	}
	name = ppath
	return
}

// FileSize returns file size in bytes and possible error.
func FileSize(file string) (int64, error) {
	f, err := os.Stat(file)
	if err != nil {
		return 0, err
	}
	return f.Size(), nil
}

// Copy copies file from source to target path.
func Copy(src, dest string) error {
	// Gather file information to set back later.
	si, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("couldn't open source file: %w", err)
	}

	// Handle symbolic link.
	if si.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		// NOTE: os.Chmod and os.Chtimes don't recoganize symbolic link,
		// which will lead "no such file or directory" error.
		return os.Symlink(target, dest)
	}

	sr, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("couldn't open source file: %w", err)
	}
	defer sr.Close()

	dw, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("couldn't open dest file: %w", err)
	}
	defer dw.Close()

	if _, err = io.Copy(dw, sr); err != nil {
		return fmt.Errorf("writing to output file failed: %w", err)
	}
	dw.Sync()

	// Set back file information.
	if err = os.Chtimes(dest, si.ModTime(), si.ModTime()); err != nil {
		return err
	}
	return os.Chmod(dest, si.Mode())
}

func Remove(name string) error {
	err := os.Remove(name)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return err
}

/*
GoLang: os.Rename() give error "invalid cross-device link" for Docker container with Volumes.
Rename(source, destination) will work moving file between folders
*/
func Rename(src, dest string) error {
	err := os.Rename(src, dest)
	if err == nil {
		return nil
	}
	if !strings.HasSuffix(err.Error(), `invalid cross-device link`) {
		return err
	}
	err = Copy(src, dest)
	if err != nil {
		if !strings.HasSuffix(err.Error(), `operation not permitted`) {
			return err
		}
	}
	// The copy was successful, so now delete the original file
	err = Remove(src)
	if err != nil {
		return fmt.Errorf("failed removing original file: %w", err)
	}
	return nil
}

// WriteFile writes data to a file named by filename.
// If the file does not exist, WriteFile creates it
// and its upper level paths.
func WriteFile(filename string, data []byte) error {
	os.MkdirAll(filepath.Dir(filename), os.ModePerm)
	return os.WriteFile(filename, data, 0655)
}

// CreateFile create file
func CreateFile(filename string) (fp *os.File, err error) {
	fp, err = os.Create(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
		err = MkdirAll(filepath.Dir(filename), os.ModePerm)
		if err != nil {
			return
		}
		fp, err = os.Create(filename)
	}
	return
}

// IsFile returns true if given path is a file,
// or returns false when it's a directory or does not exist.
func IsFile(filePath string) bool {
	f, e := os.Stat(filePath)
	if e != nil {
		return false
	}
	return !f.IsDir()
}

// IsExist checks whether a file or directory exists.
// It returns false when the file or directory does not exist.
func IsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func Unlink(file string) bool {
	return os.Remove(file) == nil
}

// SaveFile saves content type '[]byte' to file by given path.
// It returns error when fail to finish operation.
func SaveFile(filePath string, b []byte) (int, error) {
	os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	fw, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer fw.Close()
	return fw.Write(b)
}

// SaveFileS saves content type 'string' to file by given path.
// It returns error when fail to finish operation.
func SaveFileS(filePath string, s string) (int, error) {
	return SaveFile(filePath, []byte(s))
}

// ReadFile reads data type '[]byte' from file by given path.
// It returns error when fail to finish operation.
func ReadFile(filePath string) ([]byte, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return []byte(""), err
	}
	return b, nil
}

// ReadFileS reads data type 'string' from file by given path.
// It returns error when fail to finish operation.
func ReadFileS(filePath string) (string, error) {
	b, err := ReadFile(filePath)
	return string(b), err
}

var (
	selfPath string
	selfDir  string
)

func SelfPath() string {
	if len(selfPath) == 0 {
		selfPath, _ = filepath.Abs(os.Args[0])
	}
	return selfPath
}

func SelfDir() string {
	if len(selfDir) == 0 {
		selfDir = filepath.Dir(SelfPath())
	}
	return selfDir
}

// FileExists reports whether the named file or directory exists.
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// SearchFile search a file in paths.
// this is offen used in search config file in /etc ~/
func SearchFile(filename string, paths ...string) (fullpath string, err error) {
	for _, path := range paths {
		if fullpath = filepath.Join(path, filename); FileExists(fullpath) {
			return
		}
	}
	err = errors.New(fullpath + " not found in paths")
	return
}

// GrepFile like command grep -E
// for example: GrepFile(`^hello`, "hello.txt")
// \n is striped while read
func GrepFile(patten string, filename string) (lines []string, err error) {
	re, err := regexp.Compile(patten)
	if err != nil {
		return
	}

	lines = make([]string, 0)
	err = SeekFileLines(filename, func(line string) error {
		if re.MatchString(line) {
			lines = append(lines, line)
		}
		return nil
	})
	return lines, err
}

func SeekFileLines(filename string, callback func(string) error) error {
	fd, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fd.Close()
	return SeekLines(fd, callback)
}

type LineReader interface {
	ReadLine() (line []byte, isPrefix bool, err error)
}

func SeekLines(r io.Reader, callback func(string) error) (err error) {
	var reader LineReader
	var prefix string
	if rd, ok := r.(LineReader); ok {
		reader = rd
	} else {
		reader = bufio.NewReader(r)
	}
	for {
		byteLine, isPrefix, er := reader.ReadLine()
		if er != nil && er != io.EOF {
			return er
		}
		line := string(byteLine)
		if isPrefix {
			prefix += line
			continue
		}
		line = prefix + line
		if e := callback(line); e != nil {
			return e
		}
		prefix = ""
		if er == io.EOF {
			break
		}
	}
	return nil
}

func Readbuf(r io.Reader, length int) ([]byte, error) {
	buf := make([]byte, length)
	offset := 0

	for offset < length {
		read, err := r.Read(buf[offset:])
		if err != nil {
			return buf, err
		}

		offset += read
	}

	return buf, nil
}

func Bytes2readCloser(b []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewBuffer(b))
}

const HalfSecond = 500 * time.Millisecond // 0.5秒
var debugFileIsCompleted bool

// FileIsCompleted 等待文件有数据且已写完
// 费时操作 放在子线程中执行
// @param file  文件
// @param start 需要传入 time.Now.Local()，用于兼容遍历的情况
// @return true:已写完 false:外部程序阻塞或者文件不存在
// 翻译自：https://blog.csdn.net/northernice/article/details/115986671
func FileIsCompleted(file *os.File, start time.Time) (bool, error) {
	var (
		fileLength  int64
		i           int
		waitTime    = HalfSecond
		lastModTime time.Time
		finished    int
	)
	for {
		fi, err := file.Stat()
		if err != nil {
			return false, err
		}
		if debugFileIsCompleted {
			fmt.Printf("FileIsCompleted> size:%d (time:%s) [finished:%d]\n", fi.Size(), fi.ModTime(), finished)
		}
		//文件在外部一直在填充数据，每次进入循环体时，文件大小都会改变，一直到不改变时，说明文件数据填充完毕 或者文件大小一直都是0(外部程序阻塞)
		//判断文件大小是否有改变
		if fi.Size() > fileLength { //有改变说明还未写完
			fileLength = fi.Size()
			if i%120 == 0 { //每隔1分钟输出一次日志 (i为120时：120*500/1000=60秒)
				log.Println("文件: " + fi.Name() + " 正在被填充，请稍候...")
			}
			time.Sleep(waitTime) //半秒后再循环一次
			lastModTime = fi.ModTime()
		} else { //否则：只能等于 不会小于，等于有两种情况，一种是数据写完了，一种是外部程序阻塞了，导致文件大小一直为0
			if lastModTime.IsZero() {
				lastModTime = fi.ModTime()
			} else {
				if fileLength != fi.Size() {
					fileLength = fi.Size()
				} else if lastModTime.Equal(fi.ModTime()) {
					if fi.Size() != 0 {
						if finished < 3 {
							time.Sleep(waitTime)
							finished++
							continue
						}
						return true, nil
					}
				}
			}
			//等待外部程序开始写 只等60秒 120*500/1000=60秒
			//每隔1分钟输出一次日志 (i为120时：120*500/1000=60秒)
			if i%120 == 0 {
				log.Println("文件: " + fi.Name() + " 大小为" + strconv.FormatInt(fi.Size(), 10) + "，正在等待外部程序填充，已等待：" + time.Since(start).String())
			}

			//如果一直(i为120时：120*500/1000=60秒)等于0，说明外部程序阻塞了
			if i >= 3600 { //120为1分钟 3600为30分钟
				log.Println("文件: " + fi.Name() + " 大小在：" + time.Since(start).String() + " 内始终为" + strconv.FormatInt(fi.Size(), 10) + "，说明：在[程序监测时间内]文件写入进程依旧在运行，程序监测时间结束") //入库未完成或发生阻塞
				return false, nil
			}

			time.Sleep(waitTime)
		}
		if finished > 0 {
			finished = 0
		}
		i++
	}
}
