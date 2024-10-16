package actor

import (
	"github.com/asynkron/protoactor-go/actor"
)

type Stash struct {
	stash []stashElem
}

type stashElem struct {
	msg    any
	sender *actor.PID
}

func (stash *Stash) Stash(ctx actor.Context, msg any) {
	stash.stash = append(stash.stash, stashElem{
		msg:    msg,
		sender: ctx.Sender(),
	})
}

func (stash *Stash) UnstashAll(ctx actor.Context) {
	for _, elem := range stash.stash {
		ctx.RequestWithCustomSender(ctx.Self(), elem.msg, elem.sender)
	}
	stash.stash = nil
}

func (stash *Stash) UnstashOldest(ctx actor.Context) {
	if len(stash.stash) > 0 {
		first := stash.stash[0]
		ctx.RequestWithCustomSender(ctx.Self(), first.msg, first.sender)
		stash.stash = stash.stash[1:]
	}
}

func (stash *Stash) UnstashNewest(ctx actor.Context) {
	if len(stash.stash) > 0 {
		last := stash.stash[len(stash.stash)-1]
		ctx.RequestWithCustomSender(ctx.Self(), last.msg, last.sender)
		stash.stash = stash.stash[:len(stash.stash)-1]
	}
}
