package datax

import (
	"config"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"syslog"
	"time"
)

type UserAccount struct {
	Username              string
	Limit, Counter, Reset int
	LastReset             time.Time
	UserType              bool
}

var (
	DefaultUserTmp UserAccount
	SystemDatabase *sql.DB
)

func SQLInit(file *config.Config) error {
	dabase := file.GetConf("sql_database", 0)
	addr := fmt.Sprintf("(%v:%v)", file.GetConf("sql_address", 0), file.GetConf("sql_tcpport", 0))
	db, err := sql.Open("mysql", file.GetConf("sql_username", 0)+":"+file.GetConf("sql_password", 0)+"@tcp"+addr+"/"+dabase+"?parseTime=true")
	if err != nil {
		return err
	}

	limit, err := strconv.Atoi(file.GetConf("default_ratelimit", 0))
	if err != nil {
		panic(syslog.BigError{Why: errors.New("can not use non integer type (default_ratelimit)"), Cod: 1})
	}
	reset, err := strconv.Atoi(file.GetConf("default_ratelimit", 1))
	if err != nil {
		panic(syslog.BigError{Why: errors.New("can not use non integer type (default_ratelimit)"), Cod: 1})
	}
	DefaultUserTmp = UserAccount{
		Limit: limit,
		Reset: reset,
	}

	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	sqlTable := `
		CREATE TABLE IF NOT EXISTS fixrate(
			username VARCHAR(512) NOT NULL PRIMARY KEY,
			limitt INT,
			counter INT,
			reset INT,
			lastreset DATETIME,
			usertype BOOL
		);
		`
	if _, err := db.ExecContext(ctx, sqlTable); err != nil {
		return err
	}
	SystemDatabase = db
	if err := updateDefaultFixRates(); err != nil {
		return err
	}

	return nil
}

func CreateNewUser(item *UserAccount) error {
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	sqlAdditem := `
	REPLACE INTO fixrate(
		username,
		limitt,
		counter,
		reset,
		lastreset,
		usertype
	) values(?, ?, ?, ?, ?, ?)
	`
	stmt, err := SystemDatabase.Prepare(sqlAdditem)
	defer stmt.Close()

	if err != nil {
		return err
	}
	_, err = stmt.ExecContext(ctx, item.Username, item.Limit, item.Counter, item.Reset, item.LastReset, item.UserType)
	if err != nil {
		return err
	}
	return nil
}

func DBClose() {
	if err := SystemDatabase.Close(); err != nil {
		panic(syslog.BigError{Why: err, Cod: 1})
	}
}

func GetUserFromDatabase(user *string) (*UserAccount, error) {
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()

	sqlReadall := `SELECT * FROM fixrate WHERE username = ?`
	rows := SystemDatabase.QueryRowContext(ctx, sqlReadall, user)
	users := UserAccount{}
	if err := rows.Scan(&users.Username, &users.Limit, &users.Counter, &users.Reset, &users.LastReset, &users.UserType); err != nil {
		if err != sql.ErrNoRows {
			return nil, err
		}
		newuser := DefaultUserTmp
		newuser.Username = *user
		newuser.LastReset = time.Now()
		if err := CreateNewUser(&newuser); err != nil {
			return nil, err
		}
		return &newuser, nil

	}
	return &users, nil
}

func (u *UserAccount) UpdateUserCounter(counter int) error {
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := SystemDatabase.Prepare("update fixrate set counter=? where username=?")
	if err != nil {
		return err
	}
	_, err = stmt.ExecContext(ctx, counter, u.Username)
	if err != nil {
		return err
	}
	return nil
}

func (u *UserAccount) UpdateUserLastReset(nowtime time.Time) error {
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := SystemDatabase.Prepare("update fixrate set lastreset=? where username=?")
	if err != nil {
		return err
	}
	_, err = stmt.ExecContext(ctx, nowtime, u.Username)
	if err != nil {
		return err
	}
	return nil
}

func updateDefaultFixRates() error {
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := SystemDatabase.Prepare("update fixrate set limitt=?, reset=? where usertype=?")
	if err != nil {
		return err
	}
	_, err = stmt.ExecContext(ctx, DefaultUserTmp.Limit, DefaultUserTmp.Reset, DefaultUserTmp.UserType)
	if err != nil {
		return err
	}
	return nil
}
