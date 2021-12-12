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
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// IsDir returns true if given path is a directory,
// or returns false when it's a file or does not exist.
func IsDir(dir string) bool {
	f, e := os.Stat(dir)
	if e != nil {
		return false
	}
	return f.IsDir()
}

func statDir(dirPath, recPath string, includeDir, isDirOnly bool) ([]string, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	fis, err := dir.Readdir(0)
	if err != nil {
		return nil, err
	}

	var statList []string
	for _, fi := range fis {
		if strings.Contains(fi.Name(), ".DS_Store") {
			continue
		}

		relPath := filepath.Join(recPath, fi.Name())
		curPath := filepath.Join(dirPath, fi.Name())
		if fi.IsDir() {
			if includeDir {
				statList = append(statList, relPath+"/")
			}
			s, err := statDir(curPath, relPath, includeDir, isDirOnly)
			if err != nil {
				return nil, err
			}
			statList = append(statList, s...)
		} else if !isDirOnly {
			statList = append(statList, relPath)
		}
	}
	return statList, nil
}

// StatDir gathers information of given directory by depth-first.
// It returns slice of file list and includes subdirectories if enabled;
// it returns error and nil slice when error occurs in underlying functions,
// or given path is not a directory or does not exist.
//
// Slice does not include given path itself.
// If subdirectories is enabled, they will have suffix '/'.
func StatDir(rootPath string, includeDir ...bool) ([]string, error) {
	if !IsDir(rootPath) {
		return nil, errors.New("not a directory or does not exist: " + rootPath)
	}

	isIncludeDir := false
	if len(includeDir) >= 1 {
		isIncludeDir = includeDir[0]
	}
	return statDir(rootPath, "", isIncludeDir, false)
}

// GetAllSubDirs returns all subdirectories of given root path.
// Slice does not include given path itself.
func GetAllSubDirs(rootPath string) ([]string, error) {
	if !IsDir(rootPath) {
		return nil, errors.New("not a directory or does not exist: " + rootPath)
	}
	return statDir(rootPath, "", true, true)
}

// CopyDir copy files recursively from source to target directory.
//
// The filter accepts a function that process the path info.
// and should return true for need to filter.
//
// It returns error when error occurs in underlying functions.
func CopyDir(srcPath, destPath string, filters ...func(filePath string) bool) error {
	// Check if target directory exists.
	if !IsExist(destPath) {
		err := os.MkdirAll(destPath, os.ModePerm)
		if err != nil {
			return err
		}
	}

	// Gather directory info.
	infos, err := StatDir(srcPath, true)
	if err != nil {
		return err
	}

	var filter func(filePath string) bool
	if len(filters) > 0 {
		filter = filters[0]
	}

	for _, info := range infos {
		if filter != nil && filter(info) {
			continue
		}

		curPath := filepath.Join(destPath, info)
		if strings.HasSuffix(info, "/") {
			err = os.MkdirAll(curPath, os.ModePerm)
		} else {
			err = Copy(filepath.Join(srcPath, info), curPath)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func FindNotExistsDirs(dir string) ([]string, error) {
	var notExistsDirs []string
	oldParent := dir
	parent := filepath.Dir(dir)
	for oldParent != parent {
		_, err := os.Stat(parent)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return notExistsDirs, err
		}
		notExistsDirs = append(notExistsDirs, parent)
		oldParent = parent
		parent = filepath.Dir(parent)
	}
	return notExistsDirs, nil
}

func MkdirAll(dir string, mode os.FileMode) error {
	if fi, err := os.Stat(dir); err == nil {
		if fi.IsDir() {
			if fi.Mode().Perm() != mode.Perm() {
				return os.Chmod(dir, mode)
			}
			return nil
		}
	}
	needChmodDirs, err := FindNotExistsDirs(dir)
	if err != nil {
		return err
	}
	//Dump(needChmodDirs)

	err = os.MkdirAll(dir, mode)
	if err != nil {
		return err
	}
	err = os.Chmod(dir, mode)
	if err != nil {
		return err
	}
	for _, dir := range needChmodDirs {
		err = os.Chmod(dir, mode)
		if err != nil {
			return err
		}
	}
	return err
}
