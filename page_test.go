package main

import (
	"errors"
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/webx-top/com"
)

var _ = assert.Equal

func catchPanic() (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch r := r.(type) {
			case error:
				err = r
			default:
				err = fmt.Errorf("%v", r)
			}
			stack := make([]byte, 1024)
			length := runtime.Stack(stack, true)
			err = errors.New(string(stack[:length]))
		}
	}()
	//panic(errors.New(`Panic !!`))
	var a []int
	a[0] = 10
	return
}

func TestPanic(t *testing.T) {
	err := catchPanic()
	if err != nil {
		message, trace, appIndex := extractAppErrorInfo(err.Error())
		//github.com/webx-top/tower.catchPanic.func1(0xc04211de80)
		assert.Contains(t, message[0], `github.com/webx-top/tower.catchPanic.func1`)
		//C:/GoWork/src/github.com/webx-top/tower/page_test.go:25 +0x139
		assert.Contains(t, message[1], `github.com/webx-top/tower/page_test.go:25 `)
		fmt.Println(`message:`)
		com.Dump(message)
		fmt.Println(`trace:`)
		com.Dump(trace)
		fmt.Println(`appIndex:`)
		com.Dump(appIndex)

		assert.Equal(t, 25, trace[0].Line)
		assert.Equal(t, `page_test.go`, trace[0].File)
		assert.Equal(t, true, trace[0].AppFile)
		assert.Contains(t, trace[0].Func, `github.com/webx-top/tower.catchPanic.func1`)
	}
}
