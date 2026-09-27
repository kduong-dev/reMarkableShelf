package remarkable_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"testing"

	"github.com/ansel1/merry"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/remarkable"
)

// listenRejectingSSH starts an SSH server that turns down every password,
// returning its host and port.
func listenRejectingSSH(t *testing.T) (string, string) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	So(err, ShouldBeNil)
	signer, err := ssh.NewSignerFromKey(privateKey)
	So(err, ShouldBeNil)
	serverConfig := &ssh.ServerConfig{
		PasswordCallback: func(ssh.ConnMetadata, []byte) (*ssh.Permissions, error) {
			return nil, merry.New("wrong password")
		},
	}
	serverConfig.AddHostKey(signer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	So(err, ShouldBeNil)
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				_, _, _, _ = ssh.NewServerConn(connection, serverConfig)
				_ = connection.Close()
			}()
		}
	}()
	host, port, err := net.SplitHostPort(listener.Addr().String())
	So(err, ShouldBeNil)
	return host, port
}

func TestListDocumentsConnectionErrors(t *testing.T) {
	Convey("Given a tablet that rejects the SSH password", t, func() {
		host, port := listenRejectingSSH(t)
		client := remarkable.NewSSHClient(remarkable.Config{User: "root", Password: "CHANGEME", Port: port})
		Convey("When listing its documents", func() {
			_, err := client.ListDocuments(host)
			Convey("Then it returns ErrAuthenticationFailed pointing at SSH_PASSWORD", func() {
				So(merry.Is(err, remarkable.ErrAuthenticationFailed), ShouldBeTrue)
				So(merry.UserMessage(err), ShouldContainSubstring, "rejected the SSH password")
			})
		})
	})
	Convey("Given no tablet listening at the address", t, func() {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		So(err, ShouldBeNil)
		host, port, _ := net.SplitHostPort(listener.Addr().String())
		So(listener.Close(), ShouldBeNil)
		client := remarkable.NewSSHClient(remarkable.Config{User: "root", Password: "secret", Port: port})
		Convey("When listing its documents", func() {
			_, err := client.ListDocuments(host)
			Convey("Then it returns ErrTabletUnreachable", func() {
				So(merry.Is(err, remarkable.ErrTabletUnreachable), ShouldBeTrue)
				So(merry.UserMessage(err), ShouldContainSubstring, "may be asleep")
			})
		})
	})
}
