package com

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Zip 压缩为zip
// args: regexpFileName, regexpIgnoreFile
func Zip(srcDirPath string, destFilePath string, args ...*regexp.Regexp) (n int64, err error) {
	root, err := filepath.Abs(srcDirPath)
	if err != nil {
		return 0, err
	}

	f, err := os.Create(destFilePath)
	if err != nil {
		return
	}
	defer f.Close()

	w := zip.NewWriter(f)
	var regexpIgnoreFile, regexpFileName *regexp.Regexp
	argLen := len(args)
	if argLen > 1 {
		regexpIgnoreFile = args[1]
		regexpFileName = args[0]
	} else if argLen == 1 {
		regexpFileName = args[0]
	}
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if regexpIgnoreFile != nil && (regexpIgnoreFile.MatchString(info.Name()) || regexpIgnoreFile.MatchString(path)) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		} else if info.IsDir() {
			return nil
		}
		if regexpFileName != nil && (!regexpFileName.MatchString(info.Name()) && !regexpFileName.MatchString(path)) {
			return nil
		}
		relativePath := strings.TrimPrefix(path, root)
		relativePath = strings.Replace(relativePath, `\`, `/`, -1)
		relativePath = strings.TrimPrefix(relativePath, `/`)
		fw, err := w.Create(relativePath)
		if err != nil {
			return err
		}
		sf, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(fw, sf)
		sf.Close()
		return err
	})

	err = w.Close()
	if err != nil {
		return 0, err
	}

	fi, err := f.Stat()
	if err != nil {
		n = fi.Size()
	}
	return
}

func IllegalFilePath(fpath string) bool {
	if fpath == `..` {
		return true
	}
	if strings.HasSuffix(fpath, `..`) {
		i := len(fpath) - 3
		if fpath[i] == '/' || fpath[i] == '\\' {
			return true
		}
	}
	var dots int
	for _, c := range fpath {
		switch c {
		case '.':
			dots++
		case '/':
			fallthrough
		case '\\':
			if dots > 1 {
				return true
			}
		default:
			dots = 0
		}
	}
	return false
}

// Unzip unzips .zip file to 'destPath'.
// It returns error when fail to finish operation.
func Unzip(srcPath, destPath string) error {
	// Open a zip archive for reading
	r, err := zip.OpenReader(srcPath)
	if err != nil {
		return err
	}
	defer r.Close()

	// Iterate through the files in the archive
	for _, f := range r.File {
		if IllegalFilePath(f.Name) {
			return fmt.Errorf("illegal file path in %s: %v", filepath.Base(srcPath), f.Name)
		}

		fullPath := filepath.Join(destPath, f.Name)
		if f.FileInfo().IsDir() {
			if err = os.MkdirAll(fullPath, f.Mode()); err != nil {
				return err
			}
			continue
		}

		dir := filepath.Dir(f.Name)
		// Create directory before create file
		if err = os.MkdirAll(filepath.Join(destPath, dir), os.ModePerm); err != nil {
			return err
		}

		// Get files from archive
		rc, err := f.Open()
		if err != nil {
			return err
		}
		// Write data to file
		var fw *os.File
		fw, err = os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(fw, rc)
		fw.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// TarGz 压缩为tar.gz
// args: regexpFileName, regexpIgnoreFile
func TarGz(srcDirPath string, destFilePath string, args ...*regexp.Regexp) error {
	fw, err := os.Create(destFilePath)
	if err != nil {
		return err
	}
	defer fw.Close()
	// Gzip writer
	gw := gzip.NewWriter(fw)
	err = tarGz(gw, srcDirPath, args...)
	gw.Close()
	return err
}

func TarGzWithLevel(compressLevel int, srcDirPath string, destFilePath string, args ...*regexp.Regexp) error {
	fw, err := os.Create(destFilePath)
	if err != nil {
		return err
	}
	defer fw.Close()
	// Gzip writer
	var gw *gzip.Writer
	gw, err = gzip.NewWriterLevel(fw, compressLevel)
	if err != nil {
		return err
	}
	err = tarGz(gw, srcDirPath, args...)
	gw.Close()
	return err
}

func tarGz(gw *gzip.Writer, srcDirPath string, args ...*regexp.Regexp) error {
	// Tar writer
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Check if it's a file or a directory
	f, err := os.Open(srcDirPath)
	if err != nil {
		return err
	}
	fi, err := f.Stat()
	f.Close()
	if err != nil {
		return err
	}
	if fi.IsDir() {
		var regexpIgnoreFile, regexpFileName *regexp.Regexp
		argLen := len(args)
		if argLen > 1 {
			regexpIgnoreFile = args[1]
			regexpFileName = args[0]
		} else if argLen == 1 {
			regexpFileName = args[0]
		}
		// handle source directory
		fmt.Println("Cerating tar.gz from directory...")
		if err := tarGzDir(srcDirPath, `.`, tw, regexpFileName, regexpIgnoreFile); err != nil {
			return err
		}
	} else {
		// handle file directly
		fmt.Println("Cerating tar.gz from " + fi.Name() + "...")
		if err := tarGzFile(srcDirPath, fi.Name(), tw, fi); err != nil {
			return err
		}
	}
	fmt.Println("Well done!")
	return err
}

// Deal with directories
// if find files, handle them with tarGzFile
// Every recurrence append the base path to the recPath
// recPath is the path inside of tar.gz
func tarGzDir(srcDirPath string, recPath string, tw *tar.Writer, regexpFileName, regexpIgnoreFile *regexp.Regexp) error {
	// Open source diretory
	dir, err := os.Open(srcDirPath)
	if err != nil {
		return err
	}
	defer dir.Close()

	// Get file info slice
	fis, err := dir.Readdir(0)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		// Append path
		curPath := srcDirPath + "/" + fi.Name()
		if regexpIgnoreFile != nil && (regexpIgnoreFile.MatchString(fi.Name()) || regexpIgnoreFile.MatchString(curPath)) {
			continue
		}
		if regexpFileName != nil && (!regexpFileName.MatchString(fi.Name()) && !regexpFileName.MatchString(curPath)) {
			continue
		}
		// Check it is directory or file
		if fi.IsDir() {
			// Directory
			// (Directory won't add unitl all subfiles are added)
			fmt.Printf("Adding path...%s\n", curPath)
			tarGzDir(curPath, recPath+"/"+fi.Name(), tw, regexpFileName, regexpIgnoreFile)
		} else {
			// File
			fmt.Printf("Adding file...%s\n", curPath)
		}

		tarGzFile(curPath, recPath+"/"+fi.Name(), tw, fi)
	}
	return err
}

// Deal with files
func tarGzFile(srcFile string, recPath string, tw *tar.Writer, fi os.FileInfo) error {
	if fi.IsDir() {
		// Create tar header
		hdr := new(tar.Header)
		// if last character of header name is '/' it also can be directory
		// but if you don't set Typeflag, error will occur when you untargz
		hdr.Name = recPath + "/"
		hdr.Typeflag = tar.TypeDir
		hdr.Size = 0
		//hdr.Mode = 0755 | c_ISDIR
		hdr.Mode = int64(fi.Mode())
		hdr.ModTime = fi.ModTime()

		// Write hander
		err := tw.WriteHeader(hdr)
		if err != nil {
			return err
		}
	} else {
		// File reader
		fr, err := os.Open(srcFile)
		if err != nil {
			return err
		}
		defer fr.Close()

		// Create tar header
		hdr := new(tar.Header)
		hdr.Name = recPath
		hdr.Size = fi.Size()
		hdr.Mode = int64(fi.Mode())
		hdr.ModTime = fi.ModTime()

		// Write hander
		err = tw.WriteHeader(hdr)
		if err != nil {
			return err
		}

		// Write file data
		_, err = io.Copy(tw, fr)
		if err != nil {
			return err
		}
	}
	return nil
}

// UnTarGz ungzips and untars .tar.gz file to 'destPath' and returns sub-directories.
// It returns error when fail to finish operation.
func UnTarGz(srcFilePath string, destDirPath string) ([]string, error) {
	// Create destination directory
	if err := os.MkdirAll(destDirPath, os.ModePerm); err != nil {
		return nil, err
	}

	fr, err := os.Open(srcFilePath)
	if err != nil {
		return nil, err
	}
	defer fr.Close()

	// Gzip reader
	gr, err := gzip.NewReader(fr)
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	// Tar reader
	tr := tar.NewReader(gr)

	dirs := make([]string, 0, 5)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			// End of tar archive
			break
		}

		if IllegalFilePath(hdr.Name) {
			return nil, fmt.Errorf("illegal file path in %s: %v", filepath.Base(srcFilePath), hdr.Name)
		}
		fullPath := filepath.Join(destDirPath, hdr.Name)
		mode := hdr.FileInfo().Mode()

		// Check if it is directory or file
		if hdr.Typeflag != tar.TypeDir {
			// Get files from archive
			// Create directory before create file
			dir := filepath.Dir(hdr.Name)
			if err = os.MkdirAll(filepath.Join(destDirPath, dir), os.ModePerm); err != nil {
				return nil, err
			}

			// Write data to file
			var fw *os.File
			fw, err = os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
			if err != nil {
				return nil, err
			}
			_, err = io.Copy(fw, tr)
			fw.Close()
		} else {
			dirs = AppendUniqueStr(dirs, fullPath)
			err = os.MkdirAll(fullPath, mode)
		}
		if err != nil {
			return nil, err
		}
	}
	return dirs, nil
}
