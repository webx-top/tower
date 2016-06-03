package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	HttpPanicMessage = "http: panic serving"
)

var (
	BinPrefix = "tower-app-"
	AppBin    = BinPrefix + strconv.FormatInt(time.Now().Unix(), 10)
)

type App struct {
	OfflineMode     bool
	Cmds            map[string]*exec.Cmd
	RunParams       []string
	MainFile        string
	Port            string
	Ports           map[string]int64
	BuildDir        string
	Name            string
	Root            string
	KeyPress        bool
	LastError       string
	PortParamName   string //端口参数名称(用于指定应用程序监听的端口，例如：webx.exe -p 8080，这里的-p就是端口参数名)
	SwitchToNewPort bool
	DisabledBuild   bool

	BuildStart   *sync.Once
	startErr     error
	AppRestart   *sync.Once
	restartErr   error
	portBinFiles map[string]string
}

type StderrCapturer struct {
	app *App
}

func (this StderrCapturer) Write(p []byte) (n int, err error) {
	httpError := strings.Contains(string(p), HttpPanicMessage)

	if httpError {
		this.app.LastError = string(p)
		os.Stdout.Write([]byte("----------- Application Error -----------\n"))
		n, err = os.Stdout.Write(p)
		os.Stdout.Write([]byte("-----------------------------------------\n"))
	} else {
		n, err = os.Stdout.Write(p)
	}
	return
}

func NewApp(mainFile, port, buildDir, portParamName string) (app App) {
	app.Cmds = make(map[string]*exec.Cmd)
	app.MainFile = mainFile
	app.BuildDir = buildDir
	app.PortParamName = portParamName
	app.ParseMutiPort(port)
	app.Port = app.UseRandPort()
	wd, _ := os.Getwd()
	app.Name = filepath.Base(wd)
	app.Root = filepath.Dir(mainFile)
	app.BuildStart = &sync.Once{}
	app.AppRestart = &sync.Once{}
	app.portBinFiles = make(map[string]string)
	app.RunParams = []string{}
	return
}

func (this *App) ParseMutiPort(port string) {
	p := strings.Split(port, `,`)
	this.Ports = make(map[string]int64)
	for _, v := range p {
		r := strings.Split(v, `-`)
		if len(r) > 1 {
			i, _ := strconv.Atoi(r[0])
			j, _ := strconv.Atoi(r[1])
			for ; i <= j; i++ {
				port := fmt.Sprintf("%v", i)
				this.Ports[port] = 0
			}
		} else {
			this.Ports[r[0]] = 0
		}
	}
}

func (this *App) SupportMutiPort() bool {
	return this.Ports != nil && len(this.Ports) > 1 && this.PortParamName != ``
}

func (this *App) UseRandPort() string {
	lastRunTime := make([]int64, 0)
	lastRunPorts := make(map[int64]string, 0)
	for port, runningTime := range this.Ports {
		if runningTime == 0 || this.IsRunning(port) == false || isFreePort(port) {
			return port
		}
		lastRunTime = append(lastRunTime, runningTime)
		lastRunPorts[runningTime] = port
	}
	quickSort(lastRunTime, 0, len(lastRunTime)-1)
	for _, runningTime := range lastRunTime {
		return lastRunPorts[runningTime]
	}
	return this.Port
}

func (this *App) Start(build bool, args ...string) error {
	this.BuildStart.Do(func() {
		if build {
			this.startErr = this.Build()
			if this.startErr != nil {
				fmt.Println("== Fail to build " + this.Name)
				this.BuildStart = &sync.Once{}
				return
			}
		}
		port := this.Port
		if len(args) > 0 {
			port = args[0]
		}
		this.startErr = this.Run(port)
		if this.startErr != nil {
			this.startErr = errors.New("Fail to run " + this.Name)
			this.BuildStart = &sync.Once{}
			return
		}
		this.RestartOnReturn()
		this.BuildStart = &sync.Once{}
	})

	return this.startErr
}

func (this *App) Restart() error {
	this.AppRestart.Do(func() {
		this.Clean()
		this.Stop(this.Port)
		this.restartErr = this.Start(true)
		this.AppRestart = &sync.Once{} // Assign new Once to allow calling Start again.
	})

	return this.restartErr
}

func (this *App) BinFile(args ...string) (f string) {
	binFileName := AppBin
	if len(args) > 0 {
		binFileName = args[0]
	}
	if app.BuildDir != "" {
		f = filepath.Join(app.BuildDir, binFileName)
	} else {
		f = binFileName
	}
	if runtime.GOOS == "windows" {
		f += ".exe"
	}
	return
}

