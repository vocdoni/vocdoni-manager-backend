package smtpclient

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	htmlTemplate "html/template"
	"net/smtp"
	"net/textproto"
	"strings"
	txtTemplate "text/template"
	"time"

	email "github.com/knadh/smtppool"
	"go.vocdoni.io/dvote/log"
	"go.vocdoni.io/manager/config"
	"go.vocdoni.io/manager/types"
)

// SMTP struct maintains the SMTP config and conncection objects
type SMTP struct {
	// pool      *email.Pool
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
	timeout, err := time.ParseDuration(fmt.Sprintf("%ds", s.config.Timeout))
	if err != nil {
		return fmt.Errorf("error calulating timeout: %v", err)
	}
	s.pool, err = email.New(email.Opt{
		Host:              s.config.Host,
		Port:              s.config.Port,
		MaxConns:          s.config.PoolSize,
		MaxMessageRetries: 3,
		IdleTimeout:       time.Second * 10, //default value
		PoolWaitTimeout:   time.Second * timeout,
		Auth:              s.auth,
		TLSConfig:         s.tlsConfig,
	})
	if err != nil {
		return fmt.Errorf("error initializing smtp pool: %v", err)
	}
	return err
}

// ClosePool closes the SMTP pool
func (s *SMTP) ClosePool() {
	s.pool.Close()
}

// SendMail Sends one email over StartTLS (without pool)
func (s *SMTP) SendMail(email email.Email) error {
	email.Headers.Add("X-Mailgun-Require-TLS", "true")
	email.Headers.Add("X-Mailgun-Skip-Verification", "false")
	if s.pool == nil {
		return fmt.Errorf("requested pool is not initialized")
	}

	return s.pool.Send(email)

}

// SendValidationLink sends a unique validation link to the member m
func (s *SMTP) SendValidationLink(member *types.Member, entity *types.Entity) error {
	if member.Email == "" {
		return fmt.Errorf("invalid member email")
	}

	link := fmt.Sprintf("%s/%x/%s", s.config.ValidationURL, entity.ID, member.ID.String())
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
	htmlParsed, err := htmlTemplate.New("body").Parse(ValidationHTMLTemplate)
	if err != nil {
		return fmt.Errorf("error parsing HTML template: %v", err)
	}
	var htmlBuff, txtBuff, subjectBuff bytes.Buffer
	err = htmlParsed.Execute(&htmlBuff, data)
	if err != nil {
		return fmt.Errorf("error adding data to HTML template: %v", err)
	}

	textParsed, err := txtTemplate.New("body").Parse(ValidationTextTemplate)
	if err != nil {
		return fmt.Errorf("error parsing text template: %v", err)
	}
	err = textParsed.Execute(&txtBuff, data)
	if err != nil {
		return fmt.Errorf("error adding data to text template: %v", err)
	}

	subjectParsed, err := txtTemplate.New("subject").Parse(ValidationSubject)
	if err != nil {
		return fmt.Errorf("error parsing mail subject: %v", err)
	}
	err = subjectParsed.Execute(&subjectBuff, data)
	if err != nil {
		return fmt.Errorf("error adding data to mail subject: %v", err)
	}

	e := email.Email{
		From:    fmt.Sprintf("%s <%s>", s.config.SenderName, s.config.Sender),
		Sender:  s.config.Sender,
		To:      []string{fmt.Sprintf("%q %q <%s>", member.FirstName, member.LastName, member.Email)},
		Subject: subjectBuff.String(),
		Text:    []byte(txtBuff.Bytes()),
		HTML:    []byte(htmlBuff.Bytes()),
		Headers: textproto.MIMEHeader{},
	}

	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(LogoVocBase64))

	if _, err := e.Attach(reader, "logoVoc.png", "image/png; name=logoVoc.png"); err != nil {
		return fmt.Errorf("could not attach logo to the email: %v", err)
	}
	e.Attachments[0].HTMLRelated = true

	return s.SendMail(e)
}

// SendVotingLink sends a unique voting link to the member m
func (s *SMTP) SendVotingLink(ephemeralMember *types.EphemeralMemberInfo, entity *types.Entity, processID []byte) error {
	if ephemeralMember.Email == "" {
		log.Errorf("sendVotingLink: invalid member email for %s", ephemeralMember.ID.String())
		return fmt.Errorf("invalid member email")
	}
	if len(ephemeralMember.PrivKey) == 0 {
		log.Errorf("sendVotingLink: missing privKey for %s", ephemeralMember.ID.String())
		return fmt.Errorf("missing privKey")
	}

	link := fmt.Sprintf("%s/%x/%x/%x", s.config.WebpollURL, entity.ID, processID, ephemeralMember.PrivKey)
	data := struct {
		Name       string
		OrgName    string
		OrgEmail   string
		VotingLink string
		OrgMessage string
	}{
		Name:       ephemeralMember.FirstName,
		OrgName:    entity.Name,
		OrgEmail:   entity.Email,
		VotingLink: link,
		OrgMessage: "",
	}
	htmlParsed, err := htmlTemplate.New("body").Parse(VotingHTMLTemplate)
	if err != nil {
		return fmt.Errorf("error parsing HTML template: %v", err)
	}
	var htmlBuff, txtBuff, subjectBuff bytes.Buffer
	err = htmlParsed.Execute(&htmlBuff, data)
	if err != nil {
		return fmt.Errorf("error adding data to HTML template: %v", err)
	}

	textParsed, err := txtTemplate.New("body").Parse(VotingTextTemplate)
	if err != nil {
		return fmt.Errorf("error parsing text template: %v", err)
	}
	err = textParsed.Execute(&txtBuff, data)
	if err != nil {
		return fmt.Errorf("error adding data to text template: %v", err)
	}

	subjectParsed, err := txtTemplate.New("subject").Parse(ValidationSubject)
	if err != nil {
		return fmt.Errorf("error parsing mail subject: %v", err)
	}
	err = subjectParsed.Execute(&subjectBuff, data)
	if err != nil {
		return fmt.Errorf("error adding data to mail subject: %v", err)
	}

	e := email.Email{
		From:    fmt.Sprintf("%s <%s>", s.config.SenderName, s.config.Sender),
		Sender:  s.config.Sender,
		To:      []string{fmt.Sprintf("%q %q <%s>", ephemeralMember.FirstName, ephemeralMember.LastName, ephemeralMember.Email)},
		Subject: subjectBuff.String(),
		Text:    []byte(txtBuff.Bytes()),
		HTML:    []byte(htmlBuff.Bytes()),
		Headers: textproto.MIMEHeader{},
	}

	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(LogoVocBase64))

	if _, err := e.Attach(reader, "logoVoc.png", "image/png; name=logoVoc.png"); err != nil {
		return fmt.Errorf("could not attach logo to the email: %v", err)
	}
	e.Attachments[0].HTMLRelated = true

	return s.SendMail(e)
}
