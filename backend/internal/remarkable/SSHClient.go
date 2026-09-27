package remarkable

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/ansel1/merry"
	"golang.org/x/crypto/ssh"
)

// SSHClient reaches tablets over SSH, signing in with the server's key.
type SSHClient struct {
	config Config
}

func NewSSHClient(config Config) *SSHClient {
	return &SSHClient{config: config}
}

// installKeyScript adds a key line to authorized_keys unless it's already
// there. umask 077 keeps ~/.ssh private, which the tablet's SSH server
// requires before it trusts the file.
const installKeyScript = `umask 077 && mkdir -p ~/.ssh && touch ~/.ssh/authorized_keys && ` +
	`(grep -qxF '%[1]s' ~/.ssh/authorized_keys || echo '%[1]s' >> ~/.ssh/authorized_keys)`

func (client *SSHClient) Pair(host, password string) error {
	connection, err := client.dial(host, ssh.Password(password), ErrWrongPassword)
	if err != nil {
		return err
	}
	defer connection.Close()
	if _, err := run(connection, fmt.Sprintf(installKeyScript, authorizedKey(client.config.Signer))); err != nil {
		return merry.Wrap(err).WithUserMessage("installing this server's key on the tablet")
	}
	verified, err := client.dial(host, ssh.PublicKeys(client.config.Signer), ErrKeyNotAccepted)
	if err != nil {
		return err
	}
	return verified.Close()
}

// dialKey signs in to host with the server's key, reporting a rejected key
// as ErrNotPaired.
func (client *SSHClient) dialKey(host string) (*ssh.Client, error) {
	return client.dial(host, ssh.PublicKeys(client.config.Signer), ErrNotPaired)
}

// dial connects to host with auth, reporting a failed sign-in as
// authSentinel and any other failure as ErrTabletUnreachable. x/crypto/ssh
// reports a failed sign-in only through its message.
func (client *SSHClient) dial(host string, auth ssh.AuthMethod, authSentinel error) (*ssh.Client, error) {
	addr := fmt.Sprintf("%s:%s", host, client.config.Port)
	connection, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{
		User:            client.config.User,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	})
	if err == nil {
		return connection, nil
	}
	var sentinel error = ErrTabletUnreachable
	if strings.Contains(err.Error(), "unable to authenticate") {
		sentinel = authSentinel
	}
	return nil, merry.WithCause(merry.Here(sentinel).WithUserMessagef("%s (%s)", merry.UserMessage(sentinel), addr), err)
}

// run executes script in a new session and returns what it printed.
func run(connection *ssh.Client, script string) (string, error) {
	session, err := connection.NewSession()
	if err != nil {
		return "", merry.Wrap(err).WithUserMessage("opening SSH session")
	}
	defer session.Close()
	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	if err := session.Run(script); err != nil {
		return "", merry.Prependf(err, "running script (stderr: %s)", strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
