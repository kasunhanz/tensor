package misc

import (
	"errors"
	"io/ioutil"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/pearsonappeng/tensor/models/common"
	"github.com/pearsonappeng/tensor/util"
)

// raxCredFile creates a Rackspace credential file in the system temporary directory
// and returns the resulting *os.File.
// Multiple programs calling raxCredFile simultaneously
// will not choose the same file. The caller can use f.Name()
// to find the pathname of the file. It is the caller's responsibility
// to remove the file when no longer needed.
func raxCredFile(c common.Credential) (f *os.File, err error) {
	content := "#!/usr/bin/python\n[rackspace_cloud]" +
		"\nusername=" + c.Username +
		"\napi_key=" + c.Secret

	f, err = ioutil.TempFile("", "tensor_credential_rackspace")
	if err != nil {
		logrus.Errorln("Rackspace credential file creation failed")
		return
	}

	if _, err = f.Write([]byte(content)); err != nil {
		logrus.Errorln("Rackspace credential file creation failed")
		return
	}
	if err = f.Close(); err != nil {
		return
	}

	// make the credential python file executable for the process user
	if err = os.Chmod(f.Name(), 0700); err != nil {
		return
	}

	return
}

// GCECredFile creates a Google Compute Engine credential file in the system
// temporary directory and returns the resulting *os.File.
// Multiple programs calling GCECredFile simultaneously
// will not choose the same file. The caller can use f.Name()
// to find the pathname of the file. It is the caller's responsibility
// to remove the file when no longer needed.
func GCECredFile(c common.Credential) (f *os.File, err error) {
	f, err = ioutil.TempFile("", "tensor_credential_gce")
	if err != nil {
		logrus.Errorln("GCE credential file creation failed")
		return
	}

	if _, err = f.Write([]byte(util.Decipher(c.SSHKeyData))); err != nil {
		logrus.Errorln("GCE credential file creation failed")
		return
	}
	if err = f.Close(); err != nil {
		return
	}

	return
}

// GetCloudCredential cloud credential files and generates environment variables,
// This accepts string slice and common.Credential (cloud credential) interface
// and returns slice of environment variables generated and file handler to the
// credential file
func GetCloudCredential(env []string, c common.Credential) (menv []string, f *os.File, err error) {
	switch c.Kind {
	//if Cloud Credential type is AWS
	case common.CredentialKindAWS:
		{
			// add environment variables for aws
			menv = append(env, "AWS_SECRET_ACCESS_KEY="+string(util.Decipher(c.Secret)),
				"AWS_ACCESS_KEY_ID="+c.Client)
		}
	case common.CredentialKindRAX:
		{
			f, err = raxCredFile(c)
			if err != nil {
				err = errors.New("Rackspace credential file creation failed")
				return
			}

			// add environment variables for Rackspace credential
			menv = append(env, "RAX_CREDS_FILE="+f.Name())
		}
	case common.CredentialKindGCE:
		{
			f, err = GCECredFile(c)
			if err != nil {
				err = errors.New("GCE credential file creation failed")
			}

			// add environment variables for GCE credential
			menv = append(env, "GCE_EMAIL="+c.Email, "GCE_PROJECT="+c.Project, "GCE_CREDENTIALS_FILE_PATH="+f.Name())
		}
	case common.CredentialKindAZURE:
		{
			// Azure Active Directory
			if len(c.Username) > 0 {
				// add environment variables for Azure active directory credential
				menv = append(env, "AZURE_AD_USER="+c.Username,
					"AZURE_PASSWORD="+string(util.Decipher(c.Password)),
					"AZURE_SUBSCRIPTION_ID="+c.Subscription)
			} else {
				// add environment variables for Azure service principle credential
				menv = append(env, "AZURE_CLIENT_ID="+c.Client,
					"AZURE_SECRET="+string(util.Decipher(c.Secret)),
					"AZURE_SUBSCRIPTION_ID="+c.Subscription,
					"AZURE_TENANT="+c.Tenant)
			}
		}
	}

	return
}
