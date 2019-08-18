package main

import (
	"context"
	"errors"
	"io/ioutil"

	"os/user"
	"path/filepath"

	"crypto/tls"
	"crypto/x509"
	"database/sql"

	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/mysql"
	"github.com/go-sql-driver/mysql"
)

type mysqlDB struct {
	conn     *sql.DB
	addCount *sql.Stmt
	getCount *sql.Stmt
}

var (
	db *mysqlDB
)

func initData(ctx context.Context) {

	loadClientCerts()

	c, err := sql.Open("mysql", connString)
	if err != nil {
		logger.Fatalf("Error connecting to DB (%s): %v", connString, err)
	}

	if err := c.Ping(); err != nil {
		c.Close()
		logger.Fatalf("Error connecting to DB: %v", err)
	}

	db = &mysqlDB{
		conn: c,
	}

	if db.addCount, err = c.Prepare(`INSERT INTO counter
		(session_id, count_value) VALUES (?, ?) ON DUPLICATE KEY UPDATE
		count_value = count_value + 1`); err != nil {
		logger.Fatalf("Error on addCount prepare: %v", err)
	}

	if db.getCount, err = c.Prepare(`SELECT count_value
		FROM counter WHERE session_id = ?`); err != nil {
		logger.Fatalf("Error on getCount prepare: %v", err)
	}

}

func loadClientCerts() {
	user, err := user.Current()
	checkFatalErr(err, "Error getting current user")

	localCertDirPath = filepath.Join(user.HomeDir, localCertDirName)

	rootCertPool := x509.NewCertPool()
	pem, err := ioutil.ReadFile(filepath.Join(localCertDirPath, "ca.pem"))
	if err != nil {
		logger.Fatal(err)
	}
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		logger.Fatal("Failed to append PEM")
	}

	clientCert := make([]tls.Certificate, 0, 1)
	certs, err := tls.LoadX509KeyPair(filepath.Join(localCertDirPath, "client.pem"),
		filepath.Join(localCertDirPath, "client.key"))
	if err != nil {
		logger.Fatal(err)
	}
	clientCert = append(clientCert, certs)
	mysql.RegisterTLSConfig("custom", &tls.Config{
		RootCAs:      rootCertPool,
		Certificates: clientCert,
	})

}

func finalizeData(ctx context.Context) {
	if db != nil && db.conn != nil {
		db.conn.Close()
	}
}

func countSession(ctx context.Context, sessionID string) (c int64, err error) {

	if sessionID == "" {
		logger.Println("Session ID required")
		return 0, errors.New("Session ID required")
	}

	tx, e := db.conn.Begin()
	if e != nil {
		logger.Printf("Error while creating transaction: %v", e)
	}

	_, e = tx.Stmt(db.addCount).Exec(sessionID, 1)
	if e != nil {
		tx.Rollback()
		logger.Printf("Error while incrementing sessions %s: %v", sessionID, e)
	}

	rows, err := tx.Stmt(db.getCount).Query(sessionID)
	if err != nil {
		tx.Rollback()
		logger.Printf("Error while quering session %s: %v", sessionID, err)
		return 0, err
	}
	defer rows.Close()

	var sessionCount int64
	for rows.Next() {
		if err := rows.Scan(&sessionCount); err != nil {
			tx.Rollback()
			logger.Printf("Error parsing session incrementing results: %v", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		logger.Printf("Error committing session incrementing: %v", err)
		return 0, err
	}

	logger.Printf("Session incrementing result: %d", sessionCount)

	return sessionCount, nil

}
