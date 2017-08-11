// This package implements classic, well known from PHP, method of sending emails.
package sendmail

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/mail"
	"os"
	"os/exec"
	"strings"
)

var (
	_, debug = os.LookupEnv("DEBUG")
)

type Mail struct {
	Header http.Header
	From   mail.Address
	To     []mail.Address
	Text   bytes.Buffer
	Html   bytes.Buffer
}

func New(subject, from string, to []string) (*Mail, error) {
	fromAddress, err := mail.ParseAddress(from)
	if err != nil {
		return nil, err
	}
	m := Mail{
		From:   *fromAddress,
		To:     make([]mail.Address, len(to)),
		Header: make(http.Header),
	}
	for i, t := range to {
		toAddress, err := mail.ParseAddress(t)
		if err != nil {
			return nil, err
		}
		m.To[i] = *toAddress
	}
	m.Header.Set("Subject", mime.QEncoding.Encode("utf-8", subject))
	return &m, nil
}

// Send sends an email, or prints it on stderr,
// when environment variable `DEBUG` is set.
func (m *Mail) Send() error {
	m.Header.Set("From", m.From.String())
	to := make([]string, len(m.To))
	arg := make([]string, len(m.To))
	for i, t := range m.To {
		to[i] = t.String()
		arg[i] = t.Address
	}
	m.Header.Set("To", strings.Join(to, ", "))
	if debug {
		var delimiter = strings.Repeat("-", 70)
		fmt.Println(delimiter)
		m.WriteTo(os.Stdout)
		fmt.Println(delimiter)
		return nil
	}
	sendmail := exec.Command("/usr/sbin/sendmail", arg...)
	stdin, err := sendmail.StdinPipe()
	if err != nil {
		return err
	}
	stderr, err := sendmail.StderrPipe()
	if err != nil {
		return err
	}
	if err := sendmail.Start(); err != nil {
		return err
	}
	if err := m.WriteTo(stdin); err != nil {
		return err
	}
	if err := stdin.Close(); err != nil {
		return err
	}
	out, err := ioutil.ReadAll(stderr)
	if err != nil {
		return err
	}
	if len(out) != 0 {
		return errors.New(string(out))
	}
	return sendmail.Wait()
}

func (m *Mail) WriteTo(wr io.Writer) error {
	if err := m.Header.Write(wr); err != nil {
		return err
	}
	if _, err := wr.Write([]byte("\r\n")); err != nil {
		return err
	}
	if _, err := m.Text.WriteTo(wr); err != nil {
		return err
	}
	return nil
}
