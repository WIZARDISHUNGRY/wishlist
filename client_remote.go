package wishlist

import (
	"fmt"
	"io"
	"log"

	"github.com/gliderlabs/ssh"
	"github.com/muesli/termenv"
	gossh "golang.org/x/crypto/ssh"
)

type remoteClient struct {
	session ssh.Session
	stdin   io.Reader
}

func (c *remoteClient) Connect(e *Endpoint) error {
	resetPty(c.session)

	method, closers, err := remoteBestAuthMethod(c.session)
	defer closers.close()
	if err != nil {
		return fmt.Errorf("failed to find an auth method: %w", err)
	}

	conf := &gossh.ClientConfig{
		User:            firstNonEmpty(e.User, c.session.User()),
		HostKeyCallback: hostKeyCallback(e, ".wishlist/known_hosts"),
		Auth:            []gossh.AuthMethod{method},
	}

	session, _, cl, err := createSession(conf, e)
	defer cl.close()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	log.Printf("%s connect to %q, %s", c.session.User(), e.Name, c.session.RemoteAddr().String())

	session.Stdout = c.session
	session.Stderr = c.session.Stderr()
	session.Stdin = c.stdin

	pty, winch, _ := c.session.Pty()
	w := pty.Window
	if err := session.RequestPty(pty.Term, w.Height, w.Width, nil); err != nil {
		return fmt.Errorf("failed to request pty: %w", err)
	}

	done := make(chan bool, 1)
	defer func() { done <- true }()

	go c.notifyWindowChanges(session, done, winch)

	return shellAndWait(session)
}

func (c *remoteClient) notifyWindowChanges(session *gossh.Session, done <-chan bool, winch <-chan ssh.Window) {
	for {
		select {
		case <-done:
			return
		case w := <-winch:
			if w.Height == 0 && w.Width == 0 {
				// this only happens if the session is already dead, make sure there are no leftovers
				return
			}
			if err := session.WindowChange(w.Height, w.Width); err != nil {
				log.Println("failed to notify window change:", err)
				return
			}
		}
	}
}

func resetPty(w io.Writer) {
	fmt.Fprint(w, termenv.CSI+termenv.ExitAltScreenSeq)
	fmt.Fprint(w, termenv.CSI+termenv.ResetSeq+"m")
	fmt.Fprintf(w, termenv.CSI+termenv.EraseDisplaySeq, 2) // nolint:gomnd
}