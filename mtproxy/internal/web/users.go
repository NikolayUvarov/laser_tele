package web

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/config"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/logx"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/proxy"
)

type userView struct {
	config.User
	Page   string     `json:"page"` // path of the personal page
	Link   proxy.Link `json:"link"`
	Active int        `json:"active"`
}

func (s *Server) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	cfg := s.app.Config()
	active := map[string]int{}
	for _, u := range s.app.Users().Report().Users {
		active[strings.ToLower(u.Name)] = u.Active
	}
	host := linkHost(r)
	views := make([]userView, 0, len(cfg.Users))
	for _, u := range cfg.Users {
		views = append(views, userView{User: u, Page: "p/" + config.PageToken(u.Secret),
			Link: s.app.UserLink(u, host), Active: active[strings.ToLower(u.Name)]})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"users": views, "public_host": cfg.PublicHost})
}

func (s *Server) handlePutUsers(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Users []config.User `json:"users"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Users == nil {
		writeError(w, http.StatusBadRequest, "bad request")
		return
	}
	cfg := s.app.Config()
	cfg.Users = req.Users
	if _, err := s.app.Apply(cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	logx.Infof("[%s] web: users changed", clientIP(r))
	s.handleGetUsers(w, r)
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.app.Users().Report())
}

func (s *Server) handleReportCSV(w http.ResponseWriter, r *http.Request) {
	report := s.app.Users().Report()
	var buf bytes.Buffer
	buf.WriteString("\ufeff") // so spreadsheet programs read UTF-8
	cw := csv.NewWriter(&buf)
	cw.Comma = ';'
	cw.Write([]string{"user", "max_connections", "allow_external", "active", "active_ips", "peak", "peak_at",
		"connections", "refused_limit", "refused_disabled", "refused_untrusted", "bytes_up", "bytes_down", "last_seen", "last_ip"})
	ts := func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Local().Format("2006-01-02 15:04:05")
	}
	for _, u := range report.Users {
		cw.Write([]string{u.Name, strconv.Itoa(u.MaxConnections), strconv.FormatBool(u.AllowExternal),
			strconv.Itoa(u.Active), strings.Join(u.ActiveIPs, " "), strconv.Itoa(u.Peak), ts(u.PeakAt),
			strconv.FormatInt(u.Connections, 10), strconv.FormatInt(u.RefusedLimit, 10),
			strconv.FormatInt(u.RefusedDisabled, 10), strconv.FormatInt(u.RefusedUntrusted, 10),
			strconv.FormatInt(u.BytesUp, 10), strconv.FormatInt(u.BytesDown, 10), ts(u.LastSeen), u.LastIP})
	}
	cw.Flush()
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="laser-mtproxy-report-%s.csv"`,
		time.Now().Format("2006-01-02")))
	w.Write(buf.Bytes())
}

