package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/admpub/confl"
	"github.com/admpub/log"
	c "github.com/webx-top/tower/config"
)

func init() {
	log.DefaultLog.Category = `tower`
	log.Sync(true)
	log.DefaultLog.Formatter = func(_ *log.Logger, e *log.Entry) string {
		return e.Message
	}
	log.SetFatalAction(log.ActionExit)
}

const ConfigName = "tower.yml"

var (
	app   App
	build = "1"
)

func main() {
	flag.StringVar(&c.Conf.App.ExecFile, "f", "tower-app-*.exe", "path to your app's main file.")
	flag.StringVar(&c.Conf.App.MainFile, "m", "", "path to your app's main file.")
	flag.StringVar(&c.Conf.App.Port, "p", "5001-5050", "port range of your app.")
	flag.StringVar(&c.Conf.Proxy.Port, "r", "8080", "proxy port of your app.")
	flag.StringVar(&c.Conf.Proxy.Engine, "e", "standard", "fast/standard")
	flag.StringVar(&c.Conf.App.BuildDir, "o", "", "save the executable file the folder.")
	flag.StringVar(&c.Conf.App.PortParamName, "n", "", "app's port param name.")
	flag.StringVar(&c.Conf.App.RunParams, "s", "", "app's run params.")
	flag.StringVar(&c.Conf.App.BuildParams, "b", "", "build params.")
	flag.BoolVar(&c.Conf.Verbose, "v", false, "show more stuff.")
	flag.StringVar(&c.Conf.ConfigFile, "c", ConfigName, "yaml configuration file location.")
	flag.StringVar(&c.Conf.Admin.Password, "w", "", "admin password.")
	flag.StringVar(&c.Conf.Admin.IPs, "i", "127.0.0.1,::1", "admin allow IP.")
	flag.BoolVar(&c.Conf.AutoClear, "a", true, "automatically deletes previously compiled files when you startup Tower in the compile mode")
	flag.StringVar(&c.Conf.LogLevel, "logLevel", "Debug", "logger level(Debug/Info/Warn/Error/Fatal)")
	flag.BoolVar(&c.Conf.Offline, "offline", true, "offline mode")
	flag.BoolVar(&c.Conf.LogRequest, "logRequest", true, "")
	flag.StringVar(&c.Conf.Watch.FileExtension, "fileExtention", "go", "")
	flag.StringVar(&c.Conf.Watch.OtherDir, "watchOtherDir", "", "")
	flag.StringVar(&c.Conf.Watch.IgnoredPath, "watchIgnoredPath", "/\\.git", "")
	prod := flag.String("prod", "", "Production mode")

	flag.Parse()

	args := flag.Args()
	if len(args) > 0 {
		switch args[0] {
		case "init":
			generateExampleConfig()
			return
		case "get":
			if len(args) > 1 {
				a := &App{
					PkgMirrors: make(map[string]string),
				}
				cmdArgs := []string{}
				pkgArgs := []string{}
				for _, arg := range args[1:] {
					if strings.HasPrefix(arg, `-`) {
						cmdArgs = append(cmdArgs, arg)
					} else {
						pkgArgs = append(pkgArgs, arg)
					}
				}
				if len(pkgArgs) > 0 {
					if len(pkgArgs) > 1 && len(pkgArgs[1]) > 0 {
						for _, rep := range strings.Split(pkgArgs[1], `;`) {
							rep = strings.TrimSpace(rep)
							if len(rep) < 1 {
								continue
							}
							r := strings.SplitN(rep, `=>`, 2)
							r[0] = strings.TrimSpace(r[0])
							if len(r[0]) == 0 {
								continue
							}
							if len(r) == 2 {
								r[1] = strings.TrimSpace(r[1])
								a.PkgMirrors[r[0]] = r[1]
							}
						}
					}
					pkgs := [][]string{}
					for _, pkg := range strings.Split(pkgArgs[0], `,`) {
						pkg = strings.TrimSpace(pkg)
						if len(pkg) < 1 {
							continue
						}
						pkgs = append(pkgs, []string{``, pkg})
					}
					a.fetchPkg(pkgs, false, cmdArgs...)
				}
				return

			}
		}
	}
	if !fileExist(c.Conf.ConfigFile) {
		generateExampleConfig()
	}
	if len(*prod) > 0 && atob(*prod) {
		build = "0"
	}
	startTower()
}

