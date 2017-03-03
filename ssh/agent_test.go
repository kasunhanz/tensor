package ssh

// Do not refactor or reformat this code

import (
	"github.com/stretchr/testify/suite"
	"testing"
	"os"
)

type AgentTestSuite struct {
	suite.Suite
}
func (suite *AgentTestSuite) TestAgent() {
	client, socket, pid, clean := StartAgent()

	suite.NotEmpty(client, "Client should not empty")
	suite.NotEmpty(socket, "Socket should not empty")
	suite.NotEmpty(pid, "Pid should not empty")

	clean()
	_, err := os.Stat(socket)
	suite.Error(err, "Stat should return error")
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestAgentTestSuite(t *testing.T) {
	suite.Run(t, new(AgentTestSuite))
}
