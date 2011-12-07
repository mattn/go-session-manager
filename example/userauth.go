package main

import (
	"exp/sql"
	"github.com/hoisie/web.go"
	"github.com/mattn/go-session-manager"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"strings"
	"text/template"
)

const dbfile = "./user.db"

const page = `
<html>
<meta charset="utf-8"/>
<body>
{{if .Value}}
Hi {{.Value.RealName}}.
<form method="post" action="/logout">
<input type="submit" name="method" value="logout" />
</form>
You will logout after 10 seconds. Then try to reload.
{{else}}
{{if .Msg}}<b>{{.Msg}}</b>{{end}}
<form method="post" action="/login">
<label for="name">Name:</label><br />
<input type="text" id="userid" name="userid" value="" /><br />
<label for="password">Password:</label><br />
<input type="password" id="password" name="password" value="" /><br />
<input type="submit" name="method" value="login" />
</form>
{{end}}
</body>
</html>
`

var tmpl = template.Must(template.New("x").Parse(page))
var logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)
var manager = session.NewSessionManager(logger)

type User struct {
	UserId   string
	Password string
	RealName string
	Age      int64
}

func getSession(ctx *web.Context, manager *session.SessionManager) *session.Session {
	id, _ := ctx.GetSecureCookie("SessionId")
	session := manager.GetSessionById(id)
	ctx.SetSecureCookie("SessionId", session.Id, int64(manager.GetTimeout()))
	ctx.SetHeader("Pragma", "no-cache", true)
	return session
}

func getParam(ctx *web.Context, name string) string {
	value, found := ctx.Params[name]
	if found {
		return strings.Trim(value, " ")
	}
	return ""
}

func dbSetup() {
	if _, e := os.Stat(dbfile); e != nil {
		db, e := sql.Open("sqlite3", dbfile)
		if e != nil {
			logger.Print(e)
			return
		}
		for _, s := range []string {
			"create table User (userid varchar(16), password varchar(20), realname varchar(20), age integer)",
			"insert into User values('go', 'lang', 'golang', 3)",
			"insert into User values('perl', 'monger', 'perlmonger', 20)",
			"insert into User values('japan', 'hello', '日本', 10)",
		} {
			if _, e := db.Exec(s); e != nil {
				logger.Print(e)
				return
			}
		}
		db.Close()
	}
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
	dbSetup()

	//------------------------------------------------
	// go to web
	web.Config.CookieSecret = "7C19QRmwf3mHZ9CPAaPQ0hsWeufKd"
	s := "select userid, password, realname, age from User where userid = ? and password = ?"

	web.Get("/", func(ctx *web.Context) {
		session := getSession(ctx, manager)
		tmpl.Execute(ctx, map[string]interface{} {
			"Value": session.Value, "Msg": "",
		})
	})
	web.Post("/login", func(ctx *web.Context) {
		session := getSession(ctx, manager)
		userid := getParam(ctx, "userid")
		password := getParam(ctx, "password")
		if userid != "" && password != "" {
			// find user
			db, e := sql.Open("sqlite3", dbfile)
			defer db.Close()
			st, _ := db.Prepare(s)
			r, e := st.Query(userid, password)
			if e != nil {
				logger.Print(e)
				return
			}
			if !r.Next() {
				// not found
				tmpl.Execute(ctx, map[string]interface{} {
					"Value": nil, "Msg": "User not found",
				})
				return
			}
			var userid, password, realname string
			var age int64
			e = r.Scan(&userid, &password, &realname, &age)
			if e != nil {
				logger.Print(e)
				return
			}
			// store User object to sessino
			session.Value = &User{userid, password, realname, age}
			logger.Printf("User \"%s\" login", session.Value.(*User).UserId)
		}
		ctx.Redirect(302, "/")
	})
	web.Post("/logout", func(ctx *web.Context) {
		session := getSession(ctx, manager)
		if session.Value != nil {
			// abandon
			logger.Printf("User \"%s\" logout", session.Value.(*User).UserId)
			session.Abandon()
		}
		ctx.Redirect(302, "/")
	})
	web.Run(":6061")
}
