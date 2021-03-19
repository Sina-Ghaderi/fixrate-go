package main

import (
	"config"
	"datax"
	"flag"
	"fmt"
	"os"
	"runtime"
	"syscon"
	"syslog"
	"time"
)

func main() {
	if runtime.GOOS != "linux" {
		fmt.Println("this application is supposed to run on linux, there is nothing to do in here... \nexiting with status 0")
		return
	}
	defer syslog.HandlePan()

	fixDaemon := flag.NewFlagSet("daemon", flag.ExitOnError)
	confPath := fixDaemon.String("config", "fixrate.conf", "gnu config file for fixrate service")
	fixDaemon.Usage = fixRateUsage
	fixUser := flag.NewFlagSet("users", flag.ExitOnError)
	fixUser.Usage = addUserUsage
	confPathforUser := fixUser.String("config", "fixrate.conf", "gnu config file for fixrate service")
	userName := fixUser.String("username", "sina@snix.ir", "a username to add/modify in database")
	userRate := fixUser.Int("userrate", 10, "how many e-mails user should be able to send")
	userReset := fixUser.Int("counter", 120, "time interval between user counter reset")
	flag.Usage = globUsage
	if len(os.Args) < 2 {
		fmt.Println("expected 'daemon' or 'users' commands")
		flag.Usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "daemon":
		fixDaemon.Parse(os.Args[2:])
		conf := config.ReadConfigFromFile(*confPath)
		syscon.StartNewService(conf)
	case "users":
		fixUser.Parse(os.Args[2:])
		if len(os.Args[2:]) == 0 {
			fmt.Println("expect more options -- use --h for information")
			fixUser.Usage()
			os.Exit(1)
		}
		conf := config.ReadConfigFromFile(*confPathforUser)
		if err := datax.SQLInit(conf); err != nil {
			panic(syslog.BigError{Why: err, Cod: 1})
		}
		defer datax.DBClose()
		temp := &datax.UserAccount{
			Username:  *userName,
			UserType:  true,
			LastReset: time.Now(),
			Limit:     *userRate,
			Reset:     *userReset,
		}
		if err := datax.CreateNewUser(temp); err != nil {
			panic(syslog.BigError{Why: err, Cod: 1})
		}
		fmt.Printf("\033[32minfo:\033[0m username: %v with %v e-mails per %v seconds added to database.\n", *userName, *userRate, *userReset)
	default:
		fmt.Println("expected 'daemon' or 'users' commands")
		flag.Usage()
		os.Exit(1)
	}
}
func globUsage() {
	fmt.Printf(`usage of fixrate postfix module snix.ir LLC:
%v commands... [ OPTIONS ] ...

commands:
  daemon         starting fixrate daemon, should be used by systemd
    --config     pass a file to read configuration from. default: ./fixrate.conf

  users          add or modify users and attributes in database
    --config     pass a file to read configuration from. default: ./fixrate.conf
    --username   a username to add/modify in database. default is sina@snix.ir
    --counter    time interval (seconds) between user counter reset. default is 120
    --userrate   how many e-mails user should be able to send. default is 10

example: 
---- adding name@domain.com ---- 100 e-mail per 10 minutes:
%v users --username name@domain.com -- userrate 100 --counter 600

Copyright (c) 2021 git.snix.ir, All rights reserved.
Developed BY sina@snix.ir --> Sina Ghaderi  
This work is licensed under the terms of the MIT license.
`, os.Args[0], os.Args[0])
}

func fixRateUsage() {
	fmt.Printf(`usage of fixrate postfix module snix.ir LLC:
%v daemon [ OPTIONS ] ...

options:
    --config    pass a file to read configuration from. default: ./fixrate.conf
    --h         print this banner and exit.

Copyright (c) 2021 git.snix.ir, All rights reserved.
Developed BY sina@snix.ir --> Sina Ghaderi  
This work is licensed under the terms of the MIT license.
`, os.Args[0])
}

func addUserUsage() {
	fmt.Printf(`usage of fixrate postfix module snix.ir LLC:
%v users [ OPTIONS ] ...

options:
    --config      pass a file to read configuration from. default: ./fixrate.conf
    --username    a username to add/modify in database. default is sina@snix.ir
    --counter     time interval (seconds) between user counter reset. default is 120
    --userrate    how many e-mails user should be able to send. default is 10

example: 
---- adding name@domain.com ---- 100 e-mail per 10 minutes:
%v users --username name@domain.com -- userrate 100 --counter 600
	
Copyright (c) 2021 git.snix.ir, All rights reserved.
Developed BY sina@snix.ir --> Sina Ghaderi  
This work is licensed under the terms of the MIT license.
`, os.Args[0], os.Args[0])
}
