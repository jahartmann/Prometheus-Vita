package proxmox

import (
	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/service/crypto"
)

type ClientFactory interface {
	CreateClient(node *model.Node) (*Client, error)
	CreateClientFromCredentials(hostname string, port int, tokenID, tokenSecret string) *Client
}

type DefaultClientFactory struct {
	encryptor *crypto.Encryptor
}

func NewClientFactory(encryptor *crypto.Encryptor) ClientFactory {
	return &DefaultClientFactory{encryptor: encryptor}
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
	return NewClient(node.Hostname, node.Port, tokenID, tokenSecret), nil
}

func (f *DefaultClientFactory) CreateClientFromCredentials(hostname string, port int, tokenID, tokenSecret string) *Client {
	return NewClient(hostname, port, tokenID, tokenSecret)
}
