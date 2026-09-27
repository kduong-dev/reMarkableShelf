package remarkable_test

import (
	"bytes"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ansel1/merry"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/remarkable"
)

// fakeTablet is an SSH server standing in for a tablet: it accepts one
// password, accepts any key listed in home/.ssh/authorized_keys, and runs
// exec requests with sh in home.
type fakeTablet struct {
	home     string
	password string
	host     string
	port     string
}

func startFakeTablet(t *testing.T, password string) *fakeTablet {
	tablet := &fakeTablet{home: t.TempDir(), password: password}
	hostKey, err := remarkable.LoadOrCreateSigner(filepath.Join(t.TempDir(), "host_key"))
	So(err, ShouldBeNil)
	serverConfig := &ssh.ServerConfig{
		PasswordCallback: func(_ ssh.ConnMetadata, attempt []byte) (*ssh.Permissions, error) {
			if string(attempt) != tablet.password {
				return nil, merry.New("wrong password")
			}
			return nil, nil
		},
		PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			authorized, _ := os.ReadFile(filepath.Join(tablet.home, ".ssh", "authorized_keys"))
			for len(authorized) > 0 {
				listed, _, _, rest, err := ssh.ParseAuthorizedKey(authorized)
				if err != nil {
					break
				}
				if bytes.Equal(listed.Marshal(), key.Marshal()) {
					return nil, nil
				}
				authorized = rest
			}
			return nil, merry.New("unknown key")
		},
	}
	serverConfig.AddHostKey(hostKey)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	So(err, ShouldBeNil)
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			go tablet.serve(connection, serverConfig)
		}
	}()
	tablet.host, tablet.port, _ = net.SplitHostPort(listener.Addr().String())
	return tablet
}

func (tablet *fakeTablet) serve(connection net.Conn, config *ssh.ServerConfig) {
	defer connection.Close()
	_, channels, requests, err := ssh.NewServerConn(connection, config)
	if err != nil {
		return
	}
	go ssh.DiscardRequests(requests)
	for newChannel := range channels {
		channel, channelRequests, err := newChannel.Accept()
		if err != nil {
			continue
		}
		go func() {
			defer channel.Close()
			for request := range channelRequests {
				if request.Type != "exec" {
					_ = request.Reply(false, nil)
					continue
				}
				var payload struct{ Command string }
				_ = ssh.Unmarshal(request.Payload, &payload)
				_ = request.Reply(true, nil)
				command := exec.Command("sh", "-c", payload.Command)
				command.Env = []string{"HOME=" + tablet.home, "PATH=" + os.Getenv("PATH")}
				command.Dir = tablet.home
				command.Stdout, command.Stderr = channel, channel.Stderr()
				status := uint32(0)
				if err := command.Run(); err != nil {
					status = 1
				}
				_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{status}))
				return
			}
		}()
	}
}

func (tablet *fakeTablet) authorizedKeys() string {
	contents, _ := os.ReadFile(filepath.Join(tablet.home, ".ssh", "authorized_keys"))
	return string(contents)
}

func newClient(t *testing.T, port string) *remarkable.SSHClient {
	signer, err := remarkable.LoadOrCreateSigner(filepath.Join(t.TempDir(), "id_ed25519"))
	So(err, ShouldBeNil)
	return remarkable.NewSSHClient(remarkable.Config{User: "root", Port: port, Signer: signer})
}

func TestPair(t *testing.T) {
	Convey("Given a tablet with password secret", t, func() {
		tablet := startFakeTablet(t, "secret")
		client := newClient(t, tablet.port)
		Convey("When pairing with the right password", func() {
			err := client.Pair(tablet.host, "secret")
			Convey("Then the server's key is installed once, labelled, in a private ~/.ssh", func() {
				So(err, ShouldBeNil)
				So(tablet.authorizedKeys(), ShouldContainSubstring, "remarkable-shelf")
				info, err := os.Stat(filepath.Join(tablet.home, ".ssh"))
				So(err, ShouldBeNil)
				So(info.Mode().Perm(), ShouldEqual, os.FileMode(0o700))
			})
			Convey("Then pairing again doesn't add the key twice", func() {
				So(client.Pair(tablet.host, "secret"), ShouldBeNil)
				So(bytes.Count([]byte(tablet.authorizedKeys()), []byte("remarkable-shelf")), ShouldEqual, 1)
			})
			Convey("Then the key alone gets past sign-in when listing documents", func() {
				tablet.password = "changed"
				_, err := client.ListDocuments(tablet.host)
				So(merry.Is(err, remarkable.ErrNotPaired), ShouldBeFalse)
				So(merry.Is(err, remarkable.ErrTabletUnreachable), ShouldBeFalse)
			})
		})
		Convey("When pairing with the wrong password", func() {
			err := client.Pair(tablet.host, "guess")
			Convey("Then it returns ErrWrongPassword as a client error and installs nothing", func() {
				So(merry.Is(err, remarkable.ErrWrongPassword), ShouldBeTrue)
				So(merry.HTTPCode(err), ShouldEqual, 400)
				So(tablet.authorizedKeys(), ShouldBeEmpty)
			})
		})
		Convey("When listing documents without having paired", func() {
			_, err := client.ListDocuments(tablet.host)
			Convey("Then it returns ErrNotPaired", func() {
				So(merry.Is(err, remarkable.ErrNotPaired), ShouldBeTrue)
				So(merry.UserMessage(err), ShouldContainSubstring, "pair it again")
			})
		})
	})
	Convey("Given no tablet listening at the address", t, func() {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		So(err, ShouldBeNil)
		host, port, _ := net.SplitHostPort(listener.Addr().String())
		So(listener.Close(), ShouldBeNil)
		client := newClient(t, port)
		Convey("When listing its documents", func() {
			_, err := client.ListDocuments(host)
			Convey("Then it returns ErrTabletUnreachable", func() {
				So(merry.Is(err, remarkable.ErrTabletUnreachable), ShouldBeTrue)
				So(merry.UserMessage(err), ShouldContainSubstring, "may be asleep")
			})
		})
	})
}

func TestLoadOrCreateSigner(t *testing.T) {
	Convey("Given no key file yet", t, func() {
		path := filepath.Join(t.TempDir(), "keys", "id_ed25519")
		Convey("When loading the signer twice", func() {
			first, err := remarkable.LoadOrCreateSigner(path)
			So(err, ShouldBeNil)
			second, err := remarkable.LoadOrCreateSigner(path)
			So(err, ShouldBeNil)
			Convey("Then the first call creates a private key file the second reuses", func() {
				So(second.PublicKey().Marshal(), ShouldResemble, first.PublicKey().Marshal())
				info, err := os.Stat(path)
				So(err, ShouldBeNil)
				So(info.Mode().Perm(), ShouldEqual, os.FileMode(0o600))
			})
		})
	})
}
