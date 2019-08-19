package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	cloudkms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/storage"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"

	"github.com/pkg/errors"
)

const (
	certDirName = ".certs"
)

var (
	gcsClient *storage.Client
	kmsClient *cloudkms.KeyManagementClient
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

	kmsc, err := cloudkms.NewKeyManagementClient(ctx)
	if err != nil {
		return errors.Wrap(err, "Error while creating KMS client")
	}
	kmsClient = kmsc

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

	data, err := ioutil.ReadAll(o)
	if err != nil {
		return errors.Wrap(err, "Error reading object content")
	}

	keyID := fmt.Sprintf("%s/cryptoKeys/config", kmsKeyRing)
	dr := &kmspb.DecryptRequest{
		Name:       keyID,
		Ciphertext: data,
	}

	resp, err := kmsClient.Decrypt(ctx, dr)
	if err != nil {
		return errors.Wrapf(err, "Error decrypting using key %s", keyID)
	}

	// write
	certPath := filepath.Join(certDirPath, object)
	err = ioutil.WriteFile(certPath, []byte(resp.GetPlaintext()), 0644)
	if err != nil {
		return errors.Wrapf(err, "Error writting decrypted content to %s", certPath)
	}

	return nil

}
