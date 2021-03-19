# fixrate-go
last night we figured out someone sending too many emails to our mailing servers, so this is the tool for handling those email flooding attacks.  
fixrate applies sending rate limit per users, so you don't have to limit entire domain. from now on fixrate protects [mail.snix.ir](https://mail.snix.ir) service.  

### usege and installation
fixrate is written in golang, so for compiling this code you should have `golang` installed  
if not, use apt to installed `apt install golang`, after that clone this repository by useing either [git.snix.ir](https://git.snix.ir/fixrate-go) or github.com  
```
# git clone https://git.snix.ir/fixrate-go.git
# git clone https://github.com/Sina-Ghaderi/fixrate-go.git
# cd fixrate-go && go build

# ./fixrate-go
expected 'daemon' or 'users' commands
usage of fixrate postfix module snix.ir LLC:
./fixrate commands... [ OPTIONS ] ...

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
./fixrate users --username name@domain.com -- userrate 100 --counter 600

Copyright (c) 2021 git.snix.ir, All rights reserved.
Developed BY sina@snix.ir --> Sina Ghaderi  
This work is licensed under the terms of the MIT license.
```
you may wnat to create a systemd file for runing this as a linux service (`systemctl start` ... and `enable` etc...)

### config postfix and fixrate 
fixrate-go supports unix socket and tcp network listeners, in order to reduce kernel opened connections its recommended to use unix socket if you can host fixrate on postfix server  
default configuration file of fixrate is ./fixrate.go, you may want to specify another file path by using `--config` flag  

### preparing postfix server
edit postfix mail agent config file under `/etc/postfix/main.cf` (debian based systems)
if planning to use unix file you should note that for some reasons postfix root directory is under /var/spool/postfix so you should create fixrate-go socket file under this directory, otherwise you may facing with `warning: connect to <socket> no such a file or directory` warning in` mail.log` file 
```
## if using unix socket -->
# echo "smtpd_sender_restrictions = reject_unknown_sender_domain, check_policy_service unix:/fix/fixrate-go.socks" >> /etc/postfix/main.cf
# mkdir /var/spool/postfix/fix
# service postfix restart 

## if useing tcp connection --> 
# echo "smtpd_sender_restrictions = reject_unknown_sender_domain, check_policy_service inet:127.0.0.1:9984" >> /etc/postfix/main.cf
# service postfix restart
```

### config fixrate-go service
just use default config file in this package `fixrate.conf` and change it as you need.
