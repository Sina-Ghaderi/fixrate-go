package config

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"syslog"
)

const literalRegex string = `^(\s*)((?i)(socket_path|sql_database|sql_username|sql_address|sql_tcpport|sql_password|default_ratelimit|socket_perm|listener_type|listen_addr)(?i))(\s*)=`

type Config struct{}

var mapconf = make(map[string][]string)

func ReadConfigFromFile(path string) *Config {
	file, err := os.Open(path)
	if err != nil {
		panic(syslog.BigError{Why: err, Cod: 1})
	}
	defer file.Close()
	re := regexp.MustCompile(literalRegex)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if find := re.MatchString(scanner.Text()); find {
			literal := re.FindString(scanner.Text())
			mapconf[strings.TrimSpace(literal[:len(literal)-1])] = strings.Fields(strings.Split(scanner.Text(), literal)[1])
		}
	}
	if err := scanner.Err(); err != nil {
		panic(syslog.BigError{Why: err, Cod: 1})
	}
	return &Config{}
}

func (*Config) GetConf(str string, index int) string {
	if args, owkey := mapconf[str]; owkey {
		if len(args) == 0 {
			panic(syslog.BigError{Why: fmt.Errorf("literal %v have no args in config file", str), Cod: 1})
		}
		if index < len(args) {
			return args[index]
		}

		panic(syslog.BigError{Why: fmt.Errorf("not enough args in literal %v, can not reach arg %v, it's empty", str, index), Cod: 1})
	}
	panic(syslog.BigError{Why: fmt.Errorf("literal %v can not be found in config file", str), Cod: 1})
}
