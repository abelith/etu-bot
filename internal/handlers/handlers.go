package handlers

import (
	"github.com/abelith/etu-bot/pkg/maxlib/core"
)

func NewHandlers(
	suc StarHandlersUseCases,
	umuc UserMenuUseCases,
	omuc OrgMenuUseCases,
) *core.Router {
	sh := (&StartHandlers{UseCases: suc}).Router()
	umh := (&UserMenuHandlers{UseCases: umuc}).Router()
	omh := (&OrgMenuHandlers{UseCases: omuc}).Router()
	rt := &core.Router{}
	rt.Handle(sh)
	rt.Handle(umh, InhabitantFilter())
	rt.Handle(omh, OrgMemberFilter())
	return rt
}
