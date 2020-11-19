package smtpclient

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	htmlTemplate "html/template"
	"io"
	"net"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"syscall"
	txtTemplate "text/template"
	"time"

	"github.com/jordan-wright/email"
	"gitlab.com/vocdoni/go-dvote/log"
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
	s.pool, err = email.NewPool(net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port)), s.config.PoolSize, s.auth, s.tlsConfig)
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
func (s *SMTP) SendMail(email *email.Email, usePool bool) error {
	// email.Headers = textproto.MIMEHeader{}
	email.Headers.Add("X-Mailgun-Require-TLS", "true")
	email.Headers.Add("X-Mailgun-Skip-Verification", "false")
	if usePool {
		if s.pool == nil {
			return fmt.Errorf("requested pool is not initialized")
		}
		timeout, err := time.ParseDuration(fmt.Sprintf("%ds", s.config.Timeout))
		if err != nil {
			return fmt.Errorf("error calulating timeout: %v", err)
		}
		err = s.pool.Send(email, timeout)
		var failed int
		for (errors.Is(err, syscall.EPIPE) || errors.Is(err, io.EOF)) && failed < 3 {
			log.Errorf("smtp pool error: (%v)", err)
			time.Sleep(200 * time.Millisecond)
			err = s.pool.Send(email, timeout)
			failed++
		}
		return err
	}
	address := net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port))
	return email.SendWithStartTLS(address, s.auth, s.tlsConfig)

}

// SendValidationLink sends a unique validation link to the member m
func (s *SMTP) SendValidationLink(member *types.Member, entity *types.Entity, usePool bool) error {
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

	attachment, err := e.Attach(reader, "logoVoc.png", "image/png; name=logoVoc.png")
	if err != nil {
		return fmt.Errorf("could not attach logo to the email: %v", err)
	}
	attachment.HTMLRelated = true

	return s.SendMail(&e, usePool)
}

// SendVotingLink sends a unique voting link to the member m
func (s *SMTP) SendVotingLink(ephemeralMember *types.EphemeralMemberInfo, entity *types.Entity, processID string, usePool bool) error {
	if ephemeralMember.Email == "" {
		log.Errorf("sendVotingLink: invalid member email for %s", ephemeralMember.ID.String())
		return fmt.Errorf("invalid member email")
	}
	if len(ephemeralMember.PrivKey) == 0 {
		log.Errorf("sendVotingLink: missing privKey for %s", ephemeralMember.ID.String())
		return fmt.Errorf("missing privKey")
	}

	link := fmt.Sprintf("%s/0x%x/%s/0x%x", s.config.WebpollURL, entity.ID, processID, ephemeralMember.PrivKey)
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

	attachment, err := e.Attach(reader, "logoVoc.png", "image/png; name=logoVoc.png")
	if err != nil {
		return fmt.Errorf("could not attach logo to the email: %v", err)
	}
	attachment.HTMLRelated = true

	return s.SendMail(&e, usePool)
}
