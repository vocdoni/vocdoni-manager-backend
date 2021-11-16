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
		m.updateEntity,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/countMembers",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.countMembers,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/listMembers",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.listMembers,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/getMember",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.getMember,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/updateMember",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.updateMember,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/deleteMembers",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.deleteMembers,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/generateTokens",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.generateTokens,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/exportTokens",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.exportTokens,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/importMembers",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.importMembers,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/countTargets",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.countTargets,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/listTargets",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.listTargets,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/getTarget",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.getTarget,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/dumpTarget",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.dumpTarget,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/dumpCensus",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.dumpCensus,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/addCensus",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.addCensus,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/updateCensus",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.updateCensus,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/getCensus",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.getCensus,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/countCensus",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.countCensus,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/listCensus",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.listCensus,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/deleteCensus",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.deleteCensus,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/sendValidationLinks",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.sendValidationLinks,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/sendVotingLinks",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.sendVotingLinks,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/createTag",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.createTag,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/listTags",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.listTags,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/deleteTag",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.deleteTag,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/addTag",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.addTag,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/removeTag",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.removeTag,
	); err != nil {
		return err
	}
	if err := u.api.RegisterMethod(
		"/manager/adminEntityList",
		"GET",
		bearerstdapi.MethodAccessTypePublic,
		m.adminEntityList,
	); err != nil {
		return err
	}
	if m.eth != nil {
		// do not expose this endpoint if the manager does not have an ethereum client
		if err := u.api.RegisterMethod(
			"/manager/requestGas",
			"GET",
			bearerstdapi.MethodAccessTypePublic,
			m.requestGas,
		); err != nil {
			return err
		}
	}
	return nil
}