func fileExist(path string) bool {
	fi, err := os.Stat(path)
	return (err == nil || os.IsExist(err)) && !fi.IsDir()
}

func saveFile(filePath string, b []byte) (int, error) {
	os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	fw, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer fw.Close()
	return fw.Write(b)
}

func generateExampleConfig() {
	configContent := defaultConfig
	var err error
	/*
		c.Conf.Fixed()
		configContent, err = confl.Marshal(c.Conf)
		if err != nil {
			log.Fatal(err)
			return
		}
	*/
	_, err = saveFile(ConfigName, configContent)
	if err != nil {
		log.Error(err)
		return
	}
	log.Info("== Generated config file " + ConfigName)
}

func atob(a string) bool {
	return a == `1` || a == `true` || a == `on` || a == `yes`
}

func findBinFile(f string) string {
	var prefix, suffix string
	tg := strings.Split(filepath.Base(f), `*`)
	switch len(tg) {
	case 2:
		prefix = tg[0]
		suffix = tg[1]
	default:
		panic(`error format.`)
	}
	var file string
	err := filepath.Walk(filepath.Dir(f), func(filePath string, info os.FileInfo, e error) (err error) {
		if e != nil {
			return e
		}
		if info.IsDir() {
			return
		}
		name := info.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
			file = filePath
			return filepath.SkipDir
		}
		return
	})
	if err != nil && err != filepath.SkipDir {
		panic(err)
	}
	return file
}

func checkBinFile(appMainFile string, suffix string, _suffix *string, appBuildDir string) error {
	_, err := os.Stat(appMainFile)
	if err != nil {
		if len(c.Conf.App.BuildDir) == 0 {
			return errors.New(err.Error() + `: ` + appMainFile)
		}
		appMainFile = filepath.Join(c.Conf.App.BuildDir, appMainFile)
		_, err = os.Stat(appMainFile)
		if err != nil {
			return errors.New(err.Error() + `: ` + appMainFile)
		}
	}
	appMainFile, err = filepath.Abs(appMainFile)
	if err != nil {
		return errors.New(err.Error() + `: ` + appMainFile)
	}
	if len(c.Conf.App.BuildDir) == 0 {
		c.Conf.App.BuildDir = filepath.Dir(appMainFile)
	}
	fileName := filepath.Base(appMainFile)
	AppBin = fileName
	if strings.HasSuffix(AppBin, suffix) {
		AppBin = strings.TrimSuffix(AppBin, suffix)
		*_suffix = suffix
	}
	nameOk := strings.HasPrefix(AppBin, BinPrefix)
	if nameOk {
		fileName := strings.TrimPrefix(AppBin, BinPrefix)
		_, err := strconv.ParseInt(fileName, 10, 64)
		if err != nil {
			nameOk = false
		}
	}
	if !nameOk {
		return fmt.Errorf("exec参数指定的可执行文件名称格式应该为：%v0%v(当前为：%v)。\n其中的“0”是代表版本号的整数，请修改为此格式。", BinPrefix, *_suffix, fileName)
	}
	return nil
}

