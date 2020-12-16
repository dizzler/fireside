package tls

import (
    "bytes"
    "crypto/rsa"
    "crypto/x509"
    "encoding/pem"
    "io/ioutil"
    "os"
)

// encodes an X509 certificate in PEM format and returns the data as a byte array
func TlsEncodeCrt(caBytes []byte) []byte {
    caPem := new(bytes.Buffer)
    pem.Encode(caPem, &pem.Block{
        Type:  "CERTIFICATE",
        Bytes: caBytes,
    })
    return caPem.Bytes()
}

func TlsEncodePrivKey(privKey *rsa.PrivateKey) []byte {
    privKeyPem := new(bytes.Buffer)
    // PEM encode the private key
    pem.Encode(privKeyPem, &pem.Block{
        Type:  "RSA PRIVATE KEY",
        Bytes: x509.MarshalPKCS1PrivateKey(privKey),
    })
    return privKeyPem.Bytes()
}

// delete a given file
func TlsFileDelete(filepath string) error {
    if TlsFileExists(filepath) {
        if err := os.RemoveAll(filepath); err != nil {
            return err
        }
    }
    return nil
}

// TlsFileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func TlsFileExists(filepath string) bool {
    info, err := os.Stat(filepath)
    if os.IsNotExist(err) {
        return false
    }
    return !info.IsDir()
}

// encodes an RSA private key in PEM format and returns the data as a byte array
func TlsFileRead(filepath string) ([]byte, error) {
    if data, err := ioutil.ReadFile(filepath); err != nil {
        return nil, err
    } else {
        return data, nil
    }
}

// writes TLS bytes to file / disk
func TlsFileWrite(basedir string, filename string, data []byte, mode os.FileMode) error {
    // make sure the directory exists before writing file
    if err := TlsMkDir(basedir); err != nil {
        return err
    }
    // write the file
    filepath := basedir + "/" + filename
    if err := ioutil.WriteFile(filepath, data, mode); err != nil {
        return err
    }
    return nil
}

// makes a specified directory if it does not exist already
func TlsMkDir(dir string) error {
    _, err := os.Stat(dir)
    if os.IsNotExist(err) {
        // create the directory
        if merr := os.MkdirAll(dir, 0750); merr != nil {
            return merr
        }
    }
    return nil
}
