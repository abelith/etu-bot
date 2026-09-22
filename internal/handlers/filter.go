package handlers

import (
	mcontext "github.com/abelith/etu-bot/pkg/maxlib/context"
	"github.com/abelith/etu-bot/pkg/maxlib/core"
)

func InhabitantFilter() core.Filter {
	return func(c *mcontext.Context) bool {
		return true
	}
}

func OrgMemberFilter() core.Filter {
	return func(c *mcontext.Context) bool {
		return true
	}
}
