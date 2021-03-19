package syslog

import (
	"fmt"
	"log"
	"os"
)

type BigError struct {
	Why error
	Cod int
	Pid int
}

func HandlePan() {
	if hap := recover(); hap != nil {
		if ms, owkey := hap.(BigError); owkey {
			ms.Pid = os.Getpid()
			fmt.Println("\033[31mfatal:\033[0m", ms.Why, "\nprocess", ms.Pid, "exit with ststus", ms.Cod)
			os.Exit(ms.Cod)
		}
		panic(hap)
	}
}

func InformYellow(h ...interface{}) {
	h = append([]interface{}{("\033[33minfo:\033[0m")}, h...)
	log.Println(h...)
}

func InformGreen(h ...interface{}) {
	h = append([]interface{}{("\033[32minfo:\033[0m")}, h...)
	log.Println(h...)
}
func InformError(h ...interface{}) {
	h = append([]interface{}{("\033[31merror:\033[0m")}, h...)
	log.Println(h...)
}
