package email

import (
	"github.com/thejackrabbit/aero/conf"
	"github.com/thejackrabbit/aero/ds"
	"github.com/thejackrabbit/aero/panik"
	"github.com/thejackrabbit/aero/que"
	"time"
)

func (e *Email) Enque() {

	if mails == nil { // initialization
		startLoop()
	}

	go func() {
		d, err := ds.ToBytes(e)
		panik.On(err)
		mails <- d
	}()
}

func (e *Email) SendLater(after string) {
	d, err := time.ParseDuration(after)
	panik.On(err)
	e.SendAt(time.Now().Add(d))
}

func (e *Email) SendAt(t time.Time) {
	u := t.Unix()
	if time.Now().Unix() < u {
		panik.Do("Send time %s is earlier than current time", t)
	}
	e.SendAfter = &u
	e.Enque()
}

var Queue que.Queue

var mails chan []byte

func init() {
	keys := []string{"email", "queue", "default"}
	if conf.Exists(keys...) {
		Queue = que.NewQueue(keys...)
	}
}

func startLoop() {
	if mails != nil {
		return
	}

	mails = make(chan []byte, conf.Int(250, "email", "buffer"))
	panik.If(Queue == nil, "Queue is not initialized")

	go func() {
		for {
			select {
			case mb := <-mails:
				err := Queue.Push(mb)
				panik.On(err)
			}
		}
	}()
}
