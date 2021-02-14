package sess

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"

	_ "github.com/go-sql-driver/mysql" // Mysql driver
)

// AlreadyIn tells if users is alread connected
type AlreadyIn struct {
	logged bool
}

type userSessionData struct {
	uID string
}

type sessionLogged struct {
	expired string
}

type sessionLogin string

var (
	in       = AlreadyIn{}
	uSesData = userSessionData{}
	s        sessionLogin
)

// Session Initialize or Purge sesion when timeout
func Session(ip string) bool {
	db, err := sql.Open("mysql", "joaoR:Joao_846515_AX@/MSOCIAL")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	query, err := db.Query(`
		SELECT 
			IF(COUNT(*), 'true', 'false')
		FROM http 
		WHERE cliente_id = ?`, ip)
	if err != nil {
		panic(err)
	}
	defer query.Close()

	for query.Next() {
		query.Scan(&s)

		if s == "true" {
			in.ChangeLogState(true)
		} else {
			in.ChangeLogState(false)
		}
	}
	return in.logged
}

// ChangeLogState changes Session state
func (a *AlreadyIn) ChangeLogState(op bool) {
	if op {
		a.logged = true
	} else {
		a.logged = false
	}
}

func (u *userSessionData) setSessionData(sessID string) bool {
	u.uID = sessID
	return true
}

func randSeq(n int) string {
	var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// CheckSession initialize the session
func CheckSession(ip string) bool {
	var sessionExp bool

	db, err := sql.Open("mysql", "joaoR:Joao_846515_AX@/MSOCIAL")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// hash
	//hash := randSeq(40)
	query, err := db.Query(`
		SELECT IF(DATEDIFF(
			DATE(current_timestamp),
			DATE(log_date)
		) >= 1
		OR
		TIMEDIFF(
			TIME(current_timestamp),
			TIME(log_date)
		) >= 1
	, "true","false") as valid_session
	from http
	where cliente_id = ?
	`, ip)
	if err != nil {
		panic(err)
	}
	defer query.Close()
	for query.Next() {
		var sessionLoged sessionLogged
		query.Scan(&sessionLoged.expired)

		if sessionLoged.expired == "true" {
			sessionExp = false //Expirado
		} else {
			sessionExp = true // Não expirado
		}
	}
	return sessionExp
}

// SetSession inits user navigation
func SetSession(ip, username, usPasswd string) error {
	var usExists int

	sha := sha256.New()
	sha.Write([]byte(usPasswd))

	hashPass := fmt.Sprintf("%x", sha.Sum(nil))

	hash := randSeq(40)
	db, err := sql.Open("mysql", "joaoR:Joao_846515_AX@/MSOCIAL")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	result, _ := db.Query(`
		SELECT COUNT(*) AS CONTADOR FROM users
		WHERE username = ? and user_password = ?
	`, username, hashPass)
	defer result.Close()

	for result.Next() {
		result.Scan(&usExists)
	}

	if usExists == 0 {
		err = errors.New("User does not exists, check your credential and try again")
	} else {
		err = errors.New("")
		// Exclui o user ja registrado no monitor de login
		stmt, _ := db.Prepare(`
			DELETE FROM http where cliente_id = ?
		`)
		stmt.Exec(ip)

		stmt, err = db.Prepare(`
			INSERT INTO http(
				cliente_id,
				gen_key
			)
			VALUES(
				?,?
			)
		`)
		stmt.Exec(ip, hash)
	}
	return err
}

// GetIP returns the ip of client
func GetIP(r *http.Request) (string, error) {
	ip := r.Header.Get("X-REAL-IP")
	netIP := net.ParseIP(ip)
	if netIP != nil {
		return ip, nil
	}

	ips := r.Header.Get("X-FORWARDED-FOR")
	splitIps := strings.Split(ips, ",")
	for _, ip := range splitIps {
		netIP := net.ParseIP(ip)
		if netIP != nil {
			return ip, nil
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}
	netIP = net.ParseIP(ip)
	if netIP != nil {
		return ip, nil
	}
	return "", fmt.Errorf("No valid ip found")
}
