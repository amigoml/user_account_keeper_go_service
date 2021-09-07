package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"strconv"
	"time"
	// 	"fmt"
)

type server struct {
	db *sql.DB
}

type user struct {
	UserId int
	Amount int
}

type accountKeeper struct {
	Users []user
}

type historyEntry struct {
	UserId  int
	IsDebit bool
	Amount    int
	TransTime time.Time
}

type history struct {
	Histories []historyEntry
}

func returnErr(s string, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(`{"Response": "`+ s + `"}`))
}

func (s *server) getBalance(w http.ResponseWriter, r *http.Request) {
	queryString := r.URL.Query()
	userId, err := strconv.Atoi(queryString.Get("user_id"))
	if err != nil {
		returnErr("there is no user_id in request", w)
		return
	}
	var _userId, _amount int
	err = s.db.QueryRow(`SELECT "user_id", "amount" FROM "users" where "user_id"=$1`, userId).Scan(&_userId, &_amount)
	switch {
	case err == sql.ErrNoRows:
		returnErr("there is no rows with given user_id", w)
		return
	case err != nil:
		returnErr("some problems in db query", w)
		return
	}
	usersRes := accountKeeper{Users: []user{user{UserId: userId, Amount: _amount}}}
	response, err := json.Marshal(usersRes)
	if err != nil {
		returnErr("err in result marshaling", w)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func (s *server) getUserHistory(w http.ResponseWriter, r *http.Request) {
	queryString := r.URL.Query()
	userId, err1 := strconv.Atoi(queryString.Get("user_id"))
	lastNOperations, err2 := strconv.Atoi(queryString.Get("n_last_operations"))
	if err1 != nil || err2 != nil {
		returnErr("there is no user_id in request or n_last_operations", w)
		return
	}
	var _userId, _amount int
	var _isDebit bool
	var _time time.Time
	historyRes := history{}
	historyRes.Histories = make([]historyEntry, 0)
	row, err := s.db.Query(`SELECT user_id, is_debit, amount, time FROM "history" where "user_id"=$1 ORDER BY id DESC LIMIT $2`,
		userId, lastNOperations)
	if err != nil {
		returnErr("some problems in request to db", w)
		return
	}
	defer row.Close()
	for row.Next() {
		err := row.Scan(&_userId, &_isDebit, &_amount, &_time)
		if err != nil {
			returnErr("some problems in scanning row", w)
			return
		}
		_tmpHist := historyEntry{_userId, _isDebit, _amount, _time}
		historyRes.Histories = append(historyRes.Histories, _tmpHist)
	}
	response, err := json.Marshal(historyRes)
	if err != nil {
		returnErr("err in result marshaling", w)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func (s *server) isUserCreated(userId int) (bool, error) {
	var u, a int
	err := s.db.QueryRow(`SELECT "user_id", "amount" FROM "users" where "user_id"=$1`, userId).Scan(&u, &a)
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, errors.New("DB_CONN_ERR")
	}
	return true, nil
}

func (s *server) topUpBalance(w http.ResponseWriter, r *http.Request) {
	queryString := r.URL.Query()
	userId, err1 := strconv.Atoi(queryString.Get("user_id"))
	accruedAmount, err2 := strconv.Atoi(queryString.Get("accrued_amount"))
	if err1 != nil || err2 != nil || accruedAmount <= 0 {
		returnErr("there is no user_id or accrued_amount in request", w)
		return
	}
	currentTime := time.Now()
	isUserCreated, err := s.isUserCreated(userId)
	if err != nil {
		returnErr("some problems in db query", w)
		return
	}
	if isUserCreated {
		_, err1 = s.db.Exec(`UPDATE "users" set "amount"="amount" + $1 where "user_id"=$2`, accruedAmount, userId)
		if err1 != nil {
			returnErr("problems in updating result ", w)
			return
		}
	} else {
		_, err1 = s.db.Exec(`INSERT INTO "users" (user_id, amount) VALUES ($1, $2)`, userId, accruedAmount)
		if err1 != nil {
			returnErr("problems in inserting result ", w)
			return
		}
	}
	_, err = s.db.Exec(`INSERT INTO "history" (user_id, is_debit, amount, time) VALUES ($1, $2, $3, $4)`,
		userId, false, accruedAmount, currentTime)
	if err != nil {
		returnErr("problems in inserting history table", w)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"Response": "ok" }`))
}

func (s *server) writeOffMoney(w http.ResponseWriter, r *http.Request) {
	queryString := r.URL.Query()
	userId, err1 := strconv.Atoi(queryString.Get("user_id"))
	debitedAmount, err2 := strconv.Atoi(queryString.Get("debited_amount"))
	if err1 != nil || err2 != nil || debitedAmount <= 0 {
		returnErr("there is no user_id or debited_amount in request", w)
		return
	}
	currentTime := time.Now()
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		returnErr("trans problem", w)
		return
	}
	defer tx.Rollback()
	var _userId, _amountUser int
	err1 = tx.QueryRowContext(ctx, `SELECT "user_id", "amount" FROM "users" where "user_id"=$1 FOR UPDATE`,
		userId).Scan(&_userId, &_amountUser)
	switch {
	case err1 == sql.ErrNoRows:
		returnErr("there is no user_id "+strconv.Itoa(userId), w)
		return
	case err1 != nil:
		returnErr("some problems in request to db", w)
		return
	}
	if _amountUser < debitedAmount {
		returnErr("ABORT: User amount should be greater or equal than debited_amount", w)
		return
	}
	_, err1 = tx.ExecContext(ctx, `UPDATE "users" set "amount"="amount" - $1 where "user_id"=$2`, debitedAmount, userId)
	if err1 != nil {
		returnErr("problems on setting res to db", w)
		return
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO "history" (user_id, is_debit, amount, time) VALUES ($1, $2, $3, $4)`,
		userId, true, debitedAmount, currentTime)
	if err != nil {
		returnErr("problems in updating history table", w)
		return
	}
	if err = tx.Commit(); err != nil {
		returnErr("transaction problem", w)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"Response": "ok" }`))
}

func (s *server) transferMoney(w http.ResponseWriter, r *http.Request) {
	queryString := r.URL.Query()
	fromUserId, err1 := strconv.Atoi(queryString.Get("from_user_id"))
	toUserId, err2 := strconv.Atoi(queryString.Get("to_user_id"))
	amount, err3 := strconv.Atoi(queryString.Get("amount"))
	if err1 != nil || err2 != nil || err3 != nil || amount <= 0 {
		returnErr("there is no user_id or update_amount in request", w)
		return
	}
	currentTime := time.Now()
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		returnErr("trans problem in creating transaction", w)
		return
	}
	defer tx.Rollback()
	var _userId1, _userId2, amountUserFrom, _amountUserTo int
	err1 = tx.QueryRowContext(ctx, `SELECT "user_id", "amount" FROM "users" where "user_id"=$1 FOR UPDATE`,
		fromUserId).Scan(&_userId1, &amountUserFrom)
	if err1 == sql.ErrNoRows {
		returnErr("there is no from_user_id "+strconv.Itoa(fromUserId), w)
		return
	}
	if err1 != nil {
		returnErr("some problems in request to db", w)
		return
	}
	if amountUserFrom < amount {
		returnErr("ABORT: Amount at from_user should be greater or equal than update_amount", w)
		return
	}

	err2 = tx.QueryRowContext(ctx, `SELECT "user_id", "amount" FROM "users" where "user_id"=$1 FOR UPDATE`,
		toUserId).Scan(&_userId2, &_amountUserTo)
	isNotExistedToUserId := false
	if err2 == sql.ErrNoRows {
		isNotExistedToUserId = true
	}
	if err2 != nil && err2 != sql.ErrNoRows {
		returnErr("some problems in request to db", w)
		return
	}
	// 	fmt.Println("sleep")
	// 	time.Sleep(20 * time.Second)
	_, err1 = tx.ExecContext(ctx, `UPDATE "users" set "amount"="amount" - $1 where "user_id"=$2`, amount, fromUserId)
	if isNotExistedToUserId {
		_, err2 = tx.ExecContext(ctx, `INSERT INTO "users" (user_id, amount) VALUES ($1, $2)`, toUserId, amount)
	} else {
		_, err2 = tx.ExecContext(ctx, `UPDATE "users" set "amount"="amount" + $1 where "user_id"=$2`, amount, toUserId)
	}
	if err1 != nil || err2 != nil {
		returnErr("problems on setting res to db", w)
		return
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO "history" (user_id, is_debit, amount, time) VALUES ($1, $2, $3, $4)`,
		fromUserId, true, amount, currentTime)
	if err != nil {
		returnErr("problems in updating history table", w)
		return
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO "history" (user_id, is_debit, amount, time) VALUES ($1, $2, $3, $4)`,
		toUserId, false, amount, currentTime)
	if err != nil {
		returnErr("problems in updating history table", w)
		return
	}
	if err = tx.Commit(); err != nil {
		returnErr("transaction problem", w)
		return
	}
	w.Write([]byte(`{"Response": "ok" }`))
}

func main() {
	db, err := sql.Open("postgres",
		"host=postgres port=5432 user=postgres password=postgres dbname=account_keeper sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	s := server{db: db}
	http.HandleFunc("/get_balance", s.getBalance)
	http.HandleFunc("/get_user_history", s.getUserHistory)
	http.HandleFunc("/top_up_balance", s.topUpBalance)
	http.HandleFunc("/write_off_money", s.writeOffMoney)
	http.HandleFunc("/transfer_money", s.transferMoney)
	log.Println("Starting server on :3000...")
	log.Fatal(http.ListenAndServe(":3000", nil))
	//  go mod init avito_server
	//  export GO111MODULE="on"
	// 	go get -u github.com/lib/pq
}
