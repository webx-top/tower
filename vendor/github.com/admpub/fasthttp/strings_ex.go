package fasthttp

func init() {
	defaultServerName = "webx"
	defaultUserAgent = "webx"
	//defaultContentType = []byte("text/html; charset=utf-8")
}

func SetDefaultServerName(name string) {
	defaultServerName = name
}

func SetDefaultUserAgent(userAgent string) {
	defaultUserAgent = userAgent
}

func SetDefaultContentType(contentType []byte) {
	defaultContentType = contentType
}
