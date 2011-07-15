package main

import (
	"github.com/mattn/go-session-manager"
	"log"
	"os"
	"strings"
	"sqlite3"
	"template"
	"web"
)

const page = `
<!doctype html>
<html>
<meta charset="utf-8"/>
<body>
{.section session}
{.section Value}
Hi {UserId}.<br />
Your real name is {RealName}. And you are {Age} years old.<br />
<form method="post" action="/logout">
<input type="submit" name="method" value="logout" />
</form>
You will logout after 10 seconds. Then try to reload.
{.or}
<form method="post" action="/login">
<label for="name">Name:</label><br />
<input type="text" id="userid" name="userid" value="" /><br />
<label for="password">Password:</label><br />
<input type="password" id="password" name="password" value="" /><br />
<input type="submit" name="method" value="login" />
</form>
{.end}
{.end}
</body>
</html>
`

var fmap = template.FormatterMap{"html": template.HTMLFormatter}
var tmpl = template.MustParse(page, fmap)
var logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)
var manager = session.NewSessionManager(logger)

type User struct {
	UserId   string
	Password string
	RealName string
	Age      int64
}

func main() {
	//------------------------------------------------
	// initialize session manager
	manager.OnStart(func(session *session.Session) {
		logger.Printf("Start session(\"%s\")", session.Id)
	})
	manager.OnEnd(func(session *session.Session) {
		logger.Printf("End session(\"%s\")", session.Id)
	})
	manager.SetTimeout(10)

	//------------------------------------------------
	// initialize database
	sqlite3.Initialize()
	db, e := sqlite3.Open(":memory:")
	if e != nil {
		logger.Print(e.String())
		return
	}
	defer db.Close()
	table := sqlite3.Table{"User", "userid varchar(16), password varchar(20), realname varchar(20), age integer"}
	if table.Create(db) == nil {
		db.Execute("insert into User values('go', 'lang', 'golang', 3)")
		db.Execute("insert into User values('perl', 'monger', 'perlmonger', 20)")
	}
	sql := "select userid,password,realname,age from User where userid = ? and password = ?"

	//------------------------------------------------
	// utility function for web.go
	GetSession := func(ctx *web.Context) *session.Session {
		id, _ := ctx.GetSecureCookie("SessionId")
		session := manager.GetSessionById(id)
		ctx.SetSecureCookie("SessionId", session.Id, int64(manager.GetTimeout()))
		ctx.SetHeader("Pragma", "no-cache", true)
		return session
	}
	Param := func(ctx *web.Context, name string) string {
		value, found := ctx.Params[name]
		if found {
			return strings.Trim(value, " ")
		}
		return ""
	}

	//------------------------------------------------
	// go to web
	web.Config.CookieSecret = "7C19QRmwf3mHZ9CPAaPQ0hsWeufKd"

	web.Get("/", func(ctx *web.Context) {
		session := GetSession(ctx)
		tmpl.Execute(ctx, map[string]interface{}{"session": session})
	})
	web.Post("/login", func(ctx *web.Context) {
		session := GetSession(ctx)
		userid := Param(ctx, "userid")
		password := Param(ctx, "password")
		if userid != "" && password != "" {
			// find user
			st, _ := db.Prepare(sql, userid, password)
			_, e = st.All(func(s *sqlite3.Statement, values ...interface{}) {
				// store User object to sessino
				session.Value = &User{values[0].(string), values[1].(string), values[2].(string), values[3].(int64)}
				logger.Printf("User \"%s\" login", session.Value.(*User).UserId)
			})
		}
		ctx.Redirect(302, "/")
	})
	web.Post("/logout", func(ctx *web.Context) {
		session := GetSession(ctx)
		if session.Value != nil {
			// abandon
			logger.Printf("User \"%s\" logout", session.Value.(*User).UserId)
			session.Abandon()
		}
		ctx.Redirect(302, "/")
	})
	web.Run(":6061")
}
