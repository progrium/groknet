package groknet

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"os/user"
	"strings"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	Subdomain string // host tunnel on a custom subdomain
	Auth      string // enforce basic auth on tunnel endpoint, 'user:password'
	Hostname  string // host tunnel on custom hostname (requires DNS CNAME)
	Identity  string // path to private key for SSH auth (default: ~/.ssh/id_rsa)
	Region    string // ngrok region to use (default: us)
}

func Listen(config Config) (*Listener, error) {
	if config.Region == "" {
		config.Region = "us"
	}

	if config.Identity == "" {
		usr, _ := user.Current()
		dir := usr.HomeDir
		config.Identity = fmt.Sprintf("%s/.ssh/id_rsa", dir)
	}

	key, err := ioutil.ReadFile(config.Identity)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	c, err := ssh.Dial("tcp4", fmt.Sprintf("tunnel.%s.ngrok.com:22", config.Region), &ssh.ClientConfig{
		User: "",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return nil, err
	}
	client := &client{Client: c}

	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}

	r, w := io.Pipe()
	session.Stdout = w
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanLines)

	args := []string{"http"}
	if config.Subdomain != "" {
		args = append(args, fmt.Sprintf("-subdomain=%s", config.Subdomain))
	}
	if config.Hostname != "" {
		args = append(args, fmt.Sprintf("-hostname=%s", config.Hostname))
	}
	if config.Auth != "" {
		args = append(args, fmt.Sprintf("-auth=%s", config.Auth))
	}
	if err := session.Start(strings.Join(args, " ")); err != nil {
		return nil, err
	}

	nl, err := client.Listen("tcp", "127.0.0.1:443")
	if err != nil {
		return nil, err
	}
	l := &Listener{
		Listener: nl,
		Session:  session,
		Client:   c,
	}

	ready := make(chan bool)
	go func() {
		for scanner.Scan() {
			parts := strings.SplitN(scanner.Text(), " ", 2)
			switch parts[0] {
			case "Account":
				l.Account = strings.TrimSpace(parts[1])
			case "Region":
				l.Region = strings.TrimSpace(parts[1])
			case "Forwarding":
				u, err := url.Parse(strings.TrimSpace(parts[1]))
				if err != nil {
					log.Fatal(err)
				}
				switch u.Scheme {
				case "https":
					l.URL = u
				case "http":
					l.InsecureURL = u
				}
			}
			if l.URL != nil {
				ready <- true
			}
		}
	}()

	<-ready
	return l, nil
}

type Listener struct {
	net.Listener
	*ssh.Session
	*ssh.Client

	Account     string
	Region      string
	URL         *url.URL
	InsecureURL *url.URL
}

func (l *Listener) Close() error {
	if err := l.Listener.Close(); err != nil {
		return err
	}
	if err := l.Session.Close(); err != nil {
		return err
	}
	if err := l.Client.Close(); err != nil {
		return err
	}
	return nil
}
