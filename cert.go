package main

import (
	"context"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"cloud.google.com/go/storage"
	"github.com/pkg/errors"
)

const (
	certDirName = ".certs"
)

var (
	gcsClient *storage.Client
)

func getCertDirPath() (path string, err error) {
	u, e := user.Current()
	if e != nil {
		return "", errors.Wrap(e, "Error getting current user")
	}
	return filepath.Join(u.HomeDir, certDirName), nil
}

func initCertificates(ctx context.Context) error {

	certDirPath, err := getCertDirPath()
	if err != nil {
		return err
	}

	_, err = os.Stat(certDirPath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(certDirPath, 0700)
		if err != nil {
			return errors.Wrapf(err, "Error creating certificate dir (%s)", certDirPath)
		}
	} else {
		return errors.Wrapf(err, "Error getting stats from %s", certDirPath)
	}

	gcsc, err := storage.NewClient(ctx)
	if err != nil {
		return errors.Wrap(err, "Error while creating GCS client")
	}
	gcsClient = gcsc

	err = configureCert(ctx, certDirPath, "client.pem")
	if err != nil {
		return errors.Wrap(err, "Error processing client.pem")
	}

	err = configureCert(ctx, certDirPath, "client.key")
	if err != nil {
		return errors.Wrap(err, "Error processing  client.key")
	}

	err = configureCert(ctx, certDirPath, "ca.pem")
	if err != nil {
		return errors.Wrap(err, "Error processing ca.pem")
	}

	return nil

}

func configureCert(ctx context.Context, certDirPath, object string) error {

	// download
	o, err := gcsClient.Bucket(certBucket).Object(object).NewReader(ctx)
	if err != nil {
		return errors.Wrapf(err, "Error getting object %s/%s", certBucket, object)
	}
	defer o.Close()

	// read
	data, err := ioutil.ReadAll(o)
	if err != nil {
		return errors.Wrap(err, "Error reading object content")
	}

	// write
	certPath := filepath.Join(certDirPath, object)
	err = ioutil.WriteFile(certPath, data, 0644)
	if err != nil {
		return errors.Wrapf(err, "Error writting decrypted content to %s", certPath)
	}

	return nil

}
