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
	"sync"

	"gopkg.in/yaml.v1"
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
		appMainFile   = *_appMainFile
		appPort       = *_appPort
		pxyPort       = *_pxyPort
		appBuildDir   = *_appBuildDir
		portParamName = *_portParamName
		configFile    = *_configFile
		verbose       = *_verbose
		allowBuild    = build == "1"
	)
	if configFile == "" {
		configFile = ConfigName
	}
	watchedFiles := ""
	watchedOtherDir := ""
	contents, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Println(err)
	} else {
		newmap := map[string]string{}
		yamlErr := yaml.Unmarshal(contents, &newmap)
		if yamlErr != nil {
			fmt.Println(yamlErr)
		}
		appPort, _ = newmap["app_port"]
		pxyPort, _ = newmap["pxy_port"]
		appBuildDir, _ = newmap["app_buildDir"] //编译模式下有效
		portParamName, _ = newmap["app_portParamName"]
		watchedFiles, _ = newmap["watch"]
		watchedOtherDir, _ = newmap["watch_otherDir"] //编译模式下有效
		if pxyPort == "" {
			pxyPort = ProxyPort
		}
		if allowBuild {
			appMainFile, _ = newmap["main"] //编译模式下有效
		} else {
			appMainFile, _ = newmap["exec"] //非编译模式下有效
			if appMainFile == "" {
				fmt.Println("请设置exec参数用来指定执行文件位置")
				return
			}
			f, err := os.Open(appMainFile)
			if err == nil {
				_, err = f.Stat()
			}
			f.Close()
			if err != nil {
				fmt.Println(err)
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
	watcher := NewWatcher(watchedOtherDir, watchedFiles)
	runApp := func(port string) {
		app.BuildStart.Do(func() {
			err := app.Build()
			if err != nil {
				fmt.Println(err)
			}
			app.BuildStart = &sync.Once{}
		})
		err := app.Run(port)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	if allowBuild {
		watcher.OnChanged = func(file string) {
			if !app.SupportMutiPort() {
				return
			}
			port := app.UseRandPort()
			if port == app.Port {
				return
			}
			watcher.Reset()
			runApp(port)
		}
	} else {
		watcher.OnChanged = func(file string) {
			watcher.Reset()
			if !app.SupportMutiPort() {
				return
			}
			port := app.UseRandPort()
			if port == app.Port {
				return
			}

			fileName := filepath.Base(file)
			prefix := "tower-app-"
			if !strings.HasPrefix(fileName, prefix) {
				return
			}
			fileName = strings.TrimSuffix(fileName, ".exe")
			newAppBin := fileName
			fileName = strings.TrimPrefix(fileName, prefix)
			newFileTs, err := strconv.ParseInt(fileName, 10, 64)
			if err != nil {
				fmt.Println(err)
				return
			}
			fileName = strings.TrimPrefix(AppBin, prefix)
			oldFileTs, err := strconv.ParseInt(fileName, 10, 64)
			if err != nil {
				fmt.Println(err)
				return
			}
			if newFileTs <= oldFileTs {
				return
			}
			AppBin = newAppBin
			runApp(port)
		}
		app.DisabledBuild = true
	}
	runApp(app.Port)
	proxy := NewProxy(&app, &watcher)
	proxy.Port = pxyPort
	go func() {
		mustSuccess(watcher.Watch())
	}()
	mustSuccess(proxy.Listen())
}
