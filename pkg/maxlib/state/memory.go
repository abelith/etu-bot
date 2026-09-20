package state

import (
	"context"
	"github.com/abelith/etu-bot/pkg/maxlib/errors"
	"sync"
)

type FSMStorage struct {
	m sync.Map
}

func NewMemoryFSMStorage() *FSMStorage {
	return &FSMStorage{}
}

func (f *FSMStorage) Get(ctx context.Context, userID int) (*Context, error) {
	state, ok := f.m.Load(userID)
	if !ok {
		return nil, errors.ErrStateNotFound
	}

	return state.(*Context), nil
}

func (f *FSMStorage) Put(_ context.Context, state *Context) error {
	f.m.Store(state.id, state)
	return nil
}

func (f *FSMStorage) Delete(_ context.Context, state *Context) error {
	f.m.Delete(state.id)
	return nil
}
