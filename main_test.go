package main_test

import (
	"bytes"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"testing"
	"time"

	"github.com/nsqio/go-nsq"
	"github.com/stretchr/testify/suite"
)

type EndToEndTestSuite struct {
	suite.Suite
	topic    string
	consumer *nsq.Consumer
	messages chan *nsq.Message
	cmd      *exec.Cmd
}

func (suite *EndToEndTestSuite) SetupSuite() {
	var err error

	suite.messages = make(chan *nsq.Message)

	config := nsq.NewConfig()
	prefix := regexp.MustCompile(`\D`).ReplaceAllString(time.Now().Format(time.RFC3339), "")
	suite.topic = "test_" + prefix
	suite.consumer, err = nsq.NewConsumer(suite.topic, "channel"+prefix, config)
	suite.NoError(err)

	suite.consumer.AddHandler(nsq.HandlerFunc(func(m *nsq.Message) error {
		suite.messages <- m
		return nil
	}))

	err = suite.consumer.ConnectToNSQD("localhost:4150")
	suite.NoError(err)

	suite.cmd = exec.Command("./http_to_nsq")
	err = suite.cmd.Start()
	suite.NoError(err)

	// wait for the server to start
	time.Sleep(1 * time.Second)
}

func (suite *EndToEndTestSuite) TearDownSuite() {
	suite.consumer.Stop()
	suite.cmd.Process.Signal(os.Interrupt)
}

func (suite *EndToEndTestSuite) TestPostMessage() {
	resp, err := http.Post("http://localhost:4252/"+suite.topic, "text/plain", bytes.NewBufferString("message"))
	suite.NoError(err)
	suite.Require().Equal(http.StatusOK, resp.StatusCode)

	select {
	case m := <-suite.messages:
		suite.Equal("message", string(m.Body))
	case <-time.After(5 * time.Second):
		suite.Fail("timeout waiting for message")
	}
}

func TestEndToEndTestSuite(t *testing.T) {
	suite.Run(t, new(EndToEndTestSuite))
}
