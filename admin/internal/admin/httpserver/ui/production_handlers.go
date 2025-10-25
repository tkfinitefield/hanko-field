package ui

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"

	custommw "finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	adminproduction "finitefield.org/hanko-admin/internal/admin/production"
	productiontpl "finitefield.org/hanko-admin/internal/admin/templates/production"
)

// ProductionQueuesPage renders the production kanban board page.
func (h *Handlers) ProductionQueuesPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	req := buildProductionBoardRequest(r)
	result, err := h.production.Board(ctx, user.Token, req.query)
	errMsg := ""
	if err != nil {
		log.Printf("production: fetch board failed: %v", err)
		errMsg = "制作ボードの取得に失敗しました。時間を置いて再度お試しください。"
		result = adminproduction.BoardResult{}
	}

	basePath := custommw.BasePathFromContext(ctx)
	data := productiontpl.BuildPageData(basePath, req.state, result, errMsg)

	templ.Handler(productiontpl.Index(data)).ServeHTTP(w, r)
}

// ProductionQueuesBoard renders the kanban fragment for HTMX swaps.
func (h *Handlers) ProductionQueuesBoard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	req := buildProductionBoardRequest(r)
	result, err := h.production.Board(ctx, user.Token, req.query)
	errMsg := ""
	if err != nil {
		log.Printf("production: fetch board fragment failed: %v", err)
		errMsg = "制作ボードの取得に失敗しました。"
		result = adminproduction.BoardResult{}
	}

	basePath := custommw.BasePathFromContext(ctx)
	board := productiontpl.BuildBoard(basePath, req.state, result, errMsg)

	if canonical := canonicalProductionURL(basePath, req); canonical != "" {
		w.Header().Set("HX-Push-Url", canonical)
	}

	templ.Handler(productiontpl.Board(board)).ServeHTTP(w, r)
}

// OrdersProductionEvent handles drag-and-drop submissions from the kanban board.
func (h *Handlers) OrdersProductionEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := custommw.UserFromContext(ctx)
	if !ok || user == nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		http.Error(w, "注文IDが不正です。", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "リクエストの解析に失敗しました。", http.StatusBadRequest)
		return
	}

	stageValue := strings.TrimSpace(r.FormValue("type"))
	if stageValue == "" {
		http.Error(w, "ステージが指定されていません。", http.StatusBadRequest)
		return
	}
	if !isValidStage(stageValue) {
		http.Error(w, "指定されたステージに移動できません。", http.StatusBadRequest)
		return
	}

	req := adminproduction.AppendEventRequest{
		Stage:    adminproduction.Stage(stageValue),
		Note:     strings.TrimSpace(r.FormValue("note")),
		Station:  strings.TrimSpace(r.FormValue("station")),
		ActorID:  user.UID,
		ActorRef: user.Email,
	}

	if _, err := h.production.AppendEvent(ctx, user.Token, orderID, req); err != nil {
		switch {
		case errors.Is(err, adminproduction.ErrCardNotFound):
			http.Error(w, "指定された注文が見つかりません。", http.StatusNotFound)
		case errors.Is(err, adminproduction.ErrStageInvalid):
			http.Error(w, "指定されたステージに移動できません。", http.StatusBadRequest)
		default:
			log.Printf("production: append event failed: %v", err)
			http.Error(w, "制作更新に失敗しました。", http.StatusBadGateway)
		}
		return
	}

	triggerToast(w, "制作ステージを更新しました。", "success")
	w.WriteHeader(http.StatusNoContent)
}

type productionBoardRequest struct {
	query adminproduction.BoardQuery
	state productiontpl.QueryState
}

func buildProductionBoardRequest(r *http.Request) productionBoardRequest {
	values := r.URL.Query()
	queue := strings.TrimSpace(values.Get("queue"))
	priority := strings.TrimSpace(values.Get("priority"))
	productLine := strings.TrimSpace(values.Get("product_line"))
	workstation := strings.TrimSpace(values.Get("workstation"))
	selected := strings.TrimSpace(values.Get("selected"))

	state := productiontpl.QueryState{
		Queue:       queue,
		Priority:    priority,
		ProductLine: productLine,
		Workstation: workstation,
		Selected:    selected,
		RawQuery:    rebuildRawQuery(values),
	}

	query := adminproduction.BoardQuery{
		QueueID:     queue,
		Priority:    priority,
		ProductLine: productLine,
		Workstation: workstation,
		Selected:    selected,
	}

	return productionBoardRequest{query: query, state: state}
}

func canonicalProductionURL(basePath string, req productionBoardRequest) string {
	base := joinBasePath(basePath, "/production/queues")
	if req.state.RawQuery == "" {
		return base
	}
	return base + "?" + req.state.RawQuery
}

func rebuildRawQuery(values url.Values) string {
	return values.Encode()
}

func isValidStage(stage string) bool {
	switch adminproduction.Stage(stage) {
	case adminproduction.StageQueued,
		adminproduction.StageEngraving,
		adminproduction.StagePolishing,
		adminproduction.StageQC,
		adminproduction.StagePacked:
		return true
	default:
		return false
	}
}
