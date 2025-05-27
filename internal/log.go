package internal

import "log"

func LogGoroutineClosed(name string) {
	log.Printf("Goroutine closed: \"%s\"", name)
}
