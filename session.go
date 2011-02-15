package session

import (
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"http"
	"log"
	"regexp"
	"strings"
	"syscall"
	"time"
)

type Cookie struct {
	Items map[string]string
	Path string
	Expires *time.Time
	Domain string
	Secure bool
	HttpOnly bool
	Value interface{}
}

func (c *Cookie) Dump() {
	for k, v := range c.Items {
		println(k, "=", v)
	}
}

func (c *Cookie) Has(key string) bool {
	_, found := c.Items[key]
	return found
}

func (c *Cookie) Get(key string) string {
	if c == nil {
		return ""
	}
	value, found := c.Items[key]
	if found {
		return value
	}
	return ""
}

type Session struct {
	Id string
	Value interface{}
	expire int64
	manager *SessionManager
	res http.ResponseWriter
}

func (session *Session) Abandon() {
	_, found := (*session.manager).sessionMap[session.Id]
	if found {
		(*session.manager).sessionMap[session.Id] = nil, false
	}
	if session.res != nil {
		session.res.SetHeader("Set-Cookie", "SessionId=; path=/;")
	}
}

func (session *Session) Cookie() string {
	tm := time.SecondsToUTC(session.expire)
	return fmt.Sprintf("SessionId=%s; path=/; expires=%s;", session.Id, tm.Format("Fri, 02-Jan-2006 15:04:05 -0700"))
}

type SessionManager struct {
	sessionMap map[string]*Session
	onStart func(*Session)
	onEnd func(*Session)
	timeout uint
}

func NewSessionManager(logger *log.Logger) *SessionManager {
	manager := &SessionManager{make(map[string]*Session), nil, nil, 300}
	go func(manager *SessionManager) {
		for {
			l := time.LocalTime().Seconds()
			for id, v := range (*manager).sessionMap {
				if v.expire < l {
					// expire
					if logger != nil {
						logger.Printf("Expired session(id:%s)", id)
					}
					f := (*manager).onEnd
					if f != nil {
						f((*manager).sessionMap[id])
					}
					(*manager).sessionMap[id] = nil, false
				}
			}
			syscall.Sleep(1000000000)
		}
	}(manager)
	return manager
}

func (manager *SessionManager) OnStart(f func(*Session)) { manager.onStart = f }
func (manager *SessionManager) OnEnd(f func(*Session)) { manager.onEnd = f }
func (manager *SessionManager) SetTimeout(t uint) { manager.timeout = t }
func (manager *SessionManager) GetTimeout() uint { return manager.timeout }

func (manager *SessionManager) GetSessionById(id string) *Session {
	if id == "" || !manager.Has(id) {
		b := make([]byte, 32)
		_, err := rand.Read(b)
		if err != nil {
			return nil
		}
		m := md5.New()
		m.Write(b)
		id = fmt.Sprintf("%x", m.Sum())
	}
	tm := time.SecondsToUTC(time.LocalTime().Seconds() + int64(manager.timeout))
	session, found := (*manager).sessionMap[id]
	if !found {
		session = &Session{id, nil, tm.Seconds(), manager, nil}
		(*manager).sessionMap[id] = session
		f := (*manager).onStart
		if f != nil {
			f(session)
		}
	}
	return session
}

func (manager *SessionManager) GetSession(res http.ResponseWriter, req *http.Request) *Session {
	c := parseCookie(req)
	session := manager.GetSessionById(c.Get("SessionId"))
	if res != nil {
		session.res = res
		res.SetHeader("Set-Cookie",
			fmt.Sprintf("SessionId=%s; path=/; expires=%s;",
			session.Id,
			time.SecondsToUTC(session.expire).Format(
				"Fri, 02-Jan-2006 15:04:05 -0700")))
	}
	return session
}

func (manager *SessionManager) Has(id string) bool {
	_, found := (*manager).sessionMap[id]
	return found
}

func StringToCookie(h string) *Cookie {
	c := new(Cookie)
	c.Items = make(map[string]string)
	re, _ := regexp.Compile("[^=]+=[^;]+(; *(expires=[^;]+|path=[^;,]+|domain=[^;,]+|secure|HttpOnly))*,?")
	rs := re.FindAllString(h, -1)
	for _, ss := range rs {
		m := strings.Split(ss, ";", -1)
		for _, n := range m {
			t := strings.Split(n, "=", 2)
			if len(t) == 2 {
				t[0] = strings.Trim(t[0], " ")
				t[1] = strings.Trim(t[1], " ")
				switch t[0] {
				case "domain":
					c.Domain = t[1]
				case "path":
					c.Path = t[1]
				case "expires":
					tm, err := time.Parse("Fri, 02-Jan-2006 15:04:05 MST", t[1])
					if err != nil {
						tm, err = time.Parse("Fri, 02-Jan-2006 15:04:05 -0700", t[1])
					}
					c.Expires = tm
				case "secure":
					c.Secure = true
				case "HttpOnly":
					c.HttpOnly = true
				default:
					c.Items[t[0]] = t[1]
				}
			}
		}
	}
	return c
}

func parseCookie(req *http.Request) *Cookie {
	h, found := req.Header["Cookie"]
	if found && len(h) > 0 {
		return StringToCookie(h)
	}
	return nil
}
