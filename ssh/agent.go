package ssh

import (
	"bytes"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"golang.org/x/crypto/ssh/agent"
	log "github.com/Sirupsen/logrus"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"crypto"
	"crypto/rsa"
	"crypto/ecdsa"
)

// startAgent executes ssh-agent, and returns a Agent interface to it.
func StartAgent() (client agent.Agent, socket string, pid int, cleanup func()) {
	bin, err := exec.LookPath("ssh-agent")
	if err != nil {
		log.Errorln("could not find ssh-agent")
	}

	cmd := exec.Command(bin, "-s")
	out, err := cmd.Output()
	if err != nil {
		log.Errorf("cmd.Output: %v", err)
	}

	/* Output looks like:
		   SSH_AUTH_SOCK=/tmp/ssh-P65gpcqArqvH/agent.15541; export SSH_AUTH_SOCK;
	           SSH_AGENT_PID=15542; export SSH_AGENT_PID;
	           echo Agent pid 15542;
	*/
	fields := bytes.Split(out, []byte(";"))
	line := bytes.SplitN(fields[0], []byte("="), 2)
	line[0] = bytes.TrimLeft(line[0], "\n")
	if string(line[0]) != "SSH_AUTH_SOCK" {
		log.Infof("could not find key SSH_AUTH_SOCK in %q", fields[0])
	}
	socket = string(line[1])

	line = bytes.SplitN(fields[2], []byte("="), 2)
	line[0] = bytes.TrimLeft(line[0], "\n")
	if string(line[0]) != "SSH_AGENT_PID" {
		log.Infof("could not find key SSH_AGENT_PID in %q", fields[2])
	}
	pidStr := line[1]
	pid, err = strconv.Atoi(string(pidStr))
	if err != nil {
		log.Infof("Atoi(%q): %v", pidStr, err)
	}

	conn, err := net.Dial("unix", string(socket))
	if err != nil {
		log.Infof("net.Dial: %v", err)
	}

	ac := agent.NewClient(conn)
	return ac, socket, pid, func() {
		proc, _ := os.FindProcess(pid)
		if proc != nil {
			proc.Kill()
		}
		conn.Close()
		os.RemoveAll(filepath.Dir(socket))
	}
}

// Attempt to parse the given private key DER block. OpenSSL 0.9.8 generates
// PKCS#1 private keys by default, while OpenSSL 1.0.0 generates PKCS#8 keys.
// OpenSSL ecparam generates SEC1 EC private keys for ECDSA. We try all three.
func parsePrivateKey(der []byte) (crypto.PrivateKey, error) {
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	} else {
		log.Errorln(err)
	}

	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey, *ecdsa.PrivateKey:
			return key, nil
		default:
			return nil, errors.New("ssh/agent: found unknown private key type in PKCS#8 wrapping")
		}
	} else {
		log.Errorln(err)
	}

	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	} else {
		log.Errorln(err)
	}

	return nil, errors.New("ssh/agent: failed to parse private key")
}

func GetKey(data []byte) (key agent.AddedKey, err error) {
	block, _ := pem.Decode(data)
	key.PrivateKey, err = parsePrivateKey(block.Bytes)
	if err != nil {
		return
	}

	key.Comment = "Credential"
	key.ConfirmBeforeUse = false

	return
}

func GetEncryptedKey(data []byte, password string) (agent.AddedKey, error) {
	block, _ := pem.Decode(data)

	key := agent.AddedKey{}
	if block == nil {
		return key, errors.New("Error while decoding key")
	}

	if !x509.IsEncryptedPEMBlock(block) {
		return key, errors.New("Key is not PEM Encrypted")
	}

	// encrypted key, unencrypt using password
	der, err := x509.DecryptPEMBlock(block, []byte(password))
	if err != nil {
		log.Errorln("Error while decrypting key")
		return key, err
	}

	key.PrivateKey, err = parsePrivateKey(der)
	if err != nil {
		return key, err
	}

	key.Comment = "Credential"
	key.ConfirmBeforeUse = false

	return key, nil
}