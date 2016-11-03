package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

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

func dialAddress(address string, timeOut int, args ...func() bool) (err error) {
	seconds := 0
	var fn func() bool
	if len(args) > 0 {
		fn = args[0]
	}
	for {
		select {
		case <-time.After(1 * time.Second):
			conn, err := net.Dial("tcp", address)
			if err == nil {
				conn.Close()
				return err
			}
			//fmt.Println(`[`, seconds, `]`, err)
			if seconds > timeOut {
				return errors.New("Time out")
			}
			seconds++
			if fn != nil && !fn() {
				return nil
			}
		case <-time.After(5 * time.Second):
			fmt.Println("== Waiting for " + address)
			if seconds > timeOut {
				return errors.New("Time out")
			}
			seconds += 5
			if fn != nil && !fn() {
				return
			}
		case <-time.After(time.Duration(timeOut) * time.Second):
			return errors.New("Time out")
		}
	}
	return
}

func isFreePort(port string) bool {
	_, err := net.Dial("tcp", "127.0.0.1:"+port)
	return err != nil
}

func mustSuccess(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func quickSort(arr []int64, start, end int) {
	if start < end {
		i, j := start, end
		key := arr[(start+end)/2]
		for i <= j {
			for arr[i] < key {
				i++
			}
			for arr[j] > key {
				j--
			}
			if i <= j {
				arr[i], arr[j] = arr[j], arr[i]
				i++
				j--
			}
		}

		if start < j {
			quickSort(arr, start, j)
		}
		if end > i {
			quickSort(arr, i, end)
		}
	}
}
