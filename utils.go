package main

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/admpub/log"
	"github.com/webx-top/com"
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
	var fn func() bool
	if len(args) > 0 {
		fn = args[0]
	}
	oneSecondTimer := time.NewTimer(1 * time.Second)
	defer func() {
		oneSecondTimer.Stop()
	}()
	startTime := time.Now()
	timeoutDur := time.Duration(timeOut) * time.Second
	for range oneSecondTimer.C {
		conn, err := net.DialTimeout("tcp", address, timeoutDur)
		if err == nil {
			conn.Close()
			return err
		}
		if fn != nil && !fn() {
			return nil
		}
		if time.Now().After(startTime.Add(timeoutDur)) {
			return errors.New(`Time out`)
		}
		log.Warn(`failed to listen on %s: %v, starting retry`, err)
	}
	return err
}

func isFreePort(port string) bool {
	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err == nil {
		conn.Close()
		return true
	}
	return false
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

func parseParams(param string) []string {
	if param[0] != ':' {
		return com.ParseArgs(param)
	}

	//:<分割符>:<参数>
	delim := ` `
	param = strings.TrimPrefix(param, `:`)
	if pos := strings.Index(param, `:`); pos > 0 {
		delim = param[0:pos]
		param = param[pos+1:]
	}
	return strings.Split(param, delim)
}
