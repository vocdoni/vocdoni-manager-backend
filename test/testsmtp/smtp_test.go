package testsmtp

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/textproto"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	email "github.com/knadh/smtppool"
	"gitlab.com/vocdoni/manager/manager-backend/config"
	"gitlab.com/vocdoni/manager/manager-backend/smtpclient"
	"gitlab.com/vocdoni/manager/manager-backend/types"
	"go.vocdoni.io/dvote/util"
)

var s *smtpclient.SMTP
var smtpConfig *config.SMTP

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	//alternative
	//melisa.oberbrunner48@ethereal.email
	//ExFmyARM3dCtaFJfcQ
	smtpConfig = &config.SMTP{
		User:          "coby.rippin@ethereal.email",
		Password:      "HmjWVQ86X3Q6nKBR3u",
		Host:          "smtp.ethereal.email",
		Port:          587,
		ValidationURL: "https://vocdoni.link/validation",
		WebpollURL:    "https://webpoll.vocdoni.net",
		Sender:        "coby.rippin@ethereal.email",
		Timeout:       7,
		PoolSize:      4,
	}
	s = smtpclient.New(smtpConfig)
	if err := s.StartPool(); err != nil {
		os.Exit(1)
	}
	defer s.ClosePool()
	os.Exit(m.Run())
}

func TestSendMail(t *testing.T) {
	t.Run("type=text", func(t *testing.T) {
		// Test text mail
		e := email.Email{
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
		e := email.Email{
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
		e := email.Email{
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

func TestSendMailPool(t *testing.T) {
	count := 3
	e := email.Email{
		From:    smtpConfig.User,
		Sender:  smtpConfig.User,
		To:      []string{smtpConfig.User},
		Subject: "Vocdoni Participation Link Pool Sequential",
		HTML:    []byte("<html><body>This is an HTML text.</body></html>"),
		Headers: textproto.MIMEHeader{},
	}

	t.Run("type=sequential", func(t *testing.T) {
		for i := 0; i < count; i++ {
			if err := s.SendMail(e); err != nil {
				t.Fatalf("error sending sequential mails with pool: (%v)", err)
			}
		}
	})

	t.Run("type=concurrent", func(t *testing.T) {
		count := 3
		var wg sync.WaitGroup
		wg.Add(count)
		c := make(chan error, count)
		for i := 0; i < count; i++ {
			go func(i int) {
				e := email.Email{
					From:    smtpConfig.User,
					Sender:  smtpConfig.User,
					To:      []string{smtpConfig.User},
					Subject: "Vocdoni Participation Link Pool Concurrent",
					HTML:    []byte("<html><body>This is an HTML text.</body></html>"),
					Headers: textproto.MIMEHeader{},
				}
				if i == 2 {
					c <- fmt.Errorf("dummy error")
					wg.Done()
					return
				} else if err := s.SendMail(e); err != nil {
					c <- fmt.Errorf("unable to send HTML email with attachment :%s", err)
					wg.Done()
					return
				}
				wg.Done()
			}(i)
		}
		wg.Wait()
		close(c)
		var errors []error
		for err := range c {
			errors = append(errors, err)
		}
		if len(errors) != 1 {
			t.Fatalf("unexpected errors sending parallel mails with pool: (%v)", errors)
		}
	})
}

func TestValidationLink(t *testing.T) {
	m := &types.Member{
		ID: uuid.New(),
		MemberInfo: types.MemberInfo{
			FirstName: "Manos",
			LastName:  "Voc",
			Email:     "coby.rippin@ethereal.email",
		},
	}
	id, err := hex.DecodeString("1026d682dc423d984abf6c086eca923245a33f45e5d1e06e069ac2663e5fff07")
	if err != nil {
		t.Fatalf("failed to decode hex string: (%v)", err)
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

func TestVotingLink(t *testing.T) {
	// privKey := util.RandomBytes(33)
	privKeyStr := util.RandomHex(33)
	privKey, err := hex.DecodeString(privKeyStr)
	if err != nil {
		t.Fatalf("error decoding merkleRoot string to bytes: (%v)", err)
	}
	processID := fmt.Sprintf("0x%s", util.RandomHex(33))
	m := &types.EphemeralMemberInfo{
		ID:        uuid.New(),
		FirstName: "Manos",
		LastName:  "Voc",
		Email:     "coby.rippin@ethereal.email",
		PrivKey:   privKey,
	}

	id, err := hex.DecodeString("1026d682dc423d984abf6c086eca923245a33f45e5d1e06e069ac2663e5fff07")
	if err != nil {
		t.Fatalf("failed to decode hex string: (%v)", err)
	}
	e := &types.Entity{
		ID: []byte(id),
		EntityInfo: types.EntityInfo{
			Email: "hola@vocdoni.io",
			Name:  "TestOrg",
		},
	}
	if err := s.SendVotingLink(m, e, processID); err != nil {
		t.Fatalf("unable to send participation link email :%s", err)
	}
}
