package fireside

import (
    "bytes"
    "crypto/rand"
    "crypto/rsa"
    "crypto/tls"
    "crypto/x509"
    "crypto/x509/pkix"
    "encoding/pem"
    "errors"
    "fmt"
    "io/ioutil"
    "math/big"
    "net"
    "net/http"
    "net/http/httptest"
    "os"
    "strings"
    "time"

    configure "fireside/pkg/configure"
)

type TlsTrust struct {
    Name      string
    BaseDir   string
    CrtConfig *TlsCrtConfig
    CrtData   bytes.Buffer
    CrtFile   string
    KeyData   bytes.Buffer
    KeyFile   string
}

type TlsCrtConfig struct {
    CommonName    string
    Country       string
    IPAddresses   []net.IP
    Locality      string
    Organization  string
    PostalCode    string
    Province      string
    StreetAddress string
}

func NewTlsTrust (config *configure.EnvoySecret) (*TlsTrust, error) {
    var (
        ipAddressList []net.IP
        tlsCrtConfig  *TlsCrtConfig
    )
    switch config.Type {
    case configure.SecretTypeTlsCa:
        tlsCrtConfig = &TlsCrtConfig{
            CommonName: config.Crt.CommonName,
            Country: config.Crt.Country,
            Locality: config.Crt.Locality,
            Organization: config.Crt.Organization,
            PostalCode: config.Crt.PostalCode,
            Province: config.Crt.Province,
            StreetAddress: config.Crt.StreetAddress,
        }
    case configure.SecretTypeTlsClient || configure.SecretTypeTlsServer:
        for _, ip_string := range config.Crt.IpAddresses {
            ipAddressList = append(ipAddressList, parseIP(ip_string))
        }
        tlsCrtConfig = &TlsCrtConfig{
            CommonName: config.Crt.CommonName,
            Country: config.Crt.Country,
            IPAddresses: ipAddressList,
            Locality: config.Crt.Locality,
            Organization: config.Crt.Organization,
            PostalCode: config.Crt.PostalCode,
            Province: config.Crt.Province,
            StreetAddress: config.Crt.StreetAddress,
        }
    default:
        return nil, errors.New("failed to create NewTlsTrust for unsupported config Type = " + config.Type)
    }
    return &TlsTrust{
        Name: config.Name,
        BaseDir: config.BaseDir,
        CrtConfig: tlsCrtConfig,
        CrtFile: config.Crt.FileName,
        KeyFile: config.Key.FileName,
    }
}

// data structure for storing TLS configs, keys and other secret data
type TlsTrustDomain struct {
    Name    string
    CA      TlsTrust
    Clients []TlsTrust
    Servers []TlsTrust
}

// creates a new TlsTrustDomain
func NewTlsTrustDomain(config *configure.EnvoySecret) (*TlsTrustDomain, error) {
    if config.Type != configure.SecretTypeTlsCa {
        return nil, errors.New("cannot create NewTlsTrustDomain for secret type = " + config.Type)
    }
    ca := NewTlsTrust(config)
    return &TlsTrustDomain{
        Name: config.Name,
        CA: ca,
    }
}

/*
    methods for TlsTrustDomain
*/
// gets a buffer containing bytes for CA certificate
func (td *TlsTrustDomain) GetCaBytes() bytes.Buffer {
    return td.CA.CrtData
}

// gets a buffer containing bytes for server certificate and key
func (td *TlsTrustDomain) GetClientBytes(element int32, secret *configure.EnvoySecret) (string, bytes.Buffer, bytes.Buffer, error) {
    switch {
    case element < len(td.Clients):
        return td.Clients[element].Name, td.Clients[element].CrtData, td.Clients[element].KeyData, nil
    case element == len(td.Clients):
        return td.NewTlsPem(secret)
    default:
        return nil, nil, nil, errors.New("invalid GetClienBytes request for array element ", element)
    }
}

// gets a buffer containing bytes for server certificate and key
func (td *TlsTrustDomain) GetServerBytes(element int32, secret *configure.EnvoySecret) (string, bytes.Buffer, bytes.Buffer, error) {
    switch {
    case element < len(td.Servers):
        return td.Servers[element].Name, td.Servers[element].CrtData, td.Servers[element].KeyData, nil
    case element == len(td.Servers):
        return td.NewTlsPem(secret)
    default:
        return nil, nil, nil, errors.New("invalid GetClienBytes request for array element ", element)
    }
}

// creates a new PEM certificate by signing certificate with existing CA key
func (td *TlsTrustDomain) NewTlsPem(config *configure.EnvoySecret) (bytes.Buffer, bytes.Buffer, error) {
    // set up our server certificate
    cert := &x509.Certificate{
        SerialNumber: big.NewInt(2019),
        Subject: pkix.Name{
            Organization:  []string{config.Crt.Organization},
            Country:       []string{config.Crt.Country},
            Province:      []string{config.Crt.Province},
            Locality:      []string{config.Crt.Locality},
            StreetAddress: []string{config.Crt.StreetAddress},
            PostalCode:    []string{config.Crt.PostalCode},
        },
        IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
        NotBefore:    time.Now(),
        NotAfter:     time.Now().AddDate(10, 0, 0),
        SubjectKeyId: []byte{1, 2, 3, 4, 6},
        ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
        KeyUsage:     x509.KeyUsageDigitalSignature,
    }

    certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
    if err != nil {
        return nil, nil, err
    }

    certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
    if err != nil {
        return nil, nil, err
    }

    certPEM := new(bytes.Buffer)
    pem.Encode(certPEM, &pem.Block{
        Type:  "CERTIFICATE",
        Bytes: certBytes,
    })

    certPrivKeyPEM := new(bytes.Buffer)
    pem.Encode(certPrivKeyPEM, &pem.Block{
        Type:  "RSA PRIVATE KEY",
        Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
    })

    return certPEM.Bytes(), certPrivKeyPEM.Bytes(), nil
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

func TlsFileRead(filepath string) ([]byte, error) {
    if data, err := ioutil.ReadFile(filepath); err != nil {
        return nil, err
    } else {
        return data, nil
    }
}



//
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
