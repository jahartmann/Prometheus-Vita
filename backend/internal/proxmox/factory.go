package proxmox

import (
	"crypto/tls"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/service/crypto"
)

type ClientFactory interface {
	CreateClient(node *model.Node) (*Client, error)
	CreateClientFromCredentials(hostname string, port int, tokenID, tokenSecret string) *Client
	TLSConfig() *tls.Config
}

type DefaultClientFactory struct {
	encryptor *crypto.Encryptor
	tlsConfig *tls.Config
}

func NewClientFactory(encryptor *crypto.Encryptor, tlsConfig *tls.Config) ClientFactory {
	return &DefaultClientFactory{encryptor: encryptor, tlsConfig: tlsConfig}
}

func (f *DefaultClientFactory) CreateClient(node *model.Node) (*Client, error) {
	tokenID, err := f.encryptor.Decrypt(node.APITokenID)
	if err != nil {
		return nil, err
	}
	tokenSecret, err := f.encryptor.Decrypt(node.APITokenSecret)
	if err != nil {
		return nil, err
	}
	return NewClient(node.Hostname, node.Port, tokenID, tokenSecret, f.tlsConfig), nil
}

func (f *DefaultClientFactory) CreateClientFromCredentials(hostname string, port int, tokenID, tokenSecret string) *Client {
	return NewClient(hostname, port, tokenID, tokenSecret, f.tlsConfig)
}

// TLSConfig returns the TLS configuration used by this factory, or nil for insecure defaults.
func (f *DefaultClientFactory) TLSConfig() *tls.Config {
	return f.tlsConfig
}
