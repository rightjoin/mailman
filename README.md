email + email-server
=====

This is a fork of amazing yet simple, '**Robust and flexible email library for Go**'.

Or, as the author also calls it rather accurately: '**Email for Humans**' [http://github.com/jordan-wright/email](http://github.com/jordan-wright/email)

---


####Does it remove anything from the original library?
Yes, it **removes** the ability to send mail synchronously!

It **hides** the Send() method of the 'Email' struct. So, following won't compile.

```
e := email.NewEmail()
e.Send("smtp.gmail.com:587", smtp.PlainAuth("", "test@gmail.com", "password123", "smtp.gmail.com"))
```

---

####How does one send email then?

In place of Send(), there are 3 new methods availble:

```
e.Enque()         // Option1: Put the mail in a queue for sending it rightaway
e.SendLater("5m") // Option2: Put the mail in a queue & send after 5 minutes 
e.SendAt(time)    // Option3: Put the mail in a queue and send it after the specified time
```

Note: all three of these methods send emails asynchronously.

---

#### What is the basic architecture?

- Firstly, a message-queue architecture is used to send all the emails into a queue. So, if you have multiple app-servers sending emails, then they will all end-up in the queue(s). [this decouples the sender from the dispatcher]

- Secondly, the emails are read from the queue and stored in a database. And then finally dispatched from the database. (this dispatching is done by a different process, the email-server)

---

#### What are the advantages of this architecture?

The client doesn't block, it simply puts the email into the queue and moves on. 

In case of failures, the system retries automatically to send the email. You can configure two things:

- How many times a specific email should be retried in case of failure (default max: 3)
- What is the max duration to dispatch a message. If a message is not dispatched in this duration then it is considered closed. (default: 1hour)

In cases of failure, the reason gets recorded in the database.

You can also get stats in terms of emails sent and time of delivery (by querying the database yourself).

---

#### What are the basic parameters, and how can they be configured?

Configuration is read from YAML files using the [aero/conf](http://github.com/thejackrabbit/aero/conf) library. 

```
email:
    buffer: 250
    queue:
        default:
            engine: redis
            host:   dockerhost
            port:   6379
            db:     0
            name:   myqueue
```
**email.buffer:** the lenght of the channel which stores emails only momentarily before they are pushed to the queue

**email.queue.default:** holds the connection information to the queue. You can see the [aero/que](http://github.com/thejackrabbit/aero/que) project for more details


---

#### What are the email-server parameters (for dispatching), and how can those be configured?

```
email:
    queue:
        default:
            engine: redis
            host:   dockerhost
            port:   6379
            db:     0
            name:   what
    server:
        max-fails: 3
        max-time:  1h
        batch:     50
        database:
            engine: sqlite3
            path:   /Users/mak/Downloads/sql.db
        smtp:
            host: smtp.gmail.com
            port: 567
            auth:
                username: abc@def.com
                password: ghi
                host: smtp.gmail.com
```

---




