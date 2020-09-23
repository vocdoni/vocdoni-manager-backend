package smtpclient

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	htmlTemplate "html/template"
	"net"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	txtTemplate "text/template"

	"github.com/jordan-wright/email"
	"gitlab.com/vocdoni/manager/manager-backend/config"
	"gitlab.com/vocdoni/manager/manager-backend/types"
)

// SMTP struct maintains the SMTP config and conncection objects
type SMTP struct {
	pool      *email.Pool
	config    *config.SMTP
	auth      smtp.Auth
	tlsConfig *tls.Config
}

// New creates a new SMTP object initialized with the user config
func New(smtpc *config.SMTP) *SMTP {
	// Prepare TLS auth
	auth := smtp.PlainAuth("", smtpc.User, smtpc.Password, smtpc.Host)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         smtpc.Host,
	}
	return &SMTP{config: smtpc, auth: auth, tlsConfig: tlsConfig}
}

// StartPool ooens a new STMP pool using the config values
func (s *SMTP) StartPool() error {
	var err error
	s.pool, err = email.NewPool(net.JoinHostPort(s.config.Host, string(s.config.Port)), s.config.PoolSize, s.auth, s.tlsConfig)
	if err != nil {
		return fmt.Errorf("error initializing smpt pool: %v", err)
	}
	return err
}

// ClosePool closes the SMTP pool
func (s *SMTP) ClosePool() {
	s.pool.Close()
}

// SendMail Sends one email over StartTLS (without pool)
func (s *SMTP) SendMail(email *email.Email) error {
	// email.Headers = textproto.MIMEHeader{}
	email.Headers.Add("X-Mailgun-Require-TLS", "true")
	email.Headers.Add("X-Mailgun-Skip-Verification", "false")
	address := net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port))
	return email.SendWithStartTLS(address, s.auth, s.tlsConfig)
}

// SendValidationLink sends a unique validation link to the member m
func (s *SMTP) SendValidationLink(member *types.Member, entity *types.Entity) error {
	if member.Email == "" {
		return fmt.Errorf("invalid member email")
	}

	link := fmt.Sprintf("%s/0x%x/%s", s.config.ValidationURL, entity.ID, member.ID.String())
	data := struct {
		Name           string
		OrgName        string
		OrgEmail       string
		ValidationLink string
	}{
		Name:           member.FirstName,
		OrgName:        entity.Name,
		OrgEmail:       entity.Email,
		ValidationLink: link,
	}
	htmlParsed, err := htmlTemplate.New("body").Parse(HTMLTemplate)
	if err != nil {
		return fmt.Errorf("error parsing HTML template: %v", err)
	}
	var htmlBuff, txtBuff, subjectBuff bytes.Buffer
	err = htmlParsed.Execute(&htmlBuff, data)
	if err != nil {
		return fmt.Errorf("error adding data to HTML template: %v", err)
	}

	textParsed, err := txtTemplate.New("body").Parse(TextTemplate)
	if err != nil {
		return fmt.Errorf("error parsing text template: %v", err)
	}
	err = textParsed.Execute(&txtBuff, data)
	if err != nil {
		return fmt.Errorf("error adding data to text template: %v", err)
	}

	subjectParsed, err := txtTemplate.New("subject").Parse(Subject)
	if err != nil {
		return fmt.Errorf("error parsing mail subject: %v", err)
	}
	err = subjectParsed.Execute(&subjectBuff, data)
	if err != nil {
		return fmt.Errorf("error adding data to mail subject: %v", err)
	}

	e := &email.Email{
		From:    fmt.Sprintf("%s <%s>", s.config.SenderName, s.config.Sender),
		Sender:  s.config.Sender,
		To:      []string{fmt.Sprintf("%q %q <%s>", member.FirstName, member.LastName, member.Email)},
		Subject: subjectBuff.String(),
		Text:    []byte(txtBuff.Bytes()),
		HTML:    []byte(htmlBuff.Bytes()),
		Headers: textproto.MIMEHeader{},
	}

	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(LogoVocBase64))

	attachment, err := e.Attach(reader, "logoVoc.png", "image/png; name=logoVoc.png")
	if err != nil {
		return fmt.Errorf("could not attach logo to the email: %v", err)
	}
	attachment.HTMLRelated = true

	return s.SendMail(e)
}
