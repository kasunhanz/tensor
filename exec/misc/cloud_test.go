package misc

import (
	"testing"
	"github.com/pearsonappeng/tensor/models/common"
	"io/ioutil"
	"os"
	"github.com/pearsonappeng/tensor/util"
	"github.com/stretchr/testify/assert"
)

func TestRaxCredFile(t *testing.T) {
	assert := assert.New(t)
	c := common.Credential{
		Username: "test",
		Secret: "test",
	}

	expected := "#!/usr/bin/python\n[rackspace_cloud]" +
		"\nusername=" + c.Username +
		"\napi_key=" + c.Secret

	f, _ := raxCredFile(c)
	actual, _ := ioutil.ReadFile(f.Name())

	assert.Equal(expected, string(actual), "Create racspace credential has invalid content")

	info, _ := os.Stat(f.Name())

	assert.Equal(os.FileMode(0700), info.Mode(), "Rackspace file has incorrect permissions")

	os.Remove(f.Name())
}

func TestGCECredFile(t *testing.T) {
	assert := assert.New(t)
	c := common.Credential{
		Username: "test",
		SSHKeyData: util.CipherEncrypt("test"),
	}

	f, _ := GCECredFile(c)
	actual, _ := ioutil.ReadFile(f.Name())

	assert.Equal("test", string(actual), "Create GCE credential has invalid content")

	os.Remove(f.Name())
}

func TestGetCloudCredential(t *testing.T) {
	assert := assert.New(t)


	// Test AWS credential environment variables
	c := common.Credential{
		Secret: util.CipherEncrypt("test"),
		Client: "test",
		Kind: common.CredentialKindAWS,
	}

	actual, _, _ := GetCloudCredential([]string{}, c)
	expected := []string{"AWS_SECRET_ACCESS_KEY=test", "AWS_ACCESS_KEY_ID=test"}
	assert.Equal(expected, actual, "Must be equal")

	// Test Rackspace credentials
	c = common.Credential{
		Secret: util.CipherEncrypt("test"),
		Username: "test",
		Kind: common.CredentialKindRAX,
	}

	actual, f, _ := GetCloudCredential([]string{}, c)
	expected = []string{"RAX_CREDS_FILE="+ f.Name()}
	os.Remove(f.Name())

	assert.Equal(expected, actual, "Must be equal")

	// Test GCE credentials
	c = common.Credential{
		SSHKeyData: util.CipherEncrypt("test"),
		Email: "test",
		Project: "test",
		Kind: common.CredentialKindGCE,
	}

	actual, f, _ = GetCloudCredential([]string{}, c)
	expected = []string{"GCE_EMAIL=test", "GCE_PROJECT=test", "GCE_CREDENTIALS_FILE_PATH="+ f.Name()}
	os.Remove(f.Name())

	assert.Equal(expected, actual, "Must be equal")

	// Test AWS credential environment variables
	// Test Azure Active directory
	c = common.Credential{
		Username: "test",
		Password: util.CipherEncrypt("test"),
		Subscription: "test",
		Kind: common.CredentialKindAZURE,
	}

	actual, _, _ = GetCloudCredential([]string{}, c)
	expected = []string{"AZURE_AD_USER=test", "AZURE_PASSWORD=test", "AZURE_SUBSCRIPTION_ID=test"}
	assert.Equal(expected, actual, "Must be equal")

	// Test Azure service principle
	c = common.Credential{
		Client: "test",
		Secret: util.CipherEncrypt("test"),
		Subscription: "test",
		Tenant: "test",
		Kind: common.CredentialKindAZURE,
	}

	actual, _, _ = GetCloudCredential([]string{}, c)
	expected = []string{"AZURE_CLIENT_ID=test", "AZURE_SECRET=test", "AZURE_SUBSCRIPTION_ID=test", "AZURE_TENANT=test"}
	assert.Equal(expected, actual, "Must be equal")
}