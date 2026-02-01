package jesva

import "log"

var _debug = false

func Debug(format string, args ...any) {
	if _debug {
		log.Printf(format, args...)
	}
}

func EnableDebug() {
	_debug = true
}