func (this *App) Stop(port string, args ...string) {
	if !this.IsRunning(port) {
		return
	}
	fmt.Println("== Stopping " + this.Name)
	cmd := this.GetCmd(port)
	err := cmd.Process.Kill()
	if err != nil {
		fmt.Println(err)
	}
	cmd = nil
	if port == this.Port && this.DisabledBuild {
		return
	}
	bin := this.BinFile(args...)
	err = os.Remove(bin)
	if err == nil {
		this.Ports[port] = 0
		return
	}
	go func() {
		for i := 0; i < 10; i++ {
			time.Sleep(time.Second)
			err = os.Remove(bin)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(`Remove ` + bin + `: Success.`)
				this.Ports[port] = 0
				return
			}
		}
	}()
}

func (this *App) Clean() {
	for port, cmd := range this.Cmds {
		if port == this.Port || !CmdIsRunning(cmd) {
			continue
		}
		fmt.Println("== Stopping app at port: " + port)
		err := cmd.Process.Kill()
		if err != nil {
			fmt.Println(err)
		}
		cmd = nil
		if bin, ok := this.portBinFiles[port]; ok && bin != "" {
			err := os.Remove(bin)
			if err == nil {
				this.Ports[port] = 0
				continue
			}
			go func() {
				for i := 0; i < 10; i++ {
					time.Sleep(time.Second)
					err = os.Remove(bin)
					if err != nil {
						fmt.Println(err)
					} else {
						fmt.Println(`Remove ` + bin + `: Success.`)
						this.Ports[port] = 0
						return
					}
				}
			}()
		}
	}
}

func (this *App) GetCmd(args ...string) (cmd *exec.Cmd) {
	var port string
	if len(args) > 0 {
		port = args[0]
	} else {
		port = this.Port
	}
	cmd, _ = this.Cmds[port]
	return
}

func (this *App) SetCmd(port string, cmd *exec.Cmd) {
	this.Cmds[port] = cmd
}

func (this *App) Run(port string) (err error) {
	bin := this.BinFile()
	_, err = os.Stat(bin)
	if err != nil {
		return
	}
	fmt.Println("== Running at port " + port + ": " + this.Name)
	ableSwitch := this.Port != port
	this.Port = port //记录被使用的端口，避免下次使用

	var cmd *exec.Cmd
	this.portBinFiles[port] = bin
	this.Ports[port] = time.Now().Unix()
	params := []string{}
	if this.SupportMutiPort() {
		params = append(params, this.PortParamName)
		params = append(params, port)
	}
	params = append(params, app.RunParams...)
	cmd = exec.Command(bin, params...)
	this.SetCmd(this.Port, cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = StderrCapturer{this}
	var hasError bool
	go func() {
		err := cmd.Run()
		if err != nil {
			fmt.Println(`== cmd.Run Error:`, err)
			hasError = true
		}
	}()
	err = dialAddress("127.0.0.1:"+this.Port, 60, func() bool {
		return !hasError
	})
	if err == nil && ableSwitch {
		this.SwitchToNewPort = true
		if this.OfflineMode {
			this.Clean()
		}
	}
	return
}

func (this *App) Build() (err error) {
	if this.DisabledBuild {
		return nil
	}
	fmt.Println("== Building " + this.Name)
	AppBin = BinPrefix + strconv.FormatInt(time.Now().Unix(), 10)
	out, _ := exec.Command("go", "build", "-o", this.BinFile(), this.MainFile).CombinedOutput()
	if len(out) > 0 {
		msg := strings.Replace(string(out), "# command-line-arguments\n", "", 1)
		fmt.Printf("----------- Build Error -----------\n%s-----------------------------------\n", msg)
		return errors.New(msg)
	}
	fmt.Println("== Build completed.")
	return nil
}

func (this *App) IsRunning(args ...string) bool {
	return CmdIsRunning(this.GetCmd(args...))
}

func CmdIsRunning(cmd *exec.Cmd) bool {
	return cmd != nil && cmd.ProcessState == nil
}

func CmdIsQuit(cmd *exec.Cmd) bool {
	return cmd != nil && cmd.ProcessState != nil
}

func (this *App) IsQuit(args ...string) bool {
	return CmdIsQuit(this.GetCmd(args...))
}

func (this *App) RestartOnReturn() {
	if this.KeyPress {
		return
	}
	this.KeyPress = true

	// Listen to keypress of "return" and restart the app automatically
	go func() {
		in := bufio.NewReader(os.Stdin)
		for {
			input, _ := in.ReadString('\n')
			if input == "\n" {
				this.Restart()
			}
		}
	}()

	// Listen to "^C" signal and stop the app properly
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		<-sig // wait for the "^C" signal
		fmt.Println("")
		this.Stop(this.Port)
		os.Exit(0)
	}()
}
