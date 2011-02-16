package main

import (
	"http/session"
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

type User struct {
	UserId   string
	Password string
	RealName string
	Age      int64
}

func main() {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	manager := session.NewSessionManager(logger)
	manager.OnStart(func(session *session.Session) {
		println("started new session")
	})
	manager.OnEnd(func(session *session.Session) {
		println("abandon")
	})
	manager.SetTimeout(10)

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


	GetSession := func(ctx *web.Context) *session.Session {
		id, _ := ctx.GetSecureCookie("SessionId")
		session := manager.GetSessionById(id)
		ctx.SetSecureCookie("SessionId", session.Id, int64(manager.GetTimeout()))
		ctx.SetHeader("Pragma", "no-cache", true)
		return session
	}

	web.Config.CookieSecret = "7C19QRmwf3mHZ9CPAaPQ0hsWeufKd"
	web.Get("/", func(ctx *web.Context) {
		session := GetSession(ctx)
		tmpl.Execute(ctx, map[string]interface{}{"session": session})
	})
	web.Post("/login", func(ctx *web.Context) {
		session := GetSession(ctx)
		userid := strings.Trim(ctx.Params["userid"], " ")
		password := strings.Trim(ctx.Params["password"], " ")
		if userid != "" && password != "" {
			sql := "select userid,password,realname,age from User where userid = ? and password = ?"
			st, _ := db.Prepare(sql, userid, password)
			_, e = st.All(func(s *sqlite3.Statement, values ...interface{}) {
				user := new(User)
				user.UserId = values[0].(string)
				user.Password = values[1].(string)
				user.RealName = values[2].(string)
				user.Age = values[3].(int64)
				session.Value = user
			})
		}
		ctx.Redirect(302, "/")
	})
	web.Post("/logout", func(ctx *web.Context) {
		session := GetSession(ctx)
		if session.Value != nil {
			// XXX: get user own object.
			logger.Printf("User \"%s\" logout", session.Value.(User))
			session.Abandon()
		}
		ctx.Redirect(302, "/")
	})
	web.Run(":6061")
}
