package git

import (
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
)

type Remote string

func (r Remote) RemoteURL(c *Client) (string, error) {
	config, err := LoadLocalConfig(c)
	if err != nil {
		return "", err
	}
	if strings.Index(r.String(), "://") != -1 {
		// It's already a URL
		return string(r), nil
	}
	if File(r.String()).Exists() {
		// It's a known file path, so convert it to a file url
		// and let localConn handle it.
		// It needs to be absolute for the file:// url to work.
		abs, err := filepath.Abs(string(r))
		if err != nil {
			return "", err
		}
		return "file://" + abs, nil
	}
	// If it might be a remote name, look it up in the config.
	cfg, _ := config.GetConfig(fmt.Sprintf("remote.%v.url", r))
	if cfg == "" {
		return "", fmt.Errorf("Unknown remote")
	}
	return cfg, nil
}

func (r Remote) String() string {
	return string(r)
}

func (r Remote) IsStateless(c *Client) (bool, error) {
	url, err := r.RemoteURL(c)
	if err != nil {
		return false, err
	}
	url = strings.ToLower(url)
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://"), nil
}

// A RemoteConn represends a connection to a remote which communicates
// with the remote.
type RemoteConn interface {
	// Opens a connection to the remote. This requires at least one round
	// trip to the service and may mutate the state of this RemoteConn.
	//
	// After calling this, the RemoteConn should be in a useable state.
	OpenConn() error

	// Gets a list of references on the remote. If patterns is specified,
	// restrict to refs which match the pattern. If not, return all
	// refs
	GetRefs(opts LsRemoteOptions, patterns []string) ([]Ref, error)

	// Close the underlying connection to this service.
	Close() error

	// Sets the name of git-upload-pack to use for this remote, where
	// applicable. This must be called before OpenConn.
	// When called on a transport type that does not support it (such
	// as the git transport protocol), it will return a nil error. An
	// error indicates that the protocol *should* support the operation
	// but was unable to set the variable.
	SetUploadPack(string) error

	// Gets the protocol version that was negotiated during connection
	// opening. Only valid after calling OpenConn.
	ProtocolVersion() uint8

	// Returns the capabilities determined during the initial protocol
	// connection.
	//
	// The first index is the capability, the second is the arguments
	// defined for it.
	Capabilities() map[string]map[string]struct{}

	// Tells the connection to print any sideband data to w
	SetSideband(w io.Writer)

	// A RemoteConn should act as a writter. When written to, it should
	// write to the underlying connection in pkt-line format.
	io.Writer

	// Reading from a RemoteConn should return the data after decoding
	// the line length from a pktline.
	// The behaviour of the read depends on the PackProtocolMode set
	// by SetReadMode
	io.Reader

	// Determines how reading from the connection returns data to the
	// caller.
	SetReadMode(mode PackProtocolMode)

	// Send a flush packet to the connection
	Flush() error

	// Sends a Delimiter packet in protocol V2
	Delim() error
}

func NewRemoteConn(c *Client, r Remote) (RemoteConn, error) {
	urls, err := r.RemoteURL(c)
	if err != nil {
		return nil, err
	}
	uri, err := url.Parse(urls)
	if err != nil {
		return nil, err
	}
	switch uri.Scheme {
	case "http", "https":
		conn := &smartHTTPConn{
			sharedRemoteConn: sharedRemoteConn{uri: uri},
			giturl:           urls,
		}
		return conn, nil
	case "git":
		conn := &gitConn{
			sharedRemoteConn: sharedRemoteConn{uri: uri},
		}
		return conn, nil
	case "ssh":
		return &sshConn{
			sharedRemoteConn: sharedRemoteConn{uri: uri},
			uploadpack:       "git-upload-pack",
		}, nil
	case "file":
		return &localConn{
			sharedRemoteConn: sharedRemoteConn{uri: uri},
			uploadpack:       "git-upload-pack",
		}, nil
	default:
		return nil, fmt.Errorf("Unsupported remote type for: %v", r)
	}
}

// Helper for implenting things which are shared across all RemoteConn
// implementations
type sharedRemoteConn struct {
	uri             *url.URL
	protocolversion uint8
	capabilities    map[string]map[string]struct{}

	// References advertised during opening of connection. Only valid
	// for protocol v1
	refs []Ref

	packProtocolReader
}

func (r sharedRemoteConn) Capabilities() map[string]map[string]struct{} {
	return r.capabilities
}

func (r sharedRemoteConn) ProtocolVersion() uint8 {
	return r.protocolversion
}