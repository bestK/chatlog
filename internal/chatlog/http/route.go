package http

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"github.com/sjzar/chatlog/internal/errors"
	"github.com/sjzar/chatlog/pkg/util"

	"github.com/gin-gonic/gin"
)

// EFS holds embedded file system data for static assets.
//
//go:embed static
var EFS embed.FS

// initRouter sets up routes and static file servers for the web service.
// It defines endpoints for API as well as serving static content.
func (s *Service) initRouter() {

	router := s.GetRouter()

	staticDir, _ := fs.Sub(EFS, "static")
	router.StaticFS("/static", http.FS(staticDir))
	router.StaticFileFS("/favicon.ico", "./favicon.ico", http.FS(staticDir))
	router.StaticFileFS("/", "./index.htm", http.FS(staticDir))

	// MCP Server
	{
		router.GET("/sse", s.mcp.HandleSSE)
		router.POST("/messages", s.mcp.HandleMessages)
		// mcp inspector is shit
		// https://github.com/modelcontextprotocol/inspector/blob/aeaf32f/server/src/index.ts#L155
		router.POST("/message", s.mcp.HandleMessages)
	}

	// API V1 Router
	api := router.Group("/api/v1")
	{
		api.GET("/chatlog", s.GetChatlog)
		api.GET("/contact", s.ListContact)
		api.GET("/chatroom", s.ListChatRoom)
		api.GET("/session", s.GetSession)
	}

	router.NoRoute(s.NoRoute)
}

// NoRoute handles 404 Not Found errors. If the request URL starts with "/api"
// or "/static", it responds with a JSON error. Otherwise, it redirects to the root path.
func (s *Service) NoRoute(c *gin.Context) {
	path := c.Request.URL.Path
	switch {
	case strings.HasPrefix(path, "/api"), strings.HasPrefix(path, "/static"):
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	default:
		c.Header("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate, value")
		c.Redirect(http.StatusFound, "/")
	}
}

func (s *Service) GetChatlog(c *gin.Context) {

	var err error
	start, end, ok := util.TimeRangeOf(c.Query("time"))
	if !ok {
		errors.Err(c, errors.ErrInvalidArg("time"))
	}

	var limit int
	if _limit := c.Query("limit"); len(_limit) > 0 {
		limit, err = strconv.Atoi(_limit)
		if err != nil {
			errors.Err(c, errors.ErrInvalidArg("limit"))
			return
		}
	}

	var offset int
	if _offset := c.Query("offset"); len(_offset) > 0 {
		offset, err = strconv.Atoi(_offset)
		if err != nil {
			errors.Err(c, errors.ErrInvalidArg("offset"))
			return
		}
	}

	talker := c.Query("talker")

	if limit < 0 {
		limit = 0
	}

	if offset < 0 {
		offset = 0
	}

	messages, err := s.db.GetMessages(start, end, talker, limit, offset)
	if err != nil {
		errors.Err(c, err)
		return
	}

	switch strings.ToLower(c.Query("format")) {
	case "csv":
	case "json":
		// json
		c.JSON(http.StatusOK, messages)
	default:
		// plain text
		c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Flush()

		for _, m := range messages {
			c.Writer.WriteString(m.PlainText(len(talker) == 0))
			c.Writer.WriteString("\n")
			c.Writer.Flush()
		}
	}
}

func (s *Service) ListContact(c *gin.Context) {
	list, err := s.db.ListContact()
	if err != nil {
		errors.Err(c, err)
		return
	}

	format := strings.ToLower(c.Query("format"))
	switch format {
	case "json":
		// json
		c.JSON(http.StatusOK, list)
	default:
		// csv
		if format == "csv" {
			// 浏览器访问时，会下载文件
			c.Writer.Header().Set("Content-Type", "text/csv; charset=utf-8")
		} else {
			c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Flush()

		c.Writer.WriteString("UserName,Alias,Remark,NickName\n")
		for _, contact := range list.Items {
			c.Writer.WriteString(fmt.Sprintf("%s,%s,%s,%s\n", contact.UserName, contact.Alias, contact.Remark, contact.NickName))
		}
		c.Writer.Flush()
	}
}

func (s *Service) ListChatRoom(c *gin.Context) {

	if query := c.Query("query"); len(query) > 0 {
		chatRoom := s.db.GetChatRoom(query)
		if chatRoom != nil {
			c.JSON(http.StatusOK, chatRoom)
			return
		}
	}

	list, err := s.db.ListChatRoom()
	if err != nil {
		errors.Err(c, err)
		return
	}
	format := strings.ToLower(c.Query("format"))
	switch format {
	case "json":
		// json
		c.JSON(http.StatusOK, list)
	default:
		// csv
		if format == "csv" {
			// 浏览器访问时，会下载文件
			c.Writer.Header().Set("Content-Type", "text/csv; charset=utf-8")
		} else {
			c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Flush()

		c.Writer.WriteString("Name,Owner,UserCount\n")
		for _, chatRoom := range list.Items {
			c.Writer.WriteString(fmt.Sprintf("%s,%s,%d\n", chatRoom.Name, chatRoom.Owner, len(chatRoom.Users)))
		}
		c.Writer.Flush()
	}
}

func (s *Service) GetSession(c *gin.Context) {

	var err error
	var limit int
	if _limit := c.Query("limit"); len(_limit) > 0 {
		limit, err = strconv.Atoi(_limit)
		if err != nil {
			errors.Err(c, errors.ErrInvalidArg("limit"))
			return
		}
	}

	sessions, err := s.db.GetSession(limit)
	if err != nil {
		errors.Err(c, err)
		return
	}
	format := strings.ToLower(c.Query("format"))
	switch format {
	case "csv":
		c.Writer.Header().Set("Content-Type", "text/csv; charset=utf-8")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Flush()

		c.Writer.WriteString("UserName,NOrder,NickName,Content,NTime\n")
		for _, session := range sessions.Items {
			c.Writer.WriteString(fmt.Sprintf("%s,%d,%s,%s,%s\n", session.UserName, session.NOrder, session.NickName, strings.ReplaceAll(session.Content, "\n", "\\n"), session.NTime))
		}
		c.Writer.Flush()
	case "json":
		// json
		c.JSON(http.StatusOK, sessions)
	default:
		c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Flush()
		for _, session := range sessions.Items {
			c.Writer.WriteString(session.PlainText(120))
			c.Writer.WriteString("\n")
		}
		c.Writer.Flush()
	}
}
