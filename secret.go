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
)

const (
	localCertDirName = ".certs"
	localConfigName  = ".my.cnf"
)

var (
	gcsClient        *storage.Client
	kmsClient        *cloudkms.KeyManagementClient
	localCertDirPath = ""
	localConfigPath  = ""
)

func initSecrets(ctx context.Context) {

	user, err := user.Current()
	checkFatalErr(err, "Error getting current user")

	localCertDirPath = filepath.Join(user.HomeDir, localCertDirName)
	_ = os.Mkdir(localCertDirPath, os.ModeDir)
	localConfigPath = filepath.Join(user.HomeDir, localConfigName)

	gcsc, err := storage.NewClient(ctx)
	checkFatalErr(err, "Error while creating GCS client")
	gcsClient = gcsc

	kmsc, err := cloudkms.NewKeyManagementClient(ctx)
	checkFatalErr(err, "Error while creating KMS client")
	kmsClient = kmsc

	err = setupCert(ctx, "client.pem")
	checkFatalErr(err, "Error processing client.pem")

	err = setupCert(ctx, "client.key")
	checkFatalErr(err, "Error processing client.key")

	err = setupCert(ctx, "ca.pem")
	checkFatalErr(err, "Error processing ca.pem")

	// write mysql config file
	confContent := fmt.Sprintf(`[client]
ssl-mode=VERIFY_CA
ssl-ca=%s/ca.pem
ssl-cert=%s/client.pem
ssl-key=%s/client.key
`, localCertDirPath, localCertDirPath, localCertDirPath)

	err = ioutil.WriteFile(localConfigPath, []byte(confContent), 0644)
	checkFatalErr(err, "Error while saving %s: %v", localConfigPath, err)

}

func setupCert(ctx context.Context, object string) error {

	// download
	o, err := gcsClient.Bucket(certBucket).Object(object).NewReader(ctx)
	if err != nil {
		logger.Printf("Error getting object %s/%s: %v", certBucket, object, err)
		return err
	}
	defer o.Close()

	data, err := ioutil.ReadAll(o)
	if err != nil {
		logger.Printf("Error reading object content: %v", err)
		return err
	}

	keyID := fmt.Sprintf("%s/cryptoKeys/config", kmsKeyRing)
	dr := &kmspb.DecryptRequest{
		Name:       keyID,
		Ciphertext: data,
	}

	resp, err := kmsClient.Decrypt(ctx, dr)
	if err != nil {
		logger.Printf("Error decrypting using key %s: %v", keyID, err)
		return err
	}

	// write
	certPath := filepath.Join(localCertDirPath, object)
	err = ioutil.WriteFile(certPath, []byte(resp.GetPlaintext()), 0644)
	if err != nil {
		logger.Printf("Error writting decrypted content to %s: %v", certPath, err)
		return err
	}

	return nil

}

func checkFatalErr(err error, msg string, args ...interface{}) {
	if err != nil {
		logger.Printf(msg, args)
		logger.Fatal(err)
	}
}
