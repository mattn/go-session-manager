package session

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Session struct {
	Id      string
	Value   interface{}
	expire  int64
	manager *SessionManager
	res     http.ResponseWriter
}

type SessionManager struct {
	sessionMap map[string]*Session
	onStart    func(*Session)
	onEnd      func(*Session)
	timeout    uint
	path       string
	mutex      sync.RWMutex
}

func (session *Session) Abandon() {
	_, found := (*session.manager).sessionMap[session.Id]
	if found {
		delete((*session.manager).sessionMap, session.Id)
	}
	if session.res != nil {
		session.res.Header().Set("Set-Cookie", fmt.Sprintf("SessionId=; path=%s;", session.manager.path))
	}
}

func (session *Session) Cookie() string {
	tm := time.Unix(session.expire, 0).UTC()
	return fmt.Sprintf("SessionId=%s; path=%s; expires=%s;", session.Id, session.manager.path, tm.Format("Fri, 02-Jan-2006 15:04:05 -0700"))
}

func NewSessionManager(logger *log.Logger) *SessionManager {
	manager := new(SessionManager)
	manager.sessionMap = make(map[string]*Session)
	manager.timeout = 300
	manager.path = "/"
	go func(manager *SessionManager) {
		for {
			l := time.Now().Unix()
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
					delete((*manager).sessionMap, id)
				}
			}
			time.Sleep(time.Second)
		}
	}(manager)
	return manager
}

func (manager *SessionManager) OnStart(f func(*Session)) {
	manager.onStart = f
}

func (manager *SessionManager) Abandon() {
	f := (*manager).onEnd
	for id, _ := range (*manager).sessionMap {
		if f != nil {
			f((*manager).sessionMap[id])
		}
		delete((*manager).sessionMap, id)
	}
}

func (manager *SessionManager) OnEnd(f func(*Session)) {
	manager.onEnd = f
}

func (manager *SessionManager) SetTimeout(t uint) {
	manager.timeout = t
}

func (manager *SessionManager) GetTimeout() uint {
	return manager.timeout
}

func (manager *SessionManager) SetPath(t string) {
	manager.path = t
}

func (manager *SessionManager) GetPath() string {
	return manager.path
}

func (manager *SessionManager) GetSessionById(id string) (session *Session) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	if id == "" || !manager.Has(id) {
		b := make([]byte, 16)
		_, err := rand.Read(b)
		if err != nil {
			return
		}
		id = fmt.Sprintf("%x", b)
	}
	tm := time.Unix(time.Now().Unix()+int64(manager.timeout), 0).UTC()
	var found bool
	session, found = (*manager).sessionMap[id]
	if !found {
		session = &Session{id, nil, tm.Unix(), manager, nil}
		(*manager).sessionMap[id] = session
		f := (*manager).onStart
		if f != nil {
			f(session)
		}
	} else {
		session.expire = tm.Unix()
	}
	return
}

func (manager *SessionManager) GetSession(res http.ResponseWriter, req *http.Request) (session *Session) {
	if c, _ := req.Cookie("SessionId"); c != nil {
		session = manager.GetSessionById(c.Value)
	} else {
		session = manager.GetSessionById("")
	}
	if res != nil {
		session.res = res
		res.Header().Add("Set-Cookie",
			fmt.Sprintf("SessionId=%s; path=%s; expires=%s;",
				session.Id,
				session.manager.path,
				time.Unix(session.expire, 0).UTC().Format(
					"Fri, 02-Jan-2006 15:04:05 GMT")))
	}
	return
}

func (manager *SessionManager) Has(id string) (found bool) {
	_, found = (*manager).sessionMap[id]
	return
}
