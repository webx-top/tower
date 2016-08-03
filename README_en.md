# Tower

Tower makes your Go web development much more dynamic by monitoring file's changes in your project and then re-run your
app to apply those changes – yeah, no more stopping and running manually! It will also show compiler error, panic and
runtime error through a clean page (see the demo below).

## Install
```bash
go get github.com/admpub/tower
```

## Usage

```bash
cd your/project
tower # now visit localhost:8080
```

Tower will, by default, assume your web app's port is _5001-5050_. These can be changed by:

```bash
tower -p 3000-4000
```

Or put them in a config file:

```bash
tower init
vim tower.yml
tower
```

## Troubleshooting

#### 'Too many open files'

Run the following command to increase the number of files that a process can open:

```bash
ulimit -S -n 2048 # OSX
```

## How it works?

```
browser: http://localhost:8080
      \/
tower (listening 8080)
      \/ (reverse proxy)
your web app (listening port range 5001-5050)
```

Any request comes from localhost:8080 will be handled by Tower and then be redirected to your app. The redirection is
done by using _[httputil.ReverseProxy](http://golang.org/pkg/net/http/httputil/#ReverseProxy)_. Before redirecting the request, Tower will compile and run your app in
another process if your app hasn't been run or file has been changed; Tower is using
_[howeyc/fsnotify](https://github.com/howeyc/fsnotify)_ to monitor file changes.

## License

Tower is released under the [MIT License](http://www.opensource.org/licenses/MIT).
