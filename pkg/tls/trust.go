package fireside

import (
    "crypto/rand"
    "crypto/rsa"
    "crypto/x509"
    "crypto/x509/pkix"
    "errors"
    "math/big"
    "net"
    "time"

    configure "fireside/pkg/configure"

    log "github.com/sirupsen/logrus"
)

// data structure for storing TLS configs, keys and other secret data
type TlsTrustDomain struct {
    Name    string
    CA      *TlsTrust
    Clients []*TlsTrust
    Servers []*TlsTrust
}

type TlsTrust struct {
    Name      string
    Type      string
    BaseDir   string
    Crt       *TlsTrustCrt
    Key       *TlsTrustKey
    Provision *TlsTrustProvision
}

type TlsTrustCrt struct {
    Config *TlsTrustCrtConfig
    File   string
    PEM    []byte
    X509   *x509.Certificate
}

type TlsTrustCrtConfig struct {
    CommonName    string
    Country       string
    DNSNames      []string
    IPAddresses   []net.IP
    Locality      string
    Organization  string
    PostalCode    string
    Province      string
    StreetAddress string
}

type TlsTrustKey struct {
    File string
    Key  *rsa.PrivateKey
    PEM  []byte
}

type TlsTrustProvision struct {
    CreateIfAbsent bool
    ForceRecreate  bool
}

