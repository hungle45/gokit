package conc

import "sync"

type Group interface {
	Go(func())
	Wait()
}

type group struct {
	wg *sync.WaitGroup
}

func NewGroup() Group {
	return &group{
		wg: &sync.WaitGroup{},
	}
}

func (g *group) Go(fn func()) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()
		fn()
	}()
}

func (g *group) Wait() {
	g.wg.Wait()
}
