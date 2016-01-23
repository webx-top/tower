package main

import (
	"errors"
	"fmt"
	"net"
	"time"
)

func dialAddress(address string, timeOut int) (err error) {
	for {
		select {
		case <-time.After(1 * time.Second):
			_, err = net.Dial("tcp", address)
			if err == nil {
				return
			}
		case <-time.After(5 * time.Second):
			fmt.Println("== Waiting for " + address)
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
		panic(err)
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
