package urlapi

import (
	"fmt"

	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/manager/notify"
)

func (u *URLAPI) EnableNotifyHandlers(notif *notify.API) error {
	if notif == nil {
		return fmt.Errorf("notification service is nil")
	}
	u.notif = notif
	if err := u.api.RegisterMethod(
		"/notifications/register",
		"POST",
		bearerstdapi.MethodAccessTypePublic,
		u.notif.Register,
	); err != nil {
		return err
	}
	return nil
}