// creates a new TlsTrust from secret config
func NewTlsTrust(config *configure.EnvoySecret) (*TlsTrust, error) {
    var (
        ipAddressList []net.IP
        tlsCrtConfig  *TlsTrustCrtConfig
    )
    switch config.Type {
    case configure.SecretTypeTlsCa:
        tlsCrtConfig = &TlsTrustCrtConfig{
            CommonName: config.Crt.CommonName,
            Country: config.Crt.Country,
            Locality: config.Crt.Locality,
            Organization: config.Crt.Organization,
            PostalCode: config.Crt.PostalCode,
            Province: config.Crt.Province,
            StreetAddress: config.Crt.StreetAddress,
        }
    case configure.SecretTypeTlsClient,configure.SecretTypeTlsServer:
        for _, ip_string := range config.Crt.IpAddresses {
            ipAddressList = append(ipAddressList, net.ParseIP(ip_string))
            // append loopback addresses to list of certificate IPs
            ipAddressList = append(ipAddressList, net.IPv4(127, 0, 0, 1), net.IPv6loopback)
        }
        tlsCrtConfig = &TlsTrustCrtConfig{
            CommonName: config.Crt.CommonName,
            Country: config.Crt.Country,
	    DNSNames: config.Crt.DnsNames,
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
    tlsTrustCrt := &TlsTrustCrt{
        Config: tlsCrtConfig,
        File: config.Crt.FileName,
    }
    tlsTrustKey := &TlsTrustKey{
        File: config.Key.FileName,
    }
    provision := &TlsTrustProvision{
        CreateIfAbsent: config.Provision.CreateIfAbsent,
        ForceRecreate: config.Provision.ForceRecreate,
    }
    return &TlsTrust{
        Name: config.Name,
	Type: config.Type,
        BaseDir: config.BaseDir,
        Crt: tlsTrustCrt,
        Key: tlsTrustKey,
	Provision: provision,
    }, nil
}

// writes the PEM-formatted certificate to local file and, for
// non-CA certs/trusts, write the PEM-formatted key to local file, too.
func (tr *TlsTrust) WriteTrustFiles() error {
    if tr.Type == configure.SecretTypeTlsCa {
    }
    // create a func for writing certificate and key to local files
    writeCrtFile := func() error {
        if err := TlsFileWrite(tr.BaseDir, tr.Crt.File, tr.Crt.PEM, configure.FileModeCrt); err != nil {
            return err
        }
        return nil
    }
    writeKeyFile := func() error {
        if err := TlsFileWrite(tr.BaseDir, tr.Key.File, tr.Key.PEM, configure.FileModeKey); err != nil {
            return err
        }
        return nil
    }
    mkMyFiles := func() error {
        if err := writeCrtFile(); err != nil { return err }
        if tr.Type == configure.SecretTypeTlsCa {
            log.Debug("skipping write to file task for CA key in TlsTrust = " + tr.Name)
        } else {
            if err := writeKeyFile(); err != nil { return err }
        }
        return nil
    }
    // if enabled, save the certificate and key in local files
    crtPath := tr.BaseDir + "/" + tr.Crt.File
    crtExists := TlsFileExists(crtPath)
    keyPath := tr.BaseDir + "/" + tr.Key.File
    keyExists := TlsFileExists(keyPath)
    switch {
    case !crtExists && !keyExists:
        log.Debugf("TLS certificate and key do not yet exist at expected file paths : %s : %s", crtPath, keyPath)
        if tr.Provision.CreateIfAbsent {
            if err := mkMyFiles(); err != nil { return err }
        } else {
            log.Debug("not writing TLS data to files as CreateIfAbsent is (bool) false")
        }
    case crtExists && keyExists:
	log.Debugf("TLS certificate and key already exist at expected file paths : %s : %s", crtPath, keyPath)
        if tr.Provision.ForceRecreate {
            log.Debugf("re-creating TLS certificate and key : %s : %s", crtPath, keyPath)
            TlsFileDelete(tr.Crt.File)
            TlsFileDelete(tr.Key.File)
            if err := mkMyFiles(); err != nil { return err }
        } else {
            log.Debug("not writing TLS data to files as ForceRecreate is (bool) false")
        }
    case crtExists && !keyExists:
	log.Debugf("TLS certificate already exists at expected file path : %s", crtPath)
	log.Debugf("TLS key does not exist at expected file path : %s", keyPath)
        if tr.Provision.ForceRecreate {
            log.Debugf("re-creating TLS certificate and key : %s : %s", crtPath, keyPath)
            TlsFileDelete(tr.Crt.File)
            TlsFileDelete(tr.Key.File)
            if err := mkMyFiles(); err != nil { return err }
        } else {
            log.Debug("not writing TLS data to files as ForceRecreate is (bool) false")
        }
    default:
        log.Error("unsupported condition for TLS certificate or key ; only one exists but the pair should be either present or absent")
        if tr.Provision.ForceRecreate {
            log.Debugf("purging and re-creating TLS certificate and key : %s : %s", crtPath, keyPath)
            TlsFileDelete(tr.Crt.File)
            TlsFileDelete(tr.Key.File)
            if err := mkMyFiles(); err != nil { return err }
        } else {
            log.Debug("not writing TLS data to clean files as ForceRecreate is (bool) false")
        }
    }

    return nil
}


// creates a new TlsTrustDomain for a given signing CA
func NewTlsTrustDomain(config *configure.EnvoySecret) (*TlsTrustDomain, error) {
    var (
        ca  *TlsTrust
        err error
    )
    if config.Type != configure.SecretTypeTlsCa {
        return nil, errors.New("cannot create NewTlsTrustDomain for secret type = " + config.Type)
    }
    // create new TlsTrust for to hold data for CA certificate and key
    if ca, err = NewTlsTrust(config); err != nil {
	return nil, errors.New("failed to create NewTlsTrust for CA secret = " + config.Name)
    }
    // write CA certificate to local file, if configured
    if err := ca.WriteTrustFiles(); err != nil {
        return nil, err
    }
    // return a pointer to the new TlsTrustDomain struct
    return &TlsTrustDomain{
        Name: config.Name,
        CA: ca,
    }, nil
}

/*
    methods for TlsTrustDomain
*/
// gets a buffer containing bytes for CA certificate
func (td *TlsTrustDomain) GetCaBytes() []byte {
    return td.CA.Crt.PEM
}

// gets a buffer containing bytes for server certificate and key
func (td *TlsTrustDomain) GetClientBytes(element int) (string, []byte, []byte, error) {
    switch {
    case element < len(td.Clients):
        log.Debugf("getting Client PEM bytes from existing element %d in TlsTrustDomain Clients", element)
	return td.Clients[element].Name, td.Clients[element].Crt.PEM, td.Clients[element].Key.PEM, nil
    default:
        return "", nil, nil, errors.New("invalid GetClientBytes request for array element " + string(element))
    }
}

// gets a buffer containing bytes for server certificate and key
func (td *TlsTrustDomain) GetServerBytes(element int) (string, []byte, []byte, error) {
    switch {
    case element < len(td.Servers):
        log.Debugf("getting Server PEM bytes from existing element %d in TlsTrustDomain Servers", element)
	return td.Servers[element].Name, td.Servers[element].Crt.PEM, td.Servers[element].Key.PEM, nil
    default:
        return "", nil, nil, errors.New("invalid GetServerBytes request for array element " + string(element))
    }
}

// creates a new TLS CA certificate and key
func (td *TlsTrustDomain) NewTlsCa(config *configure.EnvoySecret) error {
    // set up our server certificate
    caCrt := &x509.Certificate{
        SerialNumber: big.NewInt(2019),
        Subject: pkix.Name{
            Organization:  []string{config.Crt.Organization},
            Country:       []string{config.Crt.Country},
            Province:      []string{config.Crt.Province},
            Locality:      []string{config.Crt.Locality},
            StreetAddress: []string{config.Crt.StreetAddress},
            PostalCode:    []string{config.Crt.PostalCode},
        },
        NotBefore:             time.Now(),
        NotAfter:              time.Now().AddDate(10, 0, 0),
        IsCA:                  true,
        ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
        KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
        BasicConstraintsValid: true,
    }

    // create the CA private key
    caKey, err := rsa.GenerateKey(rand.Reader, 4096)
    if err != nil { return err }
    // create the CA certificate
    caBytes, err := x509.CreateCertificate(rand.Reader, caCrt, caCrt, &caKey.PublicKey, caKey)
    if err != nil { return err }
    // PEM encode the CA certificate
    caPem := TlsEncodeCrt(caBytes)
    // PEM encode the CA private key
    caKeyPem := TlsEncodePrivKey(caKey)
    // Store the CA data as attributes of the TlsTrustDomain
    td.CA.Crt.X509 = caCrt
    td.CA.Crt.PEM = caPem
    td.CA.Key.Key = caKey
    td.CA.Key.PEM = caKeyPem
    return nil
}

// creates a new TLS Client PEM certificate by signing with CA key from *TlsTrustDomain
func (td *TlsTrustDomain) NewTlsClient(config *configure.EnvoySecret) error {
    var (
        trust  *TlsTrust
        verify error
    )
    if trust, verify = NewTlsTrust(config); verify != nil {
	return verify
    }
    // set up our Client certificate
    crtClient := &x509.Certificate{
        SerialNumber: big.NewInt(2019),
        Subject: pkix.Name{
            Organization:  []string{trust.Crt.Config.Organization},
            Country:       []string{trust.Crt.Config.Country},
            Province:      []string{trust.Crt.Config.Province},
            Locality:      []string{trust.Crt.Config.Locality},
            StreetAddress: []string{trust.Crt.Config.StreetAddress},
            PostalCode:    []string{trust.Crt.Config.PostalCode},
        },
	DNSNames:     trust.Crt.Config.DNSNames,
        IPAddresses:  trust.Crt.Config.IPAddresses,
        NotBefore:    time.Now(),
        NotAfter:     time.Now().AddDate(10, 0, 0),
        SubjectKeyId: []byte{1, 2, 3, 4, 6},
        ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
        KeyUsage:     x509.KeyUsageDigitalSignature,
    }

    // create the Client private key
    privKey, err := rsa.GenerateKey(rand.Reader, 4096)
    if err != nil { return err }

    // create the Client certificate by signing with CA key
    crtBytes, err := x509.CreateCertificate(rand.Reader, crtClient, td.CA.Crt.X509, &privKey.PublicKey, td.CA.Key.Key)
    if err != nil { return err }

    // store the Client certificate and key data as attributes of the TlsTrustDomain's list of Clients
    // PEM encode the Client certificate
    trust.Crt.X509 = crtClient
    trust.Crt.PEM = TlsEncodeCrt(crtBytes)
    // PEM encode the Client private key
    trust.Key.Key = privKey
    trust.Key.PEM = TlsEncodePrivKey(privKey)

    // write the PEM-formatted certificate and key data to separate files, if enabled
    if err := trust.WriteTrustFiles(); err != nil { return err }

    // append the complete TlsTrust to the list of Clients in the *TlsTrustDomain
    td.Clients = append(td.Clients, trust)

    return nil
}

// creates a new TLS Server PEM certificate by signing with CA key from *TlsTrustDomain
func (td *TlsTrustDomain) NewTlsServer(config *configure.EnvoySecret) error {
    var (
        trust  *TlsTrust
        verify error
    )
    if trust, verify = NewTlsTrust(config); verify != nil {
	return verify
    }
    // set up our Server certificate
    crtServer := &x509.Certificate{
        SerialNumber: big.NewInt(2019),
        Subject: pkix.Name{
            Organization:  []string{trust.Crt.Config.Organization},
            Country:       []string{trust.Crt.Config.Country},
            Province:      []string{trust.Crt.Config.Province},
            Locality:      []string{trust.Crt.Config.Locality},
            StreetAddress: []string{trust.Crt.Config.StreetAddress},
            PostalCode:    []string{trust.Crt.Config.PostalCode},
        },
	DNSNames:     trust.Crt.Config.DNSNames,
        IPAddresses:  trust.Crt.Config.IPAddresses,
        NotBefore:    time.Now(),
        NotAfter:     time.Now().AddDate(10, 0, 0),
        SubjectKeyId: []byte{1, 2, 3, 4, 6},
        ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
        KeyUsage:     x509.KeyUsageDigitalSignature,
    }

    // create the Server private key
    privKey, err := rsa.GenerateKey(rand.Reader, 4096)
    if err != nil { return err }

    // create the Server certificate by signing with CA key
    crtBytes, err := x509.CreateCertificate(rand.Reader, crtServer, td.CA.Crt.X509, &privKey.PublicKey, td.CA.Key.Key)
    if err != nil { return err }

    // store the Server certificate and key data as attributes of the TlsTrustDomain's list of Servers
    // PEM encode the Server certificate
    trust.Crt.X509 = crtServer
    trust.Crt.PEM = TlsEncodeCrt(crtBytes)
    // PEM encode the Server private key
    trust.Key.Key = privKey
    trust.Key.PEM = TlsEncodePrivKey(privKey)

    // write the PEM-formatted certificate and key data to separate files, if enabled
    if err := trust.WriteTrustFiles(); err != nil { return err }

    // append the complete TlsTrust to the list of Servers in the *TlsTrustDomain
    td.Servers = append(td.Servers, trust)

    return nil
}
