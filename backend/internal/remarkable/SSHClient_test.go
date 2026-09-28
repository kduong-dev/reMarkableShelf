package remarkable_test

import (
	"bytes"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ssh"

	"github.com/kduong-dev/reMarkableShelf/backend/internal/remarkable"
)

// fakeTablet is an SSH server standing in for a tablet: it identifies
// itself with hostKey, accepts one password (counting every attempt),
// accepts any key listed in home/.ssh/authorized_keys, and runs exec
// requests with sh in home.
type fakeTablet struct {
	home             string
	password         string
	host             string
	port             string
	hostKey          string
	passwordAttempts int
}

func startFakeTablet(t *testing.T, password string) *fakeTablet {
	tablet := &fakeTablet{home: t.TempDir(), password: password}
	hostKey, err := remarkable.LoadOrCreateSigner(filepath.Join(t.TempDir(), "host_key"))
	So(err, ShouldBeNil)
	tablet.hostKey = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(hostKey.PublicKey())))
	serverConfig := &ssh.ServerConfig{
		PasswordCallback: func(_ ssh.ConnMetadata, attempt []byte) (*ssh.Permissions, error) {
			tablet.passwordAttempts++
			if string(attempt) != tablet.password {
				return nil, errors.New("wrong password")
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
			return nil, errors.New("unknown key")
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
	return newClientFor(t, port, "")
}

func newClientFor(t *testing.T, port, documentsDir string) *remarkable.SSHClient {
	signer, err := remarkable.LoadOrCreateSigner(filepath.Join(t.TempDir(), "id_ed25519"))
	So(err, ShouldBeNil)
	return remarkable.NewSSHClient(remarkable.Config{User: "root", Port: port, Signer: signer, DocumentsDir: documentsDir})
}

func writeFile(path, contents string) {
	So(os.MkdirAll(filepath.Dir(path), 0o755), ShouldBeNil)
	So(os.WriteFile(path, []byte(contents), 0o644), ShouldBeNil)
}

func TestPair(t *testing.T) {
	Convey("Given a tablet with password secret", t, func() {
		tablet := startFakeTablet(t, "secret")
		client := newClient(t, tablet.port)
		Convey("When pairing with the right password", func() {
			_, err := client.Pair(remarkable.Tablet{Host: tablet.host}, "secret")
			Convey("Then the server's key is installed once, labelled, in a private ~/.ssh", func() {
				So(err, ShouldBeNil)
				So(tablet.authorizedKeys(), ShouldContainSubstring, "remarkable-shelf")
				info, err := os.Stat(filepath.Join(tablet.home, ".ssh"))
				So(err, ShouldBeNil)
				So(info.Mode().Perm(), ShouldEqual, os.FileMode(0o700))
			})
			Convey("Then pairing again doesn't add the key twice", func() {
				_, err = client.Pair(remarkable.Tablet{Host: tablet.host}, "secret")
				So(err, ShouldBeNil)
				So(bytes.Count([]byte(tablet.authorizedKeys()), []byte("remarkable-shelf")), ShouldEqual, 1)
			})
			Convey("Then the key alone gets past sign-in when listing documents", func() {
				tablet.password = "changed"
				_, err := client.ListDocuments(remarkable.Tablet{Host: tablet.host})
				So(errors.Is(err, remarkable.ErrNotPaired), ShouldBeFalse)
				So(errors.Is(err, remarkable.ErrTabletUnreachable), ShouldBeFalse)
			})
		})
		Convey("When pairing with the wrong password", func() {
			_, err := client.Pair(remarkable.Tablet{Host: tablet.host}, "guess")
			Convey("Then it returns ErrWrongPassword as a client error and installs nothing", func() {
				So(errors.Is(err, remarkable.ErrWrongPassword), ShouldBeTrue)
				So(tablet.authorizedKeys(), ShouldBeEmpty)
			})
		})
		Convey("When listing documents without having paired", func() {
			_, err := client.ListDocuments(remarkable.Tablet{Host: tablet.host})
			Convey("Then it returns ErrNotPaired", func() {
				So(errors.Is(err, remarkable.ErrNotPaired), ShouldBeTrue)
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
			_, err := client.ListDocuments(remarkable.Tablet{Host: host})
			Convey("Then it returns ErrTabletUnreachable", func() {
				So(errors.Is(err, remarkable.ErrTabletUnreachable), ShouldBeTrue)
			})
		})
	})
}

func TestHostKeyPinning(t *testing.T) {
	Convey("Given a tablet paired with the server", t, func() {
		tablet := startFakeTablet(t, "secret")
		client := newClient(t, tablet.port)
		hostKey, err := client.Pair(remarkable.Tablet{Host: tablet.host}, "secret")
		So(err, ShouldBeNil)
		Convey("Then pairing returns the host key the tablet presented", func() {
			So(hostKey, ShouldEqual, tablet.hostKey)
		})
		Convey("When connecting with that host key recorded", func() {
			_, err := client.ListDocuments(remarkable.Tablet{Host: tablet.host, HostKey: hostKey})
			Convey("Then the tablet is accepted", func() {
				So(errors.Is(err, remarkable.ErrHostKeyChanged), ShouldBeFalse)
				So(errors.Is(err, remarkable.ErrNotPaired), ShouldBeFalse)
			})
		})
		Convey("And another machine answers at the address with a different host key", func() {
			impostor := startFakeTablet(t, "secret")
			impostorClient := newClient(t, impostor.port)
			pinned := remarkable.Tablet{Host: impostor.host, HostKey: hostKey}
			Convey("When syncing", func() {
				_, err := impostorClient.ListDocuments(pinned)
				Convey("Then it returns ErrHostKeyChanged", func() {
					So(errors.Is(err, remarkable.ErrHostKeyChanged), ShouldBeTrue)
				})
			})
			Convey("When pairing again with the password", func() {
				_, err := impostorClient.Pair(pinned, "secret")
				Convey("Then it refuses before the password is ever sent", func() {
					So(errors.Is(err, remarkable.ErrHostKeyChanged), ShouldBeTrue)
					So(impostor.passwordAttempts, ShouldEqual, 0)
				})
			})
		})
	})
}

func TestDocumentsAndCovers(t *testing.T) {
	Convey("Given a paired tablet holding a PDF whose first page has a thumbnail", t, func() {
		tablet := startFakeTablet(t, "secret")
		documentsDir := filepath.Join(tablet.home, "xochitl")
		uuid := "086daeda-1412-40d6-9bad-7aa1f9968857"
		writeFile(filepath.Join(documentsDir, uuid+".metadata"), `{"type": "DocumentType", "visibleName": "Dale Carnegie - How to Win Friends.pdf", "lastOpened": "0"}`)
		writeFile(filepath.Join(documentsDir, uuid+".content"), `{"fileType": "pdf", "coverPageNumber": 0, "pageCount": 2, "pages": ["f3df5538-f65e-44f2-a83a-01f4cd91289f", "39960184-8156-40ff-a4a9-110d3e2f36ef"]}`)
		writeFile(filepath.Join(documentsDir, uuid+".thumbnails", "f3df5538-f65e-44f2-a83a-01f4cd91289f.png"), "\x89PNG cover bytes")
		client := newClientFor(t, tablet.port, documentsDir)
		_, err := client.Pair(remarkable.Tablet{Host: tablet.host}, "secret")
		So(err, ShouldBeNil)
		Convey("When listing its documents", func() {
			listing, err := client.ListDocuments(remarkable.Tablet{Host: tablet.host})
			Convey("Then the PDF's cover page is its first page", func() {
				So(err, ShouldBeNil)
				So(listing.Documents, ShouldHaveLength, 1)
				So(listing.Documents[0].CoverPageID, ShouldEqual, "f3df5538-f65e-44f2-a83a-01f4cd91289f")
			})
		})
		Convey("When fetching its cover and one for a page with no thumbnail", func() {
			images, err := client.CoverImages(remarkable.Tablet{Host: tablet.host}, []remarkable.CoverRequest{
				{DocumentUUID: uuid, PageID: "f3df5538-f65e-44f2-a83a-01f4cd91289f"},
				{DocumentUUID: "5b1c0000-0000-0000-0000-000000000000", PageID: "a1b2c3d4-0000-0000-0000-000000000000"},
			})
			Convey("Then it returns the thumbnail's bytes and leaves out the missing one", func() {
				So(err, ShouldBeNil)
				So(string(images[uuid]), ShouldEqual, "\x89PNG cover bytes")
				So(images, ShouldHaveLength, 1)
			})
		})
		Convey("When a cover request names a path outside the thumbnails", func() {
			images, err := client.CoverImages(remarkable.Tablet{Host: tablet.host}, []remarkable.CoverRequest{
				{DocumentUUID: uuid, PageID: "../../../etc/passwd"},
			})
			Convey("Then it's skipped without reaching the shell", func() {
				So(err, ShouldBeNil)
				So(images, ShouldBeEmpty)
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
