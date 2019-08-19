package main

import (
	"context"
	"io/ioutil"
	"os"

	"path/filepath"

	"crypto/tls"
	"crypto/x509"
	"database/sql"

	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
)

type mysqlDB struct {
	conn     *sql.DB
	addCount *sql.Stmt
	getCount *sql.Stmt
}

var (
	db *mysqlDB
)

func initData(ctx context.Context) error {

	// HACK: go mysql client doesn't read cnf files
	// https://github.com/go-sql-driver/mysql/issues/542
	dnsTLS, err := configTLS()
	if err != nil {
		return errors.Wrap(err, "Error configuring TLS")
	}

	c, err := sql.Open("mysql", connString+dnsTLS)
	if err != nil {
		return errors.Wrap(err, "Error connecting to DB")
	}

	if err := c.Ping(); err != nil {
		c.Close()
		return errors.Wrap(err, "Error connecting to DB")
	}

	db = &mysqlDB{
		conn: c,
	}

	if db.addCount, err = c.Prepare(`INSERT INTO counter
		(session_id, count_value) VALUES (?, ?) ON DUPLICATE KEY UPDATE
		count_value = count_value + 1`); err != nil {
		return errors.Wrap(err, "Error on addCount prepare")
	}

	if db.getCount, err = c.Prepare(`SELECT count_value
		FROM counter WHERE session_id = ?`); err != nil {
		return errors.Wrap(err, "Error on getCount prepare")
	}

	return nil

}

func configTLS() (dsnSufix string, err error) {

	certDirPath, e := getCertDirPath()
	if err != nil {
		return "", errors.Wrap(e, "Error getting cert dir path")
	}

	caPath := filepath.Join(certDirPath, "ca.pem")
	if _, err := os.Stat(caPath); os.IsNotExist(err) {
		return "", errors.Wrapf(err, "Required file does not exist: %s", caPath)
	}

	certPath := filepath.Join(certDirPath, "client.pem")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return "", errors.Wrapf(err, "Required file does not exist: %s", certPath)
	}

	keyPath := filepath.Join(certDirPath, "client.key")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return "", errors.Wrapf(err, "Required file does not exist: %s", keyPath)
	}

	rootCertPool := x509.NewCertPool()

	pem, err := ioutil.ReadFile(caPath)
	if err != nil {
		return "", errors.Wrapf(err, "Error reading cert: %s", caPath)
	}

	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return "", errors.Wrap(err, "Error appending PEM to TLS cert manager")
	}

	clientCert := make([]tls.Certificate, 0, 1)
	certs, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return "", errors.Wrap(err, "Error loading certs")
	}
	clientCert = append(clientCert, certs)

	mysql.RegisterTLSConfig("custom", &tls.Config{
		RootCAs:            rootCertPool,
		Certificates:       clientCert,
		InsecureSkipVerify: true,
	})

	return "&tls=custom", nil

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
