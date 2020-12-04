package fireside

import (
    "bytes"
    "crypto/rsa"
    "crypto/x509"
    "encoding/pem"
    "io/ioutil"
    "os"
)

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

/*
func GetTlsPem(config *configure.EnvoySecret) (bytes.Buffer, bytes.Buffer, error) {
    // created a nested function for re-use within switch
    mkPem := func() (bytes.Buffer, bytes.Buffer, error) {
            if crtData, keyData, err := NewTlsPem(config); err != nil {
                return nil, nil, err
            }
            return crtData, keyData, nil
    }
    // get or create TLS cert and key, using existing files if allowed/present
    switch {
    case !TlsFileExists(config.Ca.FilePath):
        return nil, nil, errors.New("cannot GetTlsPem because CA cert does not exist at path " + config.Ca.FilePath)
    case TlsFileExists(config.Crt.FilePath) && TlsFileExists(config.Key.FilePath):
        if config.Provision.ForceRecreate {
            mkPem()
        } else {
            if crtData, err := TlsFileRead(config.Crt.FilePath); err != nil {
                return nil, nil, err
            }
            if keyData, err := TlsFileRead(config.Key.FilePath); err != nil {
                return nil, nil, err
            }
            return crtData, keyData, nil
        }
    case !TlsFileExists(config.Crt.FilePath) && !TlsFileExists(config.Key.FilePath):
        if config.Provision.CreateIfAbsent {
            mkPem()
        } else {
            return nil, nil, errors.New("cannot create secret : " + config.Name)
        }
    default:
        TlsFileDelete(config.Crt.FilePath)
        TlsFileDelete(config.Key.FilePath)
        mkPem()
    }
}
*/
