package ssh

import (
	"bytes"
	"github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh/agent"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"github.com/ScaleFT/sshkeys"
)

// startAgent executes ssh-agent, and returns a Agent interface to it.
func StartAgent() (client agent.Agent, socket string, pid int, cleanup func()) {
	bin, err := exec.LookPath("ssh-agent")
	if err != nil {
		logrus.Errorln("could not find ssh-agent")
	}

	cmd := exec.Command(bin, "-s")
	out, err := cmd.Output()
	if err != nil {
		logrus.Errorf("cmd.Output: %v", err)
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
		logrus.Infof("could not find key SSH_AUTH_SOCK in %q", fields[0])
	}
	socket = string(line[1])

	line = bytes.SplitN(fields[2], []byte("="), 2)
	line[0] = bytes.TrimLeft(line[0], "\n")
	if string(line[0]) != "SSH_AGENT_PID" {
		logrus.Infof("could not find key SSH_AGENT_PID in %q", fields[2])
	}
	pidStr := line[1]
	pid, err = strconv.Atoi(string(pidStr))
	if err != nil {
		logrus.Infof("Atoi(%q): %v", pidStr, err)
	}

	conn, err := net.Dial("unix", string(socket))
	if err != nil {
		logrus.Infof("net.Dial: %v", err)
	}

	client = agent.NewClient(conn)
	cleanup = func() {
		proc, _ := os.FindProcess(pid)
		if proc != nil {
			proc.Kill()
		}
		conn.Close()
		os.RemoveAll(filepath.Dir(socket))
	}
	return
}

func GetKey(key []byte, secret []byte) (addedkey agent.AddedKey, err error) {
	addedkey = agent.AddedKey{}
	addedkey.PrivateKey, err = sshkeys.ParseEncryptedRawPrivateKey(key, secret)
	return
}