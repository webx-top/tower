package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"os"

	"github.com/stretchr/testify/assert"
	c "github.com/webx-top/tower/config"
)

func TestCmd(t *testing.T) {
	confFile := `test/dev/configs/tower.yml`
	c.Conf.ConfigFile = confFile
	go startTower()
	err := dialAddress("127.0.0.1:8080", 60)
	if err != nil {
		panic(err)
	}
	defer func() {
		app.Clean()
		fmt.Println("")
		fmt.Println("")
		fmt.Println("")
		fmt.Println("")
		fmt.Println("")
	}()

	assert.Equal(t, "server 1", get("http://127.0.0.1:8080/"))
	assert.Equal(t, "server 1", get("http://127.0.0.1:8080/?k=v1&k=v2&k1=v3")) // Test logging parameters
	assert.Equal(t, "server 1", get("http://127.0.0.1:"+app.Port+"/"))

	app.Clean()
	concurrency := 10
	compileChan := make(chan bool)
	for i := 0; i < concurrency; i++ {
		go func() {
			get("http://127.0.0.1:8080/")
			compileChan <- true
		}()
	}

	for i := 0; i < concurrency; i++ {
		select {
		case <-compileChan:
		case <-time.After(10 * time.Second):
			assert.True(t, false, "Timeout on concurrency testing.")
		}
	}

	// test app exits unexpectedly
	assert.Contains(t, get("http://127.0.0.1:8080/exit"), "App quit unexpetedly") // should restart the application

	// test error page
	highlightCode := `<dd class="codes bold">&nbsp;&nbsp;&nbsp;&nbsp;`
	assert.Contains(t, get("http://127.0.0.1:8080/panic"), "Panic !!")                          // should be able to detect panic
	assert.Contains(t, get("http://127.0.0.1:8080/panic"), highlightCode+`panic(errors.New`)    // should show code snippet
	assert.Contains(t, get("http://127.0.0.1:8080/panic"), `<dt class="numbers bold">40`)       // should show line number
	assert.Contains(t, get("http://127.0.0.1:8080/error"), "runtime error: index out of range") // should be able to detect runtime error
	assert.Contains(t, get("http://127.0.0.1:8080/error"), highlightCode+`paths[0]`)            // should show code snippet
	assert.Contains(t, get("http://127.0.0.1:8080/error"), `<dt class="numbers bold">18`)       // should show line number
	/*
		defer exec.Command("git", "checkout", "test").Run()

		exec.Command("cp", "test/files/server2.go_", "test/server1.go").Run()
	*/

	os.Rename(`test/dev/server1.go`, `test/dev/server1.go_`)
	reverse := func() {
		os.Remove(`test/dev/server1.go`)
		os.Rename(`test/dev/server1.go_`, `test/dev/server1.go`)
	}
	err = copy(`test/dev/files/server2.go_`, `test/dev/server1.go`)
	if err != nil {
		reverse()
		panic(err)
	}

	time.Sleep(5 * time.Second)
	assert.Equal(t, "server 2", get("http://127.0.0.1:8080/"))

	//exec.Command("cp", "test/files/error.go_", "test/server1.go").Run()

	err = copy(`test/dev/files/error.go_`, `test/dev/server1.go`)
	if err != nil {
		reverse()
		panic(err)
	}

	//time.Sleep(5 * time.Second)
	//只有编译成功后还会生效，所以取消本项测试
	//assert.Contains(t, get("http://127.0.0.1:8080/"), "Build Error")
	time.Sleep(2 * time.Second)
	reverse()
}

func get(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	b, _ := ioutil.ReadAll(resp.Body)
	return string(b)
}

func copy(src string, dest string) error {
	b, e := ioutil.ReadFile(src)
	if e != nil {
		return e
	}
	e = ioutil.WriteFile(dest, b, os.ModePerm)
	return e
}
