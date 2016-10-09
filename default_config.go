package main

import (
	"github.com/admpub/confl"
	c "github.com/webx-top/tower/config"
)

var defaultConfig = []byte(`
app {
  # 生产环境下的可执行文件。支持用“*”代替文件名的一部分，例如："tower-app-*.exe"
  exec : "tower-app-*.exe"

  # 开发环境下用“go run”命令运行的源文件，一般为“main.go”
  main : ""

  # 你的项目在本机运行的端口列表,可以用半角逗号分隔也可以用减号指定范围，也可以两种结合起来用，例如： "5001,5003,5050-5060"。如果为空，则代表不支持访问端口。
  port : "5001-5050"

  # 指定app端口的参数名，例如：webx.exe -p 8080 其中的“-p”就是。如果为空，则代表不支持访问端口。
  portParamName : "-p"

  # go build -o 命令生成的二进制文件保存位置
  buildDir : ""

  # 运行app所需的其它参数，例如：webx.exe -p 8080 -e 90 -d 100 其中的“-e 90 -d 100”就是(注意：内部用[单个]半角空格隔开)。
  params : ""
}

proxy {
  # 你的项目对外公开访问的端口
  port : "8080"

  # 代理引擎。支持fast和standard
  engine : "fast"
}

admin {
  password : ""
  ips : "127.0.0.1,::1"
}

watch {
  # 要监控更改的文件扩展名。多个扩展名时使用"|"隔开，例如：go|html
  fileExtension : "go"

  # 默认会自动监控上面main参数所指定的文件所在之文件夹，如果你还要监控其它文件夹，请在这里指定。如要指定多个文件夹路径，请用“|”分隔。
  otherDir : ""
  
  # 忽略的路径(正则表达式)，不填则不限制(排除某个完整的文件夹名请用“/文件夹名/”的格式)
  ignoredPath : ""
}

# 是否显示细节信息。如果设置为true，会自动将下面的logLevel设置为Debug
verbose : false

# 日志等级。支持的值有Debug/Info/Warn/Error/Fatal
logLevel : "Debug"

# 是否在控制台显示request日志
logRequest : true

# 是否自动删除以前的可执行文件
autoClear : true

# 是否离线模式(即开发模式)
offline : true

`)

func convertOldConfigFormat(configFile string) error {
	newmap := map[string]string{}
	_, err := confl.DecodeFile(configFile, &newmap)
	if err != nil {
		return err
	}
	if v, ok := newmap["app_port"]; ok {
		c.Conf.App.Port = &v
	}
	if v, ok := newmap["pxy_port"]; ok {
		c.Conf.Proxy.Port = &v
	}
	if v, ok := newmap["pxy_engine"]; ok {
		c.Conf.Proxy.Engine = &v
	}
	if v, ok := newmap["auto_clear"]; ok {
		b := atob(v)
		c.Conf.AutoClear = &b
	}
	if v, ok := newmap["log_level"]; ok {
		c.Conf.LogLevel = &v
	}
	if v, ok := newmap["app_buildDir"]; ok {
		c.Conf.App.BuildDir = &v
	}
	if v, ok := newmap["app_portParamName"]; ok {
		c.Conf.App.PortParamName = &v
	}
	if v, ok := newmap["app_runParams"]; ok {
		c.Conf.App.RunParams = &v
	}
	if v, ok := newmap["watch"]; ok {
		c.Conf.Watch.FileExtension = &v
	}
	if v, ok := newmap["watch_otherDir"]; ok {
		c.Conf.Watch.OtherDir = &v
	} //编译模式下有效
	if v, ok := newmap["watch_ignoredPath"]; ok {
		c.Conf.Watch.IgnoredPath = &v
	}
	if v, ok := newmap["offline_mode"]; ok {
		b := atob(v)
		c.Conf.Offline = &b
	}
	if v, ok := newmap["admin_pwd"]; ok {
		c.Conf.Admin.Password = &v
	}
	if v, ok := newmap["admin_ip"]; ok {
		c.Conf.Admin.IPs = &v
	}
	if v, ok := newmap["log_request"]; ok {
		b := atob(v)
		c.Conf.LogRequest = &b
	}
	if v, ok := newmap["main"]; ok {
		c.Conf.App.MainFile = &v
	} //编译模式下有效
	if v, ok := newmap["exec"]; ok {
		c.Conf.App.ExecFile = &v
	} //非编译模式下有效
	return nil
}
