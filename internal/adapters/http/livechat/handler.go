package livechathttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"cashflow_backend/internal/domain/livechat"
	"cashflow_backend/internal/platform/response"
	livechatusecase "cashflow_backend/internal/usecase/livechat"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Simplified for development
	},
}

type Handler struct {
	useCase *livechatusecase.UseCase
	logger  *slog.Logger
}

func NewHandler(useCase *livechatusecase.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *Handler) GetChannel(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	channel, err := h.useCase.GetChannel(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, channel)
}

func (h *Handler) InitSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChannelID   int64  `json:"channel_id"`
		VisitorUUID string `json:"visitor_uuid"`
		Name        string `json:"name"`
		Email       string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	session, err := h.useCase.InitSession(r.Context(), req.ChannelID, req.VisitorUUID, req.Name, req.Email)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, session)
}

func (h *Handler) HandleWS(w http.ResponseWriter, r *http.Request) {
	sessionID, _ := strconv.ParseInt(chi.URLParam(r, "session_id"), 10, 64)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	h.logger.Info("websocket connected", "session_id", sessionID)

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			h.logger.Info("websocket disconnected", "session_id", sessionID)
			return
		}

		var wsReq struct {
			Body    string `json:"body"`
			FileURL string `json:"file_url"`
		}
		if err := json.Unmarshal(p, &wsReq); err != nil {
			continue
		}

		// Handle message (visitor sending to operator)
		msg, err := h.useCase.SendMessage(r.Context(), sessionID, livechat.SenderTypeVisitor, nil, wsReq.Body, wsReq.FileURL)
		if err != nil {
			continue
		}

		// Echo back or broadcast (simplified)
		resp, _ := json.Marshal(msg)
		if err := conn.WriteMessage(messageType, resp); err != nil {
			return
		}
	}
}

func (h *Handler) RateSession(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var req struct {
		Score   int    `json:"score"`
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.useCase.RateSession(r.Context(), id, req.Score, req.Comment); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) ListActiveSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.useCase.GetActiveSessions(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, sessions)
}

func (h *Handler) CloseSession(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.useCase.CloseSession(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

func (h *Handler) ConvertToTicket(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	ticketID, err := h.useCase.ConvertToTicket(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]int64{"ticket_id": ticketID})
}
