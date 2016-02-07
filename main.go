package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/admpub/confl"
)

const ConfigName = ".tower.yml"

var (
	_appMainFile   *string
	_appPort       *string
	_pxyPort       *string
	_appBuildDir   *string
	_portParamName *string
	_verbose       *bool
	_configFile    *string

	app   App
	build string = "1"
)

func main() {
	_appMainFile = flag.String("m", "main.go", "path to your app's main file.")
	_appPort = flag.String("p", "5000", "port of your app.")
	_pxyPort = flag.String("r", "8080", "proxy port of your app.")
	_appBuildDir = flag.String("o", "", "save the executable file the folder.")
	_portParamName = flag.String("n", "", "app's port param name.")
	_verbose = flag.Bool("v", false, "show more stuff.")
	_configFile = flag.String("c", ConfigName, "yaml configuration file location.")

	flag.Parse()

	args := flag.Args()
	if len(args) == 1 && args[0] == "init" {
		generateExampleConfig()
		return
	}
	startTower()
}

func generateExampleConfig() {
	_, file, _, _ := runtime.Caller(0)
	exampleConfig := path.Dir(file) + "/tower.yml"
	exec.Command("cp", exampleConfig, ConfigName).Run()
	fmt.Println("== Generated config file " + ConfigName)
}

func startTower() {
	var (
		appMainFile        = *_appMainFile
		appPort            = *_appPort
		pxyPort            = *_pxyPort
		appBuildDir        = *_appBuildDir
		portParamName      = *_portParamName
		configFile         = *_configFile
		verbose            = *_verbose
		allowBuild         = build == "1"
		suffix             = ".exe"
		_suffix            = ""
		watchedFiles       string
		watchedOtherDir    string
		ignoredPathPattern string
	)
	if configFile == "" {
		configFile = ConfigName
	}
	contents, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Println(err)
	} else {
		newmap := map[string]string{}
		yamlErr := confl.Unmarshal(contents, &newmap)
		if yamlErr != nil {
			fmt.Println(yamlErr)
		}
		appPort, _ = newmap["app_port"]
		pxyPort, _ = newmap["pxy_port"]
		appBuildDir, _ = newmap["app_buildDir"] //编译模式下有效
		portParamName, _ = newmap["app_portParamName"]
		watchedFiles, _ = newmap["watch"]
		watchedOtherDir, _ = newmap["watch_otherDir"] //编译模式下有效
		ignoredPathPattern, _ = newmap["watch_ignoredPath"]
		if pxyPort == "" {
			pxyPort = ProxyPort
		}
		if allowBuild {
			appMainFile, _ = newmap["main"] //编译模式下有效
		} else {
			appMainFile, _ = newmap["exec"] //非编译模式下有效
			if appMainFile == "" {
				fmt.Println("请设置exec参数用来指定执行文件位置")
				time.Sleep(time.Second * 10)
				return
			}
			f, err := os.Open(appMainFile)
			if err == nil {
				_, err = f.Stat()
			}
			f.Close()
			if err != nil {
				fmt.Println(err)
				time.Sleep(time.Second * 10)
				return
			}
			fileName := filepath.Base(appMainFile)
			AppBin = fileName
			if strings.HasSuffix(AppBin, suffix) {
				AppBin = strings.TrimSuffix(AppBin, suffix)
				_suffix = suffix
			}
			nameOk := strings.HasPrefix(AppBin, BinPrefix)
			if nameOk {
				fileName := strings.TrimPrefix(AppBin, BinPrefix)
				_, err = strconv.ParseInt(fileName, 10, 64)
				if err != nil {
					nameOk = false
				}
			}
			if !nameOk {
				fmt.Println("exec参数指定的可执行文件名称格式应该为：", BinPrefix+"0"+_suffix, "。")
				fmt.Println("其中的“0”是代表版本号的整数，请修改为此格式。")
				time.Sleep(time.Second * 300)
				return
			}
		}
	}

	err = dialAddress("127.0.0.1:"+appPort, 1)
	if err == nil {
		fmt.Println("Error: port (" + appPort + ") already in used.")
		os.Exit(1)
	}

	if verbose {
		fmt.Println("== Application Info")
		fmt.Printf("  build app with: %s\n", appMainFile)
		fmt.Printf("  redirect requests from localhost:%s to localhost:%s\n\n", ProxyPort, appPort)
	}

	app = NewApp(appMainFile, appPort, appBuildDir, portParamName)
	if watchedOtherDir != "" {
		watchedOtherDir += "|" + app.Root
	}
	watcher := NewWatcher(watchedOtherDir, watchedFiles, ignoredPathPattern)
	proxy := NewProxy(&app, &watcher)
	if allowBuild {
		watcher.OnChanged = func(file string) {
			fileName := filepath.Base(file)
			if strings.HasPrefix(fileName, BinPrefix) {
				watcher.Reset()
				return
			}
			if !app.SupportMutiPort() {
				fmt.Println(`Unspecified switchable other ports.`)
				return
			}
			port := app.UseRandPort()
			for i := 0; i < 3 && port == app.Port; i++ {
				app.Clean()
				time.Sleep(time.Second)
				port = app.UseRandPort()
			}
			if port == app.Port {
				fmt.Println(`取得的端口与当前端口相同，无法编译切换`)
				return
			}
			watcher.Reset()
			err = app.Start(true, port)
			if err != nil {
				fmt.Println(err)
			}
		}
	} else {
		watcher.OnChanged = func(file string) {
			watcher.Reset()
			if !app.SupportMutiPort() {
				fmt.Println(`Unspecified switchable other ports.`)
				return
			}
			port := app.UseRandPort()
			for i := 0; i < 3 && port == app.Port; i++ {
				app.Clean()
				time.Sleep(time.Second)
				port = app.UseRandPort()
			}
			if port == app.Port {
				fmt.Println(`取得的端口与当前端口相同，无法切换`)
				return
			}

			fileName := filepath.Base(file)
			if !strings.HasPrefix(fileName, BinPrefix) {
				return
			}
			if _suffix != "" {
				fileName = strings.TrimSuffix(fileName, _suffix)
			}
			newAppBin := fileName
			fileName = strings.TrimPrefix(fileName, BinPrefix)
			newFileTs, err := strconv.ParseInt(fileName, 10, 64)
			if err != nil {
				fmt.Println(err)
				return
			}
			fileName = strings.TrimPrefix(AppBin, BinPrefix)
			oldFileTs, err := strconv.ParseInt(fileName, 10, 64)
			if err != nil {
				fmt.Println(err)
				return
			}
			if newFileTs <= oldFileTs {
				return
			}
			AppBin = newAppBin
			err = app.Start(true, port)
			if err != nil {
				fmt.Println(err)
			}
		}
		watcher.OnlyWatchBin = true
		app.DisabledBuild = true
	}
	proxy.Port = pxyPort
	go func() {
		mustSuccess(watcher.Watch())
	}()
	err = app.Start(true, app.Port)
	if err != nil {
		fmt.Println(err)
	}
	mustSuccess(proxy.Listen())
}
