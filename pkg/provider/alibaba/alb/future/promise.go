package future

import "time"

type Promise struct {
	FllowControl map[string]chan struct{}
}

func NewPromise() Promise {
	return Promise{
		FllowControl: make(map[string]chan struct{}),
	}
}

func (p *Promise) Start(future Future) {
	futureKey := future.Key()

	p.getFllowControl(futureKey)

	future.Run()

	go future.When()

	future.Result()

	go p.returnAfterOneSecond(futureKey)
}

func (p *Promise) getFllowControl(key string) {
	ch := p.FllowControl[key]
	if ch == nil {
		p.FllowControl[key] = make(chan struct{}, 20)
		ch = p.FllowControl[key]
	}
	ch <- struct{}{}
}

func (p *Promise) returnAfterOneSecond(key string) {
	time.Sleep(time.Second)
	ch := p.FllowControl[key]
	<-ch
}