func (s *Server) handleReportReset(w http.ResponseWriter, r *http.Request) {
	s.app.Users().Reset()
	logx.Infof("[%s] web: user statistics reset", clientIP(r))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// userByToken finds the user of a personal page
func (s *Server) userByToken(token string) (config.User, bool) {
	if len(token) != 24 {
		return config.User{}, false
	}
	for _, u := range s.app.Config().Users {
		if config.PageToken(u.Secret) == token {
			return u, true
		}
	}
	return config.User{}, false
}

var pageTexts = map[string]map[string]string{
	"ru": {
		"title": "Прокси Telegram", "hello": "Подключение к прокси Telegram", "user": "Пользователь",
		"scan": "Наведите камеру телефона на код или нажмите кнопку на устройстве, где установлен Telegram.",
		"open": "Открыть в Telegram", "copy": "Копировать ссылку", "copied": "Скопировано", "png": "Скачать QR (PNG)",
		"manual": "Вручную", "manual_path": "Настройки → Данные и память (на компьютере: Продвинутые настройки → Тип подключения) → Прокси → Добавить прокси → MTProto",
		"server": "Сервер", "port": "Порт", "secret": "Секрет",
		"lan_only": "Работает только из локальной сети.", "external": "Работает из любой сети.",
		"limit": "Одновременных подключений: до", "disabled": "Доступ выключен. Обратитесь к администратору.",
		"private": "Не пересылайте эту страницу и ссылку: по ним подключится кто угодно от вашего имени.",
	},
	"en": {
		"title": "Telegram proxy", "hello": "Connect to the Telegram proxy", "user": "User",
		"scan": "Point the phone camera at the code, or tap the button on a device with Telegram installed.",
		"open": "Open in Telegram", "copy": "Copy link", "copied": "Copied", "png": "Download QR (PNG)",
		"manual": "Manually", "manual_path": "Settings → Data and Storage (on desktop: Advanced → Connection type) → Proxy → Add proxy → MTProto",
		"server": "Server", "port": "Port", "secret": "Secret",
		"lan_only": "Works only from the local network.", "external": "Works from any network.",
		"limit": "Simultaneous connections: up to", "disabled": "Access is disabled. Contact the administrator.",
		"private": "Don't forward this page or the link: anyone who has them connects as you.",
	},
}

var personalPage = template.Must(template.New("p").Parse(`<!doctype html>
<html lang="{{.Lang}}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex">
<title>{{.T.title}}</title>
<link rel="icon" href="../favicon.svg" type="image/svg+xml">
<link rel="stylesheet" href="../style.css">
<script src="../personal.js" defer></script>
</head>
<body>
<main class="wrap personal">
  <div class="card">
    <h1>{{.T.hello}}</h1>
    <p class="muted">{{.T.user}}: <b>{{.Name}}</b></p>
    {{if .Disabled}}
      <p class="banner banner-error">{{.T.disabled}}</p>
    {{else}}
      <div class="qr-box"><img src="{{.Token}}/qr.svg" alt="QR" width="280" height="280"></div>
      <p class="hint center">{{.T.scan}}</p>
      <div class="link-actions center">
        <a class="btn btn-primary" href="{{.TG}}">{{.T.open}}</a>
        <button class="btn" type="button" data-copy="{{.Link.TMe}}" data-done="{{.T.copied}}">{{.T.copy}}</button>
        <a class="btn" href="{{.Token}}/qr.png" download="telegram-proxy-{{.Name}}.png">{{.T.png}}</a>
      </div>
      <div class="codebox"><code>{{.Link.TMe}}</code></div>
      <h2>{{.T.manual}}</h2>
      <p class="hint">{{.T.manual_path}}</p>
      <dl class="kv">
        <dt>{{.T.server}}</dt><dd class="mono">{{.Link.Server}}</dd>
        <dt>{{.T.port}}</dt><dd class="mono">{{.Link.Port}}</dd>
        <dt>{{.T.secret}}</dt><dd class="mono">{{.Link.Secret}}</dd>
      </dl>
      <p class="hint">{{if .External}}{{.T.external}}{{else}}{{.T.lan_only}}{{end}} {{.T.limit}} {{.Max}}.</p>
      <p class="hint">{{.T.private}}</p>
    {{end}}
  </div>
</main>
</body>
</html>
`))

func (s *Server) pageLang(r *http.Request) string {
	lang := r.URL.Query().Get("lang")
	if _, ok := pageTexts[lang]; ok {
		return lang
	}
	if lang = s.app.Config().Web.Language; lang == "en" {
		return lang
	}
	return "ru"
}

func (s *Server) handlePersonalPage(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	u, ok := s.userByToken(token)
	if !ok {
		http.NotFound(w, r)
		return
	}
	lang := s.pageLang(r)
	link := s.app.UserLink(u, linkHost(r))
	data := map[string]interface{}{
		"Lang": lang, "T": pageTexts[lang], "Name": u.Name, "Token": token, "Disabled": u.MaxConnections == 0,
		"Link": link, "TG": template.URL(link.TG), "External": u.AllowExternal, "Max": u.MaxConnections,
	}
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	personalPage.Execute(w, data)
}

func (s *Server) qrFor(w http.ResponseWriter, r *http.Request) (barcode.Barcode, string, bool) {
	token := r.PathValue("token")
	u, ok := s.userByToken(token)
	if !ok || u.MaxConnections == 0 {
		http.NotFound(w, r)
		return nil, "", false
	}
	code, err := qr.Encode(s.app.UserLink(u, linkHost(r)).TMe, qr.M, qr.Auto)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, "", false
	}
	return code, u.Name, true
}

const qrQuiet = 4 // modules of the white border required by the standard

func (s *Server) handleQRSVG(w http.ResponseWriter, r *http.Request) {
	code, _, ok := s.qrFor(w, r)
	if !ok {
		return
	}
	size := code.Bounds().Dx()
	full := size + 2*qrQuiet
	var path strings.Builder
	for y := 0; y < size; y++ {
		for x := 0; x < size; {
			if !isDark(code, x, y) {
				x++
				continue
			}
			start := x
			for x < size && isDark(code, x, y) {
				x++
			}
			fmt.Fprintf(&path, "M%d %dh%dv1h-%dz", start+qrQuiet, y+qrQuiet, x-start, x-start)
		}
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprintf(w, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges">`+
		`<rect width="%d" height="%d" fill="#fff"/><path fill="#000" d="%s"/></svg>`, full, full, full, full, path.String())
}

func (s *Server) handleQRPNG(w http.ResponseWriter, r *http.Request) {
	code, _, ok := s.qrFor(w, r)
	if !ok {
		return
	}
	const scale = 10
	size := code.Bounds().Dx()
	img := image.NewGray(image.Rect(0, 0, (size+2*qrQuiet)*scale, (size+2*qrQuiet)*scale))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if isDark(code, x, y) {
				cell := image.Rect((x+qrQuiet)*scale, (y+qrQuiet)*scale, (x+qrQuiet+1)*scale, (y+qrQuiet+1)*scale)
				draw.Draw(img, cell, image.NewUniform(color.Black), image.Point{}, draw.Src)
			}
		}
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	png.Encode(w, img)
}

func isDark(code barcode.Barcode, x, y int) bool {
	r, _, _, _ := code.At(x, y).RGBA()
	return r < 0x8000
}
