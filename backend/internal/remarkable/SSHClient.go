package remarkable

import (
	"bytes"
	"fmt"
	"net"
	"regexp"
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
	if config.DocumentsDir == "" {
		config.DocumentsDir = DefaultDocumentsDir
	}
	return &SSHClient{config: config}
}

// idPattern matches the UUIDs the tablet names documents and pages with,
// which cover paths are built from, so nothing else reaches the shell.
var idPattern = regexp.MustCompile(`^[0-9a-f-]+$`)

func (client *SSHClient) CoverImages(tablet Tablet, requests []CoverRequest) (map[string][]byte, error) {
	images := make(map[string][]byte, len(requests))
	if len(requests) == 0 {
		return images, nil
	}
	connection, _, err := client.dialKey(tablet)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	for _, request := range requests {
		if !idPattern.MatchString(request.DocumentUUID) || !idPattern.MatchString(request.PageID) {
			continue
		}
		path := fmt.Sprintf("%s/%s.thumbnails/%s.png", client.config.DocumentsDir, request.DocumentUUID, request.PageID)
		output, err := run(connection, fmt.Sprintf("cat '%s'", path))
		if err != nil {
			continue // not rendered yet; the next sync tries again
		}
		images[request.DocumentUUID] = []byte(output)
	}
	return images, nil
}

// installKeyScript adds a key line to authorized_keys unless it's already
// there. umask 077 keeps ~/.ssh private, which the tablet's SSH server
// requires before it trusts the file.
const installKeyScript = `umask 077 && mkdir -p ~/.ssh && touch ~/.ssh/authorized_keys && ` +
	`(grep -qxF '%[1]s' ~/.ssh/authorized_keys || echo '%[1]s' >> ~/.ssh/authorized_keys)`

// Pair checks the tablet's host key before offering the password, since SSH
// verifies the server before authenticating, so an impostor never sees it.
// The key check then pins the host key the password connection saw.
func (client *SSHClient) Pair(tablet Tablet, password string) (string, error) {
	connection, hostKey, err := client.dial(tablet, ssh.Password(password), ErrWrongPassword)
	if err != nil {
		return "", err
	}
	defer connection.Close()
	if _, err := run(connection, fmt.Sprintf(installKeyScript, authorizedKey(client.config.Signer))); err != nil {
		return "", merry.Wrap(err).WithUserMessage("installing this server's key on the tablet")
	}
	verified, _, err := client.dial(Tablet{Host: tablet.Host, HostKey: hostKey}, ssh.PublicKeys(client.config.Signer), ErrKeyNotAccepted)
	if err != nil {
		return "", err
	}
	return hostKey, verified.Close()
}

// dialKey signs in to the tablet with the server's key, reporting a
// rejected key as ErrNotPaired.
func (client *SSHClient) dialKey(tablet Tablet) (*ssh.Client, string, error) {
	return client.dial(tablet, ssh.PublicKeys(client.config.Signer), ErrNotPaired)
}

// dial connects to the tablet with auth, returning the host key it
// presented. It reports a host key other than tablet.HostKey as
// ErrHostKeyChanged, a failed sign-in as authSentinel, and any other failure
// as ErrTabletUnreachable. x/crypto/ssh reports a failed sign-in only
// through its message.
func (client *SSHClient) dial(tablet Tablet, auth ssh.AuthMethod, authSentinel error) (*ssh.Client, string, error) {
	addr := fmt.Sprintf("%s:%s", tablet.Host, client.config.Port)
	var presented string
	hostKeyChanged := false
	connection, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{
		User: client.config.User,
		Auth: []ssh.AuthMethod{auth},
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			presented = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
			if tablet.HostKey != "" && !sameKey(tablet.HostKey, key) {
				hostKeyChanged = true
				return ErrHostKeyChanged
			}
			return nil
		},
		Timeout: 10 * time.Second,
	})
	if err == nil {
		return connection, presented, nil
	}
	var sentinel error = ErrTabletUnreachable
	switch {
	case hostKeyChanged:
		sentinel = ErrHostKeyChanged
	case strings.Contains(err.Error(), "unable to authenticate"):
		sentinel = authSentinel
	}
	return nil, "", merry.WithCause(merry.Here(sentinel).WithUserMessagef("%s (%s)", merry.UserMessage(sentinel), addr), err)
}

// sameKey reports whether key is the one recorded as authorized, a line in
// authorized_keys format.
func sameKey(authorized string, key ssh.PublicKey) bool {
	recorded, _, _, _, err := ssh.ParseAuthorizedKey([]byte(authorized))
	return err == nil && bytes.Equal(recorded.Marshal(), key.Marshal())
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
