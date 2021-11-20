package urlapi

import (
	"fmt"

	"go.vocdoni.io/dvote/httprouter/bearerstdapi"
	"go.vocdoni.io/manager/manager"
)

func (u *URLAPI) EnableManagerHandlers(m *manager.Manager) error {
	if m == nil {
		return fmt.Errorf("manager is nil")
	}
	u.manager = m
	if err := u.api.RegisterMethod(
		"/manager/signUp",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.manager.SignUp,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/getEntity",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		u.manager.GetEntity,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/updateEntity",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.UpdateEntity,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/countMembers",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.CountMembers,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/listMembers",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.ListMembers,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/getMember",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.GetMember,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/updateMember",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.UpdateMember,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/deleteMembers",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.DeleteMembers,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/generateTokens",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.GenerateTokens,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/exportTokens",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.ExportTokens,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/importMembers",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.ImportMembers,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/countTargets",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.CountTargets,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/listTargets",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.ListTargets,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/getTarget",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.GetTarget,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/dumpTarget",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.DumpTarget,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/dumpCensus",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.DumpCensus,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/addCensus",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.AddCensus,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/updateCensus",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.UpdateCensus,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/getCensus",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.GetCensus,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/countCensus",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.CountCensus,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/listCensus",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.ListCensus,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/deleteCensus",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.DeleteCensus,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/sendValidationLinks",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.SendValidationLinks,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/sendVotingLinks",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.SendVotingLinks,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/createTag",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.CreateTag,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/listTags",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.ListTags,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/deleteTag",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.DeleteTag,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/addTag",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.AddTag,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/removeTag",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.RemoveTag,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/adminEntityList",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.AdminEntityList,
	); err != nil {
		return err
	}
	if m.HasEthClient() {
		// do not expose this endpoint if the manager does not have an ethereum client
		if err := u.api.RegisterMethod(
			"/manager/requestGas",
			"GET",
			bearerstdapi.MethodAccessTypePublic,
			m.RequestGas,
		); err != nil {
			return err
		}
	}
	return nil
}
