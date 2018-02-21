package Misc

import "log"

func ErrorFatal(msg string, e error) {
	if e != nil {
		log.Fatal("%s, err = %s\n", msg, e.Error())
	}
}