func startTower() {
	var (
		allowBuild = atob(build)
		suffix     = ".exe"
		_suffix    = ""
	)
	if len(c.Conf.ConfigFile) == 0 {
		c.Conf.ConfigFile = ConfigName
	}
	configFile := c.Conf.ConfigFile
	_, err := confl.DecodeFile(configFile, c.Conf)
	if err != nil {
		if strings.HasSuffix(err.Error(), `. Expected map but found 'string'.`) {
			err = convertOldConfigFormat(configFile)
			if err != nil {
				log.Error(err.Error())
			} else {
				os.Rename(configFile, configFile+`.`+time.Now().Format(`20060102150405`))
				configContent, err := confl.Marshal(c.Conf)
				if err != nil {
					log.Fatal(err)
					return
				}
				_, err = saveFile(configFile, configContent)
				if err != nil {
					log.Error(err)
					return
				}
				log.Info("== Upgrade config file " + ConfigName)
			}
		} else {
			log.Error(err.Error())
		}
	} else {
		if strings.Contains(c.Conf.Watch.IgnoredPath, `\\`) {
			c.Conf.Watch.IgnoredPath = strings.Replace(c.Conf.Watch.IgnoredPath, `\\`, `\`, -1)
		}
		if len(c.Conf.App.BuildDir) == 0 {
			c.Conf.App.BuildDir, _ = os.Getwd()
		}
	}
	if !allowBuild {
		if len(c.Conf.App.ExecFile) == 0 {
			log.Error("请设置exec参数用来指定执行文件位置")
			time.Sleep(time.Second * 10)
			return
		}
	}
	if c.Conf.Verbose {
		c.Conf.LogLevel = `Debug`
	}

	log.DefaultLog.SetLevel(c.Conf.LogLevel)
	if len(c.Conf.Proxy.Port) > 0 {
		err := dialAddress("127.0.0.1:"+c.Conf.Proxy.Port, 1)
		if err == nil {
			log.Error("Error: port (" + c.Conf.Proxy.Port + ") already in used.")
			os.Exit(1)
		}
	}
	if !allowBuild {
		if strings.Contains(c.Conf.App.ExecFile, `*`) {
			orgiMainFile := c.Conf.App.ExecFile
			c.Conf.App.ExecFile = findBinFile(c.Conf.App.ExecFile)
			if len(c.Conf.App.ExecFile) == 0 {
				if len(c.Conf.App.BuildDir) > 0 {
					c.Conf.App.ExecFile = filepath.Join(c.Conf.App.BuildDir, orgiMainFile)
					c.Conf.App.ExecFile = findBinFile(c.Conf.App.ExecFile)
				}
			}
		}
		if err := checkBinFile(c.Conf.App.ExecFile, suffix, &_suffix, c.Conf.App.BuildDir); err != nil {
			fmt.Println(err)
			time.Sleep(time.Second * 300)
			return
		}
		app = NewApp(c.Conf.App.ExecFile, c.Conf.App.Port, c.Conf.App.BuildDir, c.Conf.App.PortParamName)
	} else {
		if len(c.Conf.App.BuildDir) == 0 {
			c.Conf.App.MainFile, _ = filepath.Abs(c.Conf.App.MainFile)
			c.Conf.App.BuildDir = filepath.Dir(c.Conf.App.MainFile)
		}
		if c.Conf.AutoClear {
			err := filepath.Walk(c.Conf.App.BuildDir, func(filePath string, info os.FileInfo, e error) (err error) {
				if e != nil {
					return e
				}
				if info.IsDir() {
					return
				}
				name := info.Name()
				if strings.HasPrefix(name, BinPrefix) {
					err = os.Remove(filePath)
					if err != nil {
						return
					}
				}
				return
			})
			if err != nil {
				log.Error(err)
			}
		}
		app = NewApp(c.Conf.App.MainFile, c.Conf.App.Port, c.Conf.App.BuildDir, c.Conf.App.PortParamName)
	}
	app.OfflineMode = c.Conf.Offline
	app.DisabledLogRequest = c.Conf.LogRequest == false
	app.PkgMirrors = c.Conf.App.PkgMirrors
	app.Env = append(app.Env, c.Conf.App.Env...)
	if len(c.Conf.App.RunParams) > 0 {
		delim := ` `
		param := c.Conf.App.RunParams
		//:<分割符>:<参数>
		if param[0] == ':' {
			param = strings.TrimPrefix(param, `:`)
			if pos := strings.Index(param, `:`); pos > 0 {
				delim = param[0:pos]
				param = param[pos+1:]
			}
		}
		app.RunParams = strings.Split(param, delim)
	}
	if len(c.Conf.App.BuildParams) > 0 {
		delim := ` `
		param := c.Conf.App.BuildParams
		//:<分割符>:<参数>
		if param[0] == ':' {
			param = strings.TrimPrefix(param, `:`)
			if pos := strings.Index(param, `:`); pos > 0 {
				delim = param[0:pos]
				param = param[pos+1:]
			}
		}
		app.BuildParams = strings.Split(param, delim)
	}
	watchedDir := app.Root
	if !allowBuild {
		if len(app.BuildDir) > 0 {
			watchedDir = app.BuildDir
		}
	}
	if len(c.Conf.Watch.OtherDir) > 0 {
		watchedDir = c.Conf.Watch.OtherDir + "|" + watchedDir
	}
	watcher := NewWatcher(watchedDir, c.Conf.Watch.FileExtension, c.Conf.Watch.IgnoredPath)
	proxy := NewProxy(&app, &watcher)
	proxy.AdminPwd = c.Conf.Admin.Password
	proxy.Engine = c.Conf.Proxy.Engine
	if len(c.Conf.Admin.IPs) > 0 {
		proxy.AdminIPs = strings.Split(c.Conf.Admin.IPs, `,`)
	}
	if allowBuild {
		watcher.OnChanged = func(file string) {
			watcher.Reset()
			fileName := filepath.Base(file)
			if strings.HasPrefix(fileName, BinPrefix) {
				log.Info(`忽略`, fileName, `更改`)
				return
			}
			port, err := getPort()
			if err != nil {
				log.Error(err)
				return
			}
			err = app.Start(true, port)
			if err != nil {
				log.Error(err)
			}
		}
	} else {
		watcher.OnChanged = func(file string) {
			watcher.Reset()
			port, err := getPort()
			if err != nil {
				log.Error(err)
				return
			}
			log.Debug(`== Switch port to `, port)
			fileName := filepath.Base(file)
			if !strings.HasPrefix(fileName, BinPrefix) {
				log.Info(`忽略非`, BinPrefix, `前缀文件更改`)
				return
			}
			if len(_suffix) > 0 {
				fileName = strings.TrimSuffix(fileName, _suffix)
			}
			newAppBin := fileName
			fileName = strings.TrimPrefix(fileName, BinPrefix)
			newFileTs, err := strconv.ParseInt(fileName, 10, 64)
			if err != nil {
				log.Error(err)
				return
			}
			fileName = strings.TrimPrefix(AppBin, BinPrefix)
			oldFileTs, err := strconv.ParseInt(fileName, 10, 64)
			if err != nil {
				log.Error(err)
				return
			}
			if newFileTs <= oldFileTs {
				log.Info(`新文件时间戳小于旧文件，忽略`)
				return
			}
			AppBin = newAppBin
			err = app.Start(true, port)
			if err != nil {
				log.Error(err)
			}
		}
		watcher.OnlyWatchBin = true
		app.DisabledBuild = true
	}
	proxy.Port = c.Conf.Proxy.Port
	go func() {
		mustSuccess(watcher.Watch())
	}()
	err = app.Start(true, app.Port)
	if err != nil {
		log.Error(err)
	}
	mustSuccess(proxy.Listen())
}

func getPort() (port string, err error) {
	port = app.Port
	if !app.DisabledVisitPort() {
		if !app.SupportMutiPort() {
			err = errors.New(`Unspecified switchable other ports.`)
			return
		}
		port = app.UseRandPort()
		for i := 0; i < 3 && port == app.Port; i++ {
			app.Clean()
			time.Sleep(time.Second)
			port = app.UseRandPort()
		}
		if port == app.Port {
			err = errors.New(`取得的端口与当前端口相同，无法编译切换`)
		}
	}
	return
}
