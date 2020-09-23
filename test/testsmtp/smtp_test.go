package testsmtp

import (
	"encoding/hex"
	"math/rand"
	"net/textproto"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jordan-wright/email"
	"gitlab.com/vocdoni/manager/manager-backend/config"
	"gitlab.com/vocdoni/manager/manager-backend/smtpclient"
	"gitlab.com/vocdoni/manager/manager-backend/types"
)

var s *smtpclient.SMTP
var smtpConfig *config.SMTP

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	smtpConfig = &config.SMTP{
		User:          "coby.rippin@ethereal.email",
		Password:      "HmjWVQ86X3Q6nKBR3u",
		Host:          "smtp.ethereal.email",
		Port:          587,
		ValidationURL: "https://vocdoni.link/validation",
		Sender:        "coby.rippin@ethereal.email",
	}
	s = smtpclient.New(smtpConfig)
	os.Exit(m.Run())
}

func TestSendMail(t *testing.T) {
	t.Run("type=text", func(t *testing.T) {
		// Test text mail
		e := &email.Email{
			From:    smtpConfig.User,
			Sender:  smtpConfig.User,
			To:      []string{smtpConfig.User},
			Subject: "Vocdoni Participation Link",
			Text:    []byte("Hello There"),
			Headers: textproto.MIMEHeader{},
		}
		if err := s.SendMail(e); err != nil {
			t.Fatalf("unable to send simple text email :%s", err)
		}
	})

	// Test html mail
	t.Run("type=html", func(t *testing.T) {
		e := &email.Email{
			From:    smtpConfig.User,
			Sender:  smtpConfig.User,
			To:      []string{smtpConfig.User},
			Subject: "Vocdoni Participation Link",
			HTML:    []byte("<html><body>This is an HTML text.</body></html>"),
			Headers: textproto.MIMEHeader{},
		}
		if err := s.SendMail(e); err != nil {
			t.Fatalf("unable to send simple HTML email :%s", err)
		}
	})

	// Test attachment
	t.Run("type=attachment", func(t *testing.T) {
		e := &email.Email{
			From:    smtpConfig.User,
			Sender:  smtpConfig.User,
			To:      []string{smtpConfig.User},
			Subject: "Vocdoni Participation Link",
			HTML:    []byte("<html><body>This is an HTML text.</body></html>"),
			Headers: textproto.MIMEHeader{},
		}
		if err := s.SendMail(e); err != nil {
			t.Fatalf("unable to send HTML email with attachment :%s", err)
		}
	})

}

func TestValidationLink(t *testing.T) {
	m := &types.Member{
		ID: uuid.New(),
		MemberInfo: types.MemberInfo{
			FirstName: "Manos",
			LastName:  "Voc",
			Email:     "manos@vocdoni.io",
		},
	}
	id, err := hex.DecodeString("1026d682dc423d984abf6c086eca923245a33f45e5d1e06e069ac2663e5fff07)")
	if err != nil {
		t.Fatal("failed to decode hex string")
	}
	e := &types.Entity{
		ID: []byte(id),
		EntityInfo: types.EntityInfo{
			Email: "hola@vocdoni.io",
			Name:  "TestOrg",
		},
	}
	if err := s.SendValidationLink(m, e); err != nil {
		t.Fatalf("unable to send participation link email :%s", err)
	}
}
