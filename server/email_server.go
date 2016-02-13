package server

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/mgutz/logxi/v1"
	"github.com/thejackrabbit/aero/conf"
	"github.com/thejackrabbit/aero/db/orm"
	"github.com/thejackrabbit/aero/ds"
	"github.com/thejackrabbit/aero/panik"
	"github.com/thejackrabbit/aero/que"
	"github.com/thejackrabbit/email"
	"net/mail"
	"net/smtp"
	"time"
)

type EmailServer struct {
	dbo     gorm.DB
	queue   que.Queue
	auth    smtp.Auth
	address string
}

func NewEmailServer() *EmailServer {

	es := EmailServer{
		dbo: orm.ReadConfig("email.server.database"),
		auth: smtp.PlainAuth(
			conf.String("", "email.server.smtp.auth", "identity"),
			conf.String("", "email.server.smtp.auth", "username"),
			conf.String("", "email.server.smtp.auth", "password"),
			conf.String("", "email.server.smtp.auth", "host"),
		),
	}

	// testing
	es.dbo.LogMode(true)

	es.address = conf.String("", "email.server.smtp", "host")
	if conf.Exists("email.server.smtp", "port") {
		es.address = fmt.Sprintf("%s:%d", es.address, conf.Int(0, "email.server.smtp", "port"))
	}

	if conf.Exists("email.queue.default") {
		es.queue = que.NewQueue("email.queue.default")
	}

	// create relational tables if missing
	if !es.dbo.HasTable(Message{}) {
		panik.On(es.dbo.AutoMigrate(&Message{}).Error)
		fmt.Println("[message] table created.")
	}

	return &es
}

func (s *EmailServer) Run() {
	go s.queueToDb()

	batch := conf.Int(3, "email", "server", "batch")
	maxFails := conf.Int(3, "email", "server", "max-fails")
	maxTime := conf.String("1h", "email", "server", "max-time")
	dur, err := time.ParseDuration(maxTime)
	panik.On(err)
	maxTimeSec := int(dur.Seconds())
	now := time.Now().Unix()

	var upd map[string]interface{}
	var msgs []Message

	sql := `
	  sent = 0
	  and num_fails < ?
	  and ((send_after is null and ? - created_at between 0 and ?) or
	  (send_after is not null and ? - send_after between 0 and ?))`

	for {
		s.dbo.Where(sql, maxFails, now, maxTimeSec, now, maxTimeSec).
			Order("priority desc, created_at asc").
			Limit(batch).
			Find(&msgs)

		if len(msgs) == 0 {
			time.Sleep(5 * time.Second)
		} else {
			for i := 0; i < len(msgs); i++ {
				err := s.dispatchMessage(&msgs[i])
				now = time.Now().Unix()
				if err == nil {
					upd = map[string]interface{}{
						"sent":       1,
						"sent_at":    &now,
						"updated_at": &now,
					}
				} else {
					upd = map[string]interface{}{
						"num_fails":      msgs[i].NumFails + 1,
						"last_failed_at": &now,
						"failure":        err.Error(),
						"updated_at":     &now,
					}
				}
				edb := s.dbo.Model(Message{}).Where("id = ?", msgs[i].Id).Updates(upd).Error
				panik.On2(edb, func() {
					log.Error("Failed to update message table", "email-err", err, "db-err", edb)
				})
			}
		}
	}
}

func (s *EmailServer) dispatchMessage(m *Message) error {
	var e email.Email
	//err := ds.LoadStruct(&e, []byte(m.Data))
	err := ds.Load(&e, []byte(m.Data))
	if err != nil {
		return err
	}

	// Merge the To, Cc, and Bcc fields
	to := make([]string, 0, len(e.To)+len(e.Cc)+len(e.Bcc))
	to = append(append(append(to, e.To...), e.Cc...), e.Bcc...)
	for i := 0; i < len(to); i++ {
		addr, err := mail.ParseAddress(to[i])
		if err != nil {
			return err
		}
		to[i] = addr.Address
	}

	// Check to make sure there is at least one recipient and one "From" address
	if e.From == "" || len(to) == 0 {
		return errors.New("Must specify at least one From address and one To address")
	}
	from, err := mail.ParseAddress(e.From)
	if err != nil {
		return err
	}
	raw, err := e.Bytes()
	if err != nil {
		return err
	}

	return smtp.SendMail(s.address, s.auth, from.Address, to, raw)
}

func (s *EmailServer) queueToDb() {

	var em email.Email
	wait, _ := time.ParseDuration("5m")

	for {
		d, err := s.queue.PopWait(wait)
		panik.On(err)
		if len(d) > 0 {
			// read from queue
			//err = ds.LoadStruct(&em, d)
			err = ds.Load(&em, d)
			if err != nil {
				log.Error("Error reading email from queue", "email", em, "error", err)
				panik.On(err)
			}

			// save to db
			m := Message{
				Data:      string(d),
				Priority:  em.Priority,
				CreatedAt: time.Now().Unix(),
				SendAfter: em.SendAfter,
			}
			err = s.dbo.Create(&m).Error
			if err != nil { // on error, put back in queue
				s.queue.Push(d)
				panik.On(err)
			}
		}
	}
}
