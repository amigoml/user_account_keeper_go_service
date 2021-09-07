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
	User_id int
	Amount  int
}

type account_keeper struct {
	Users []user
}

type history_entry struct {
	User_id    int
	Is_debit   bool
	Amount     int
	Trans_time time.Time
}

type history struct {
	Histories []history_entry
}

func returnErr(s string, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(s))
}

func (s *server) get_balance(w http.ResponseWriter, r *http.Request) {
	queryString := r.URL.Query()
	user_id, err := strconv.Atoi(queryString.Get("user_id"))
	if err != nil {
		returnErr("there is no user_id in request", w)
		return
	}
	var _user_id, amount int
	err = s.db.QueryRow(`SELECT "user_id", "amount" FROM "users" where "user_id"=$1`, user_id).Scan(&_user_id, &amount)
	switch {
	case err == sql.ErrNoRows:
		returnErr("there is no rows with given user_id", w)
		return
	case err != nil:
		returnErr("some problems in db query", w)
		return
	}
	users_res := account_keeper{Users: []user{user{User_id: user_id, Amount: amount}}}
	response, err := json.Marshal(users_res)
	if err != nil {
		returnErr("err in result marshaling", w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func (s *server) get_user_history(w http.ResponseWriter, r *http.Request) {
	queryString := r.URL.Query()
	user_id, err1 := strconv.Atoi(queryString.Get("user_id"))
	last_n_operations, err2 := strconv.Atoi(queryString.Get("n_last_operations"))
	if err1 != nil || err2 != nil {
		returnErr("there is no user_id in request or n_last_operations", w)
		return
	}
	var _user_id, amount int
	var is_debit bool
	var t time.Time
	history_res := history{}
	history_res.Histories = make([]history_entry, 0)
	row, err := s.db.Query(`SELECT user_id, is_debit, amount, time FROM "history" where "user_id"=$1 ORDER BY id DESC LIMIT $2`,
		user_id, last_n_operations)
	if err != nil {
		returnErr("some problems in request to db", w)
		return
	}
	defer row.Close()
	for row.Next() {
		row.Scan(&_user_id, &is_debit, &amount, &t)
		tmp_hist := history_entry{_user_id, is_debit, amount, t}
		history_res.Histories = append(history_res.Histories, tmp_hist)
	}
	response, err := json.Marshal(history_res)
	if err != nil {
		returnErr("err in result marshaling", w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func (s *server) isUserCreated(user_id int) (bool, error) {
	var u, a int
	err := s.db.QueryRow(`SELECT "user_id", "amount" FROM "users" where "user_id"=$1`, user_id).Scan(&u, &a)
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, errors.New("DB_CONN_ERR")
	}
	return true, nil
}

func (s *server) top_up_balance(w http.ResponseWriter, r *http.Request) {
	queryString := r.URL.Query()
	user_id, err1 := strconv.Atoi(queryString.Get("user_id"))
	accrued_amount, err2 := strconv.Atoi(queryString.Get("accrued_amount"))
	if err1 != nil || err2 != nil || accrued_amount <= 0 {
		returnErr("there is no user_id or accrued_amount in request", w)
		return
	}
	current_time := time.Now()
	is_user_created, err := s.isUserCreated(user_id)
	if err != nil {
		returnErr("some problems in db query", w)
		return
	}
	if is_user_created {
		_, err1 = s.db.Exec(`UPDATE "users" set "amount"="amount" + $1 where "user_id"=$2`, accrued_amount, user_id)
		if err1 != nil {
			returnErr("problems in updating result ", w)
			return
		}
	} else {
		_, err1 = s.db.Exec(`INSERT INTO "users" (user_id, amount) VALUES ($1, $2)`, user_id, accrued_amount)
		if err1 != nil {
			returnErr("problems in inserting result ", w)
			return
		}
	}
	_, err = s.db.Exec(`INSERT INTO "history" (user_id, is_debit, amount, time) VALUES ($1, $2, $3, $4)`,
		user_id, false, accrued_amount, current_time)
	if err != nil {
		returnErr("problems in inserting history table", w)
		return
	}
	w.Write([]byte("ok"))
}

func (s *server) write_off_money(w http.ResponseWriter, r *http.Request) {
	queryString := r.URL.Query()
	user_id, err1 := strconv.Atoi(queryString.Get("user_id"))
	debited_amount, err2 := strconv.Atoi(queryString.Get("debited_amount"))
	if err1 != nil || err2 != nil || debited_amount <= 0 {
		returnErr("there is no user_id or debited_amount in request", w)
		return
	}
	current_time := time.Now()
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		returnErr("trans problem", w)
		return
	}
	defer tx.Rollback()
	var _user_id, _amount_user int
	err1 = tx.QueryRowContext(ctx, `SELECT "user_id", "amount" FROM "users" where "user_id"=$1 FOR UPDATE`,
		user_id).Scan(&_user_id, &_amount_user)
	switch {
	case err1 == sql.ErrNoRows:
		returnErr("there is no user_id "+strconv.Itoa(user_id), w)
		return
	case err1 != nil:
		returnErr("some problems in request to db", w)
		return
	}
	if _amount_user < debited_amount {
		returnErr("ABORT: User amount should be greater or equal than debited_amount", w)
		return
	}
	_, err1 = tx.ExecContext(ctx, `UPDATE "users" set "amount"="amount" - $1 where "user_id"=$2`, debited_amount, user_id)
	if err1 != nil {
		returnErr("problems on setting res to db", w)
		return
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO "history" (user_id, is_debit, amount, time) VALUES ($1, $2, $3, $4)`,
		user_id, true, debited_amount, current_time)
	if err != nil {
		returnErr("problems in updating history table", w)
		return
	}
	if err = tx.Commit(); err != nil {
		returnErr("transaction problem", w)
		return
	}
	w.Write([]byte("ok"))
}

func (s *server) transfer_money(w http.ResponseWriter, r *http.Request) {
	queryString := r.URL.Query()
	from_user_id, err1 := strconv.Atoi(queryString.Get("from_user_id"))
	to_user_id, err2 := strconv.Atoi(queryString.Get("to_user_id"))
	amount, err3 := strconv.Atoi(queryString.Get("amount"))
	if err1 != nil || err2 != nil || err3 != nil || amount <= 0 {
		returnErr("there is no user_id or update_amount in request", w)
		return
	}
	current_time := time.Now()
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		returnErr("trans problem in creating transaction", w)
		return
	}
	defer tx.Rollback()
	var _user_id_1, _user_id_2, amount_user_from, amount_user_to int
	err1 = tx.QueryRowContext(ctx, `SELECT "user_id", "amount" FROM "users" where "user_id"=$1 FOR UPDATE`,
		from_user_id).Scan(&_user_id_1, &amount_user_from)
	if err1 == sql.ErrNoRows {
		returnErr("there is no from_user_id "+strconv.Itoa(from_user_id), w)
		return
	}
	if err1 != nil {
		returnErr("some problems in request to db", w)
		return
	}
	if amount_user_from < amount {
		returnErr("ABORT: Amount at from_user should be greater or equal than update_amount", w)
		return
	}

	err2 = tx.QueryRowContext(ctx, `SELECT "user_id", "amount" FROM "users" where "user_id"=$1 FOR UPDATE`,
		to_user_id).Scan(&_user_id_2, &amount_user_to)
	is_no_to_user_id := false
	if err2 == sql.ErrNoRows {
		is_no_to_user_id = true
	}
	if err2 != nil && err2 != sql.ErrNoRows {
		returnErr("some problems in request to db", w)
		return
	}
	// 	fmt.Println("sleep")
	// 	time.Sleep(20 * time.Second)
	_, err1 = tx.ExecContext(ctx, `UPDATE "users" set "amount"="amount" - $1 where "user_id"=$2`, amount, from_user_id)
	if is_no_to_user_id {
		_, err2 = tx.ExecContext(ctx, `INSERT INTO "users" (user_id, amount) VALUES ($1, $2)`, to_user_id, amount)
	} else {
		_, err2 = tx.ExecContext(ctx, `UPDATE "users" set "amount"="amount" + $1 where "user_id"=$2`, amount, to_user_id)
	}
	if err1 != nil || err2 != nil {
		returnErr("problems on setting res to db", w)
		return
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO "history" (user_id, is_debit, amount, time) VALUES ($1, $2, $3, $4)`,
		from_user_id, true, amount, current_time)
	if err != nil {
		returnErr("problems in updating history table", w)
		return
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO "history" (user_id, is_debit, amount, time) VALUES ($1, $2, $3, $4)`,
		to_user_id, false, amount, current_time)
	if err != nil {
		returnErr("problems in updating history table", w)
		return
	}
	if err = tx.Commit(); err != nil {
		returnErr("transaction problem", w)
		return
	}
	w.Write([]byte("ok"))
}

func main() {
	db, err := sql.Open("postgres", "host=postgres port=5432 user=postgres password=postgres dbname=account_keeper sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	s := server{db: db}
	http.HandleFunc("/get_balance", s.get_balance)
	http.HandleFunc("/get_user_history", s.get_user_history)
	http.HandleFunc("/top_up_balance", s.top_up_balance)
	http.HandleFunc("/write_off_money", s.write_off_money)
	http.HandleFunc("/transfer_money", s.transfer_money)
	log.Println("Starting server on :3000...")
	log.Fatal(http.ListenAndServe(":3000", nil))
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	//  go mod init avito_server
	//  export GO111MODULE="on"
	// 	go get -u github.com/lib/pq
}
