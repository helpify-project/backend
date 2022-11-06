package controllers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/uptrace/bun"
	"go.uber.org/zap"

	"github.com/helpify-project/backend/internal/cctx"
	"github.com/helpify-project/backend/internal/jsonrpc"
	"github.com/helpify-project/backend/internal/router"
	"github.com/helpify-project/backend/internal/rpc"
)

var _ router.Controller = (*ChatController)(nil)

const (
	chatSessionCookieName = "chat_session"
	chatSupportCookieName = "chat_support"
)

var (
	wsPool = new(sync.Pool)
)

type ChatController struct {
	DB            *bun.DB
	SessionSecret string

	sessionKey  paseto.V4AsymmetricSecretKey
	tokenParser paseto.Parser
	upgrader    *websocket.Upgrader
	rpc         *rpc.Server
}

func (c *ChatController) handleChat(w http.ResponseWriter, r *http.Request) {
	// Ensure user has session cookie
	sid, wsHeader, err := c.getOrCreateChatSessionCookie(r)
	if err != nil {
		zap.L().Error("failed to get chat session cookie", zap.Error(err))
		return
	}

	conn, err := c.upgrader.Upgrade(w, r, wsHeader)
	if err != nil {
		zap.L().Error("failed to upgrade connection", zap.Error(err))
		return
	}

	r = c.prepareRequest(r, sid)
	c.rpc.HandleWebsocketConnection(r, conn)
}

func (c *ChatController) Register(router *mux.Router) {
	var err error
	if c.sessionKey, err = loadPasetoPrivateKey(c.SessionSecret); err != nil {
		zap.L().Error("failed to decode session private key, using random key", zap.Error(err))
	}

	c.tokenParser = paseto.MakeParser([]paseto.Rule{
		paseto.IssuedBy("helpify"),
		paseto.NotExpired(),
	})

	c.upgrader = &websocket.Upgrader{
		HandshakeTimeout:  10 * time.Second,
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		WriteBufferPool:   wsPool,
		EnableCompression: true,
		CheckOrigin: func(r *http.Request) bool {
			// TODO: need allowed domains from the configuration
			return true
		},
	}

	// Set up JSON-RPC services
	//log.Root().SetHandler(log.StderrHandler)

	c.rpc = rpc.NewServer()
	c.rpc.RegisterName("chat", jsonrpc.NewChatService(c.DB))
	c.rpc.RegisterName("room", jsonrpc.NewRoomService(c.DB))

	router.HandleFunc("/chat/ws", c.handleChat).Methods(http.MethodGet)

	// TODO: remove
	router.HandleFunc("/chat/rpc", func(w http.ResponseWriter, r *http.Request) {
		// Ensure user has session cookie
		sid, wsHeader, err := c.getOrCreateChatSessionCookie(r)
		if err != nil {
			zap.L().Error("failed to get chat session cookie", zap.Error(err))
			return
		}

		for k, vs := range wsHeader {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}

		c.rpc.ServeHTTP(w, c.prepareRequest(r, sid))
	})

	// XXX: easter egg, or lazy dev job?
	router.HandleFunc("/chat/iamsupport", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     chatSupportCookieName,
			Value:    "true",
			Path:     "/chat",
			Expires:  time.Now().Add(24 * time.Hour),
			SameSite: http.SameSiteStrictMode,
			HttpOnly: true,
			// TODO: need allowed domains from the configuration
			//Domain: r.URL.Hostname(),
			//Secure: r.URL.Scheme == "https",
		})

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	}).Methods(http.MethodGet)
}

func (c *ChatController) getOrCreateChatSessionCookie(r *http.Request) (sid string, newHeader http.Header, err error) {
	var token *paseto.Token

	// Try to get the cookie value
	var cookie *http.Cookie
	if cookie, err = r.Cookie(chatSessionCookieName); errors.Is(err, http.ErrNoCookie) {
		err = nil
	} else if err == nil {
		token, err = c.tokenParser.ParseV4Public(c.sessionKey.Public(), cookie.Value, nil)
		if err != nil {
			zap.L().Debug("invalid token", zap.Error(err))
		}

		// Ignore
		err = nil
	} else {
		// Propagate error
		return
	}

	// Attempt to get existing SID
	if token != nil {
		if sid, err = token.GetSubject(); err != nil {
			zap.L().Debug("failed to get sid from token", zap.Error(err))
		}
	}

	// Generate brand new SID if it's still empty
	if sid == "" {
		sid = strings.ReplaceAll(uuid.New().String(), "-", "")
	}

	// Create new token
	now := time.Now()
	expiresAt := now.Add(2 * time.Hour)
	token = newToken()
	token.SetIssuer("helpify")
	token.SetExpiration(expiresAt)
	token.SetIssuedAt(now)
	token.SetNotBefore(now)
	token.SetSubject(sid)
	token.SetAudience("user")

	cookie = &http.Cookie{
		Name:     chatSessionCookieName,
		Value:    token.V4Sign(c.sessionKey, nil),
		Path:     "/chat",
		Expires:  expiresAt.Add(24 * time.Hour), // XXX: Add 24 hours to work around time zones, because cookies suck. Best effort
		MaxAge:   2 * 60 * 60,
		SameSite: http.SameSiteNoneMode,
		HttpOnly: true,
		Secure:   true,
		// TODO: need allowed domains from the configuration
		//Domain: r.URL.Hostname(),
		//Secure: r.URL.Scheme == "https",
	}

	if err = cookie.Valid(); err != nil {
		return
	}

	newHeader = make(http.Header)
	newHeader.Add("Set-Cookie", cookie.String())
	return
}

func (c *ChatController) prepareRequest(r *http.Request, sid string) *http.Request {
	supportPersonnel := false

	var supportCookie *http.Cookie
	if supportCookie, _ = r.Cookie(chatSupportCookieName); supportCookie != nil {
		supportPersonnel = true
	}

	return r.WithContext(cctx.WithValues(
		r.Context(),
		cctx.SessionID, sid,
		cctx.SupportPersonnel, supportPersonnel,
	))
}

func loadPasetoPrivateKey(sessionSecret string) (key paseto.V4AsymmetricSecretKey, err error) {
	var decoded []byte
	if decoded, err = base64.StdEncoding.DecodeString(sessionSecret); err != nil {
		return
	}

	return paseto.NewV4AsymmetricSecretKeyFromBytes(decoded)
}

// XXX: paseto library is silly
func newToken() *paseto.Token {
	t := paseto.NewToken()
	return &t
}
