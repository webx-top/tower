package config

var Conf = &Config{
	App:   App{},
	Proxy: Proxy{},
	Admin: Admin{},
	Watch: Watch{},
}

type App struct {
	ExecFile      string            `json:"exec"` //非编译模式下有效
	MainFile      string            `json:"main"` //编译模式下有效
	Port          string            `json:"port"`
	PortParamName string            `json:"portParamName"`
	Generate      bool              `json:"generate"` // 是否在执行 go build 以前执行 go generate
	BuildDir      string            `json:"buildDir"`
	BuildParams   string            `json:"buildParams"`
	RunParams     string            `json:"params"`
	PkgMirrors    map[string]string `json:"pkgMirrors"`
	Env           []string          `json:"env"`
}

type Proxy struct {
	IP     string `json:"ip"`
	Port   string `json:"port"`
	Engine string `json:"engine"`
}

type Watch struct {
	FileExtension string `json:"fileExtension"`
	OtherDir      string `json:"otherDir"` //编译模式下有效
	IgnoredPath   string `json:"ignoredPath"`
}

type Admin struct {
	Password string `json:"password"`
	IPs      string `json:"ips"`
}

type Config struct {
	App        App    `json:"app"`
	Proxy      Proxy  `json:"proxy"`
	Admin      Admin  `json:"admin"`
	Watch      Watch  `json:"watch"`
	Verbose    bool   `json:"verbose"`
	ConfigFile string `json:"-"`
	LogLevel   string `json:"logLevel"`
	LogRequest bool   `json:"logRequest"`
	AutoClear  bool   `json:"autoClear"`
	Offline    bool   `json:"offline"`
}
