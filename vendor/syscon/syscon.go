package syscon

import (
	"bufio"
	"bytes"
	"config"
	"datax"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syslog"
	"time"
)

func StartNewService(conf *config.Config) {
	if err := datax.SQLInit(conf); err != nil {
		panic(syslog.BigError{Why: err, Cod: 1})
	}
	defer datax.DBClose()
	lisfunc, isthere := networkListenType[conf.GetConf("listener_type", 0)]
	if !isthere {
		panic(syslog.BigError{Why: errors.New("invalid listener_type in config file, should be either inet or unix"), Cod: 1})
	}
	l := lisfunc(conf)
	defer l.Close()
	syslog.InformGreen("connection to sql server established")
	syn := regexp.MustCompile(`(^sasl_username=((.*[^\s].*)|))$`)

	for {
		conn, err := l.Accept()
		if err != nil {
			syslog.InformError(err)
			continue
		}

		go handlePostfixConn(conn, syn)
	}

}

func handlePostfixConn(conn net.Conn, syn *regexp.Regexp) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		if find := syn.MatchString(scanner.Text()); find {
			pureName := strings.ReplaceAll(scanner.Text(), "sasl_username=", "")
			if pureName == "" {
				if _, err := conn.Write([]byte("action=OK\n\n")); err != nil {
					syslog.InformError(err)
					return
				}
				continue
			}
			userAcc, err := datax.GetUserFromDatabase(&pureName)
			if err != nil {
				syslog.InformError(err)
				_, err := conn.Write([]byte("action=REJECT fixrate error: internal  server error <1>, contact administrator\n\n"))
				if err != nil {
					syslog.InformError(err)
					return
				}
				continue
			}

			if userAcc.LastReset.Add(time.Duration(userAcc.Reset) * time.Second).Before(time.Now()) {
				if err := userAcc.UpdateUserLastReset(time.Now()); err != nil {
					syslog.InformError(err)
					_, err := conn.Write([]byte("action=REJECT fixrate error: internal server error <2>, contact administrator\n\n"))
					if err != nil {
						syslog.InformError(err)
						return
					}
					continue
				}

				if err := userAcc.UpdateUserCounter(0); err != nil {
					syslog.InformError(err)
					_, err := conn.Write([]byte("action=REJECT fixrate error: internal  server error <3>, contact administrator\n\n"))
					if err != nil {
						syslog.InformError(err)
						return
					}
					continue
				}
				userAcc.Counter = 0
			}
			if userAcc.Limit <= userAcc.Counter {
				diff := userAcc.LastReset.Add(time.Duration(userAcc.Reset) * time.Second).Sub(time.Now())
				_, err := conn.Write([]byte(fmt.Sprintf("action=REJECT fixrate: sending limit exceeded, you can't send anything until next %v\n\n", diff.Round(time.Second))))
				if err != nil {
					syslog.InformError(err)
					return
				}
				continue
			}
			if err := userAcc.UpdateUserCounter(userAcc.Counter + 1); err != nil {
				syslog.InformError(err)
				_, err := conn.Write([]byte("action=REJECT fixrate error: internal  server error <4>, contact administrator\n\n"))
				if err != nil {
					syslog.InformError(err)
					return
				}
				continue
			}
			if _, err := conn.Write([]byte("action=OK\n\n")); err != nil {
				syslog.InformError(err)
				return
			}
		}
	}
	if err := scanner.Err(); err != nil {
		panic(syslog.BigError{Why: err, Cod: 1})
	}
}

func checkUinixSocket(socketAddr string) (string, bool) {
	unix, err := os.Open("/proc/net/unix")
	if err != nil {
		panic(syslog.BigError{Why: err, Cod: 1})
	}
	defer unix.Close()
	u := bufio.NewScanner(unix)
	for u.Scan() {
		if bytes.Contains(u.Bytes(), []byte(socketAddr)) {
			pid := findPid(strings.Fields(u.Text())[6])
			return pid, true
		}
	}
	if err := u.Err(); err != nil {
		panic(syslog.BigError{Why: err, Cod: 1})
	}
	return "", false
}

var networkListenType = map[string]func(conf *config.Config) net.Listener{
	"unix": func(conf *config.Config) net.Listener {
		socketAddr := conf.GetConf("socket_path", 0)
		if pid, exists := checkUinixSocket(socketAddr); exists {
			panic(syslog.BigError{Why: fmt.Errorf("unix socket %v is held by another process (pid: %v), is another fixrate daemon runing?", socketAddr, pid), Cod: 1})
		}
		if err := os.RemoveAll(socketAddr); err != nil {
			panic(syslog.BigError{Why: err, Cod: 1})
		}
		SockPerm, err := strconv.Atoi(conf.GetConf("socket_perm", 0))
		if err != nil {
			panic(syslog.BigError{Why: errors.New("socket_path should hold an integer"), Cod: 1})
		}
		l, err := net.Listen("unix", socketAddr)
		if err != nil {
			panic(syslog.BigError{Why: err, Cod: 1})
		}

		mod := os.FileMode(SockPerm)
		if err = os.Chmod(socketAddr, mod); err != nil {
			l.Close()
			panic(syslog.BigError{Why: err, Cod: 1})
		}
		syslog.InformGreen("start listening on unix socket", socketAddr)
		return l
	},
	"inet": func(conf *config.Config) net.Listener {
		listenAddr := conf.GetConf("listen_addr", 0)
		l, err := net.Listen("tcp", listenAddr)
		if err != nil {
			panic(syslog.BigError{Why: err, Cod: 1})
		}
		syslog.InformGreen("start listening on tcp address", listenAddr)
		return l
	},
}

func findPid(inode string) string {
	type sysInodes struct {
		path string
		link string
	}
	var pid string
	fd, err := filepath.Glob("/proc/[0-9]*/fd/[0-9]*")
	if err != nil {
		return pid
	}

	inodes := make([]sysInodes, len(fd))
	mx := make(chan sysInodes, len(fd))

	go func(fd *[]string, outchan chan<- sysInodes) {
		for _, item := range *fd {
			link, _ := os.Readlink(item)
			outchan <- sysInodes{item, link}
		}
	}(&fd, mx)

	for range fd {
		inodes = append(inodes, <-mx)
	}

	re := regexp.MustCompile(inode)
	for _, item := range inodes {
		out := re.FindString(item.link)
		if len(out) != 0 {
			pid = strings.Split(item.path, "/")[2]
		}
	}
	return pid
}
