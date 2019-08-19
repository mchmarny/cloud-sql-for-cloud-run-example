package main

import (
	"context"
	"errors"
	"io/ioutil"
	"os"

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

	// HACK: go mysql client doesn't read cnf files
	// https://github.com/go-sql-driver/mysql/issues/542
	dnsTLS := configTLS()

	c, err := sql.Open("mysql", connString+dnsTLS)
	if err != nil {
		logger.Fatalf("Error connecting to DB: %v", err)
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

func configTLS() string {

	//certDirPath := getCertDirPath()
	certDirPath := "/Users/mchmarny/.cloud-sql/cloudylabs/demo3"
	caPath := filepath.Join(certDirPath, "ca.pem")
	failIfFileNotExists(caPath)
	certPath := filepath.Join(certDirPath, "client.pem")
	failIfFileNotExists(certPath)
	keyPath := filepath.Join(certDirPath, "client.key")
	failIfFileNotExists(keyPath)

	rootCertPool := x509.NewCertPool()

	pem, err := ioutil.ReadFile(caPath)
	if err != nil {
		logger.Fatal(err)
	}

	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		logger.Fatal("Failed to append PEM")
	}

	clientCert := make([]tls.Certificate, 0, 1)
	certs, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		logger.Fatal(err)
	}
	clientCert = append(clientCert, certs)

	mysql.RegisterTLSConfig("custom", &tls.Config{
		RootCAs:            rootCertPool,
		Certificates:       clientCert,
		InsecureSkipVerify: true,
	})

	return "&tls=custom"

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

func failIfFileNotExists(filename string) {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) || info.IsDir() {
		logger.Fatalf("Required file does not exist: %s", filename)
	}
}
