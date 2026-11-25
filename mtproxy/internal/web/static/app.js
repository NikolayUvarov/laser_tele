"use strict";

const I18N = {
  ru: {
    "tab.users": "Пользователи",
    "tab.report": "Отчёт",
    "connect.to_users": "Пользователи, QR-коды и ссылки",
    "connect.users_hint": "У каждого пользователя свой секрет и личная страница с QR-кодом. Адрес страницы можно отправить человеку.",
    "connect.users_summary": "Пользователей: {total}, включено: {enabled}, подключений сейчас: {active}.",
    "hint.secrets_moved": "Секреты задаются для каждого пользователя на вкладке «Пользователи».",
    "users.title": "Пользователи MTProto",
    "users.hint": "Лимит: 0 — пользователь выключен, N — до N одновременных подключений. Одно приложение Telegram держит несколько подключений (основное и отдельные для загрузки медиа), поэтому ставьте с запасом, например 10–20 на устройство; пик виден в отчёте.",
    "users.add": "Добавить пользователя",
    "users.edit": "Изменить пользователя «{name}»",
    "users.add_btn": "Добавить",
    "users.save_btn": "Сохранить",
    "users.name": "Имя",
    "users.max": "Одновременных подключений (0 — выключен)",
    "users.external": "Пускать не только из доверенных сетей",
    "users.comment": "Комментарий",
    "users.secret": "Секрет (пусто — создать автоматически)",
    "users.col_name": "Имя",
    "users.col_now": "Сейчас / лимит",
    "users.col_where": "Откуда",
    "users.col_comment": "Комментарий",
    "users.col_actions": "Действия",
    "users.where_lan": "доверенные сети",
    "users.where_any": "любые адреса",
    "users.off": "выключен",
    "users.page": "QR и ссылка",
    "users.copy_page": "Адрес страницы",
    "users.edit_btn": "Изменить",
    "users.new_secret": "Новый секрет",
    "users.delete": "Удалить",
    "users.none": "Пользователей нет. Добавьте первого ниже.",
    "users.confirm_secret": "Создать новый секрет для «{name}»? Старая ссылка и страница перестанут работать, его подключения оборвутся.",
    "users.confirm_delete": "Удалить пользователя «{name}»? Его подключения оборвутся.",
    "users.saved": "Сохранено, изменения уже действуют.",
    "users.page_copied": "Адрес личной страницы скопирован",
    "users.no_public": "У пользователей с доступом снаружи в ссылке стоит адрес этой страницы. Если снаружи он недоступен, укажите «Внешний адрес в ссылках» в настройках.",
    "err.user_name": "Укажите имя",
    "err.user_max": "Лимит: целое число от 0",
    "report.title": "Отчёт по пользователям",
    "report.csv": "Скачать CSV",
    "report.reset": "Сбросить статистику",
    "report.since": "Статистика с {since}. Обновляется каждые 5 секунд.",
    "report.confirm_reset": "Обнулить статистику всех пользователей?",
    "report.user": "Пользователь",
    "report.now": "Сейчас / лимит",
    "report.peak": "Пик",
    "report.connections": "Подключений",
    "report.refused": "Отказы: лимит / выключен / не из сети",
    "report.traffic": "Трафик ↑ / ↓",
    "report.last": "Последнее подключение",
    "report.ips": "Адреса сейчас",
    "report.total": "Итого",
    "report.never": "не подключался",
    "theme.toggle": "Тема: авто / светлая / тёмная",
    "btn.logout": "Выйти",
    "btn.login": "Войти",
    "btn.restart": "Перезапустить прокси",
    "btn.generate": "Создать",
    "btn.reset": "Отменить изменения",
    "btn.save": "Сохранить и перезапустить",
    "btn.cancel": "Отмена",
    "btn.copy": "Копировать",
    "login.title": "Вход",
    "login.username": "Логин",
    "login.password": "Пароль",
    "login.reason": "Ваш адрес {ip} не входит в доверенные сети, нужен пароль.",
    "login.reason_host": "Страница открыта по имени хоста, которого нет среди разрешённых (защита от DNS rebinding). Откройте её по IP-адресу или добавьте имя в «Дополнительные имена хоста» в настройках веб-интерфейса.",
    "login.no_password": "Пароль не задан, поэтому с вашего адреса {ip} вход невозможен. Задайте пароль из доверенной сети.",
    "login.admin_blocked": "Администрирование доступно только из локальной сети, а ваш адрес {ip} в неё не входит.",
    "f.admin_trusted_only": "Администрирование только из доверенных сетей (вход по паролю снаружи запрещён)",
    "tab.connect": "Подключение",
    "tab.stats": "Статистика",
    "tab.settings": "Настройки",
    "tab.tests": "Проверка",
    "tab.logs": "Логи",
    "status.running": "работает",
    "status.stopped": "остановлен",
    "status.error": "ошибка",
    "connect.state": "Состояние",
    "connect.access": "Доступ",
    "connect.apps": "Приложения Telegram (MTProto)",
    "connect.bots": "Боты (Bot API через шлюз)",
    "k.proxy": "MTProto-прокси",
    "k.listen": "Слушает",
    "k.uptime": "Работает",
    "k.gateway": "Шлюз Bot API",
    "k.fake_tls": "Fake TLS",
    "k.dc": "DC → IP",
    "k.cf_domains": "CF-доменов в пуле",
    "k.version": "Версия",
    "k.your_ip": "Ваш адрес",
    "k.trusted": "Доверенные сети",
    "k.password": "Пароль",
    "v.trusted": "доверенный",
    "v.untrusted": "не доверенный, вход по паролю",
    "v.set": "задан",
    "v.not_set": "не задан",
    "v.off": "выключен",
    "v.disabled": "выключен",
    "v.direct_only": "нет (только CF/TCP)",
    "gw.off": "Шлюз выключен. Включите его в настройках, чтобы бот на laser_tele мог работать через этот сервер.",
    "gw.error": "Шлюз не запустился: {error}",
    "gw.intro": "Боты работают по HTTPS (Bot API), а не по MTProto, поэтому для них сервер поднимает HTTP CONNECT и SOCKS5 прокси. Туннели открываются только к адресам: {allowed}.",
    "gw.code": "В коде бота:",
    "gw.env": "Или через переменную окружения:",
    "gw.auth": "С адресов вне доверенных сетей добавьте логин и пароль: http://{user}:ПАРОЛЬ@{addr}",
    "howto.title": "Как подключить Telegram вручную",
    "howto.desktop": "Telegram Desktop: Настройки → Продвинутые настройки → Тип подключения → Прокси → Добавить прокси → MTProto.",
    "howto.mobile": "Android / iOS: Настройки → Данные и память → Прокси → Добавить прокси → MTProto. Проще всего открыть на телефоне личную страницу пользователя и нажать «Открыть в Telegram».",
    "howto.fields": "Сервер, порт и секрет есть на личной странице пользователя: вкладка «Пользователи» → «QR и ссылка».",
    "howto.media": "Не грузятся фото и видео? Оставьте в «DC → IP» только 4:149.154.167.220 или очистите поле (вкладка «Настройки»).",
    "howto.ipv6": "Если в логах ошибки подключения по IPv6, выключите в настройках прокси Telegram «Попытка соединения через IPv6».",
    "stats.title": "Статистика с запуска процесса",
    "stats.auto": "обновляется каждые 5 секунд",
    "st.connections_total": "Подключений всего",
    "st.connections_active": "Активных",
    "st.connections_ws": "Через WS-пул",
    "st.connections_h2": "Через HTTP/2",
    "st.connections_cfproxy": "Через Cloudflare",
    "st.connections_tcp_fallback": "Прямой TCP",
    "st.connections_fronting": "WS с подменой SNI (fronting)",
    "st.connections_bad": "Неверный секрет",
    "st.connections_untrusted": "Отклонено: пользователь только для локальной сети",
    "st.connections_masked": "Переадресовано на сайт-маску (Fake TLS)",
    "st.ws_errors": "Ошибок WS",
    "st.bytes_up": "Отправлено",
    "st.bytes_down": "Получено",
    "st.pool": "Попаданий в WS-пул",
    "st.cf_pool": "Попаданий в пул CF Worker",
    "st.h2_requests": "HTTP/2 запросов",
    "st.h2_errors": "HTTP/2 ошибок",
    "st.h2_tcp_connections": "HTTP/2 соединений",
    "st.gateway_total": "Шлюз: подключений",
    "st.gateway_active": "Шлюз: активных",
    "st.gateway_denied": "Шлюз: отказано",
    "st.gateway_via_cf": "Шлюз: через CF Worker",
    "st.gateway_failures": "Шлюз: ошибок соединения",
    "st.gateway_bytes": "Шлюз: трафик ↑/↓",
    "sec.mtproto": "Подключение MTProto",
    "sec.dc": "Датацентры Telegram (DC → IP)",
    "sec.cfproxy": "Cloudflare Proxy",
    "sec.cfworker": "Cloudflare Worker",
    "sec.access": "Доступ",
    "sec.gateway": "Шлюз Bot API (HTTP CONNECT и SOCKS5)",
    "sec.web": "Веб-интерфейс",
    "sec.logs": "Логи",
    "f.host": "IP-адрес",
    "f.port": "Порт",
    "f.public_host": "Внешний адрес в ссылках",
    "f.fake_tls_domain": "Fake TLS домен (ee-секрет)",
    "f.proxy_protocol": "PROXY protocol (за nginx/haproxy)",
    "f.dc_ip": "По одному правилу на строку, формат номер:IP",
    "f.pool_size": "Пул WebSocket-сессий на DC",
    "f.buf_kb": "Буфер сокета, КБ",
    "f.cfproxy": "Включить CF-прокси для недоступных DC",
    "f.h2": "Медиа через Cloudflare по общему HTTP/2-соединению",
    "f.cf_custom": "Свои домены вместо автоматического выбора",
    "f.cf_domains": "Свои домены (через запятую)",
    "f.cf_url": "Адрес списка доменов (обновляется раз в час, пусто = только встроенный список)",
    "f.worker_enabled": "Использовать Cloudflare Worker",
    "f.worker_domains": "Домены Worker (через запятую)",
    "f.no_secure": "Выключить TLS для CF-прокси и CF Worker (порт 80)",
    "f.force_test_dc": "Весь трафик на тестовые DC",
    "f.trusted": "Доверенные сети: без пароля (по одной на строку)",
    "f.username": "Логин",
    "f.new_password": "Новый пароль",
    "f.clear_password": "Удалить пароль",
    "f.gw_enabled": "Включить шлюз",
    "f.gw_allowed": "Разрешённые адреса назначения (host:port, *.domain:port; по одному на строку)",
    "f.gw_via_worker": "Если прямое соединение не удалось, идти через CF Worker",
    "f.language": "Язык по умолчанию",
    "f.gw_fallback": "Запасные IP хостов (host=IP, IP; по одному хосту на строку)",
    "tip.gw_fallback": "Если адрес хоста из DNS недоступен, шлюз пробует эти IP. Фронтенды Telegram отдают и api.telegram.org, TLS остаётся сквозным",
    "err.fallback": "Неверная строка host=IP: {entry}",
    "f.tls_cert": "TLS-сертификат (файл)",
    "f.tls_key": "TLS-ключ (файл)",
    "f.web_allowed": "Дополнительные имена хоста веб-интерфейса (через запятую)",
    "f.verbose": "Подробное логирование (verbose)",
    "f.log_file": "Файл лога (пусто = не писать)",
    "f.log_max_mb": "Макс. размер лога, МБ",
    "ph.auto": "автоматически",
    "ph.off": "выключено",
    "ph.unchanged": "оставить прежний",
    "ph.cf_domains": "свои CF-домены через запятую",
    "tip.host": "Адрес, на котором прокси принимает подключения. 0.0.0.0 — все интерфейсы",
    "tip.port": "Порт прокси, тот же порт указывается в Telegram",
    "tip.public_host": "Адрес сервера в ссылках пользователей, которых пускают снаружи. Пусто — адрес, по которому открыта страница",
    "tip.fake_tls": "Маскировка под TLS к этому сайту (ee-секрет). Подключения с неверным секретом уходят на этот сайт",
    "tip.dc": "Номер DC и IP для прямого WebSocket-маршрута. Остальные DC идут через Cloudflare или TCP",
    "tip.pool": "Сколько готовых WebSocket-соединений держать на DC. 0 отключает прямой маршрут",
    "tip.buf_kb": "Размер буфера приёма/передачи сокета в килобайтах",
    "tip.h2": "Одно HTTP/2-соединение на датацентр и домен для всех загрузок медиа. Нужны CF-прокси и TLS.",
    "tip.trusted": "Ключевые слова: loopback (этот компьютер), lan (частные подсети интерфейсов этой машины, публичные не входят), private (все частные адреса), any (все адреса, пароль не спрашивается никогда). Также CIDR и отдельные IP.",
    "tip.web_restart": "Адрес и TLS веб-интерфейса применяются после перезапуска процесса.",
    "hint.iface": "Сейчас «lan» — это: {nets}",
    "hint.password_set": "Пароль задан.",
    "hint.password_none": "Пароль не задан: шлюз и вход по паролю с недоверенных адресов недоступны.",
    "msg.saved": "Сохранено, прокси перезапущен.",
    "msg.saved_web": "Сохранено. Адрес веб-интерфейса изменится после перезапуска процесса.",
    "msg.restarted": "Прокси перезапущен",
    "msg.copied": "Скопировано",
    "msg.copy_failed": "Не удалось скопировать",
    "msg.invalid": "Проверьте выделенные поля",
    "msg.reset": "Изменения отменены",
    "err.secret": "Секрет: ровно 32 hex-символа",
    "err.port": "Порт: число от 1 до 65535",
    "err.ip": "Нужен IP-адрес",
    "err.dc": "Неверная строка DC:IP: {entry}",
    "test.title": "Проверка маршрутов",
    "test.secure": "С TLS (порт 443)",
    "test.hint": "Проверка открывает WebSocket к каждому DC. Поля заполнены доменами из настроек.",
    "test.direct": "Прямой маршрут DC → IP",
    "test.cfproxy": "CF-прокси: свои домены",
    "test.cfauto": "CF-прокси: встроенные",
    "test.cfworker": "CF Worker",
    "test.running": "Проверяю…",
    "test.ok_all": "Все {total} DC доступны",
    "test.partial": "Доступно {ok} из {total}",
    "test.none": "Недоступно",
    "test.no_domains": "Укажите домены",
    "test.via": "маршрут",
    "test.result": "Результат",
    "test.auto_best": "Встроенные домены",
    "logs.title": "Лог",
    "logs.follow": "Следить",
    "logs.clear": "Очистить экран",
    "logs.download": "Скачать файл",
    "time.d": "д",
    "time.h": "ч",
    "time.m": "мин",
    "time.s": "с",
    "connect.access_closed": "С доверенных адресов пароль не нужен. Снаружи MTProto пускает только пользователей с отметкой «Пускать не только из доверенных сетей», шлюз требует пароль, администрирование закрыто.",
    "connect.access_password": "С доверенных адресов пароль не нужен. Снаружи MTProto пускает только пользователей с отметкой «Пускать не только из доверенных сетей», а шлюз и администрирование требуют пароль.",
    "st.connections_limited": "Отклонено: лимит или пользователь выключен",
  },
  en: {
    "tab.users": "Users",
    "tab.report": "Report",
    "connect.to_users": "Users, QR codes and links",
    "connect.users_hint": "Every user has their own secret and a personal page with a QR code. You can send the page address to the person.",
    "connect.users_summary": "Users: {total}, enabled: {enabled}, connections now: {active}.",
    "hint.secrets_moved": "Each user's secret is set on the Users tab.",
    "users.title": "MTProto users",
    "users.hint": "Limit: 0 disables the user, N allows up to N simultaneous connections. One Telegram application keeps several connections (the main one and more for media downloads), so leave room, for example 10 to 20 per device; the report shows the peak.",
    "users.add": "Add user",
    "users.edit": "Edit user \"{name}\"",
    "users.add_btn": "Add",
    "users.save_btn": "Save",
    "users.name": "Name",
    "users.max": "Simultaneous connections (0 = disabled)",
    "users.external": "Allow from outside the trusted networks",
    "users.comment": "Comment",
    "users.secret": "Secret (leave empty to generate)",
    "users.col_name": "Name",
    "users.col_now": "Now / limit",
    "users.col_where": "From",
    "users.col_comment": "Comment",
    "users.col_actions": "Actions",
    "users.where_lan": "trusted networks",
    "users.where_any": "any address",
    "users.off": "disabled",
    "users.page": "QR and link",
    "users.copy_page": "Page address",
    "users.edit_btn": "Edit",
    "users.new_secret": "New secret",
    "users.delete": "Delete",
    "users.none": "No users. Add the first one below.",
    "users.confirm_secret": "Generate a new secret for \"{name}\"? The old link and page will stop working, and the user's connections will be closed.",
    "users.confirm_delete": "Delete user \"{name}\"? Their connections will be closed.",
    "users.saved": "Saved. The changes are already in effect.",
    "users.page_copied": "Address of the personal page copied",
    "users.no_public": "Links of users allowed from outside contain the address of this page. If it isn't reachable from outside, set \"External address in links\" in Settings.",
    "err.user_name": "Enter a name",
    "err.user_max": "Limit: an integer from 0",
    "report.title": "Report by user",
    "report.csv": "Download CSV",
    "report.reset": "Reset statistics",
    "report.since": "Statistics since {since}. Refreshed every 5 seconds.",
    "report.confirm_reset": "Reset statistics of all users?",
    "report.user": "User",
    "report.now": "Now / limit",
    "report.peak": "Peak",
    "report.connections": "Connections",
    "report.refused": "Refused: limit / disabled / outside",
    "report.traffic": "Traffic ↑ / ↓",
    "report.last": "Last connection",
    "report.ips": "Addresses now",
    "report.total": "Total",
    "report.never": "never connected",
    "theme.toggle": "Theme: auto / light / dark",
    "btn.logout": "Log out",
    "btn.login": "Log in",
    "btn.restart": "Restart proxy",
    "btn.generate": "Generate",
    "btn.reset": "Discard changes",
    "btn.save": "Save and restart",
    "btn.cancel": "Cancel",
    "btn.copy": "Copy",
    "login.title": "Log in",
    "login.username": "Username",
    "login.password": "Password",
    "login.reason": "Your address {ip} isn't in the trusted networks. Log in with the password.",
    "login.reason_host": "The page was opened by a host name that isn't allowed (DNS rebinding protection). Open it by IP address or add the name to \"Extra host names\" in the web interface settings.",
    "login.no_password": "No password is set, so you can't log in from {ip}. Set one from a trusted network.",
    "login.admin_blocked": "Administration is available only from the local network. Your address {ip} isn't in it.",
    "f.admin_trusted_only": "Administration only from trusted networks (no password login from outside)",
    "tab.connect": "Connect",
    "tab.stats": "Statistics",
    "tab.settings": "Settings",
    "tab.tests": "Tests",
    "tab.logs": "Logs",
    "status.running": "running",
    "status.stopped": "stopped",
    "status.error": "error",
    "connect.state": "State",
    "connect.access": "Access",
    "connect.apps": "Telegram applications (MTProto)",
    "connect.bots": "Bots (Bot API through the gateway)",
    "k.proxy": "MTProto proxy",
    "k.listen": "Listening",
    "k.uptime": "Uptime",
    "k.gateway": "Bot API gateway",
    "k.fake_tls": "Fake TLS",
    "k.dc": "DC → IP",
    "k.cf_domains": "CF domains in the pool",
    "k.version": "Version",
    "k.your_ip": "Your address",
    "k.trusted": "Trusted networks",
    "k.password": "Password",
    "v.trusted": "trusted",
    "v.untrusted": "not trusted, logged in with the password",
    "v.set": "set",
    "v.not_set": "not set",
    "v.off": "off",
    "v.disabled": "disabled",
    "v.direct_only": "none (CF/TCP only)",
    "gw.off": "The gateway is disabled. Enable it in settings so a laser_tele bot can work through this server.",
    "gw.error": "The gateway didn't start: {error}",
    "gw.intro": "Bots use HTTPS (Bot API), not MTProto, so the server runs an HTTP CONNECT and SOCKS5 proxy for them. Tunnels go only to: {allowed}.",
    "gw.code": "In the bot code:",
    "gw.env": "Or with the environment variable:",
    "gw.auth": "From outside the trusted networks, add the username and password: http://{user}:PASSWORD@{addr}",
    "howto.title": "How to set up Telegram manually",
    "howto.desktop": "Telegram Desktop: Settings → Advanced → Connection type → Proxy → Add proxy → MTProto.",
    "howto.mobile": "Android / iOS: Settings → Data and Storage → Proxy → Add proxy → MTProto. The simplest way is to open the user's personal page on the phone and tap \"Open in Telegram\".",
    "howto.fields": "The server, port and secret are on the user's personal page: Users tab → \"QR and link\".",
    "howto.media": "Photos and videos don't load? Leave only 4:149.154.167.220 in \"DC → IP\" or clear the field (Settings tab).",
    "howto.ipv6": "If the log shows IPv6 connection errors, turn off \"Try connecting through IPv6\" in the Telegram proxy settings.",
    "stats.title": "Statistics since the process start",
    "stats.auto": "refreshed every 5 seconds",
    "st.connections_total": "Connections total",
    "st.connections_active": "Active",
    "st.connections_ws": "Through WS pool",
    "st.connections_h2": "Through HTTP/2",
    "st.connections_cfproxy": "Through Cloudflare",
    "st.connections_tcp_fallback": "Direct TCP",
    "st.connections_fronting": "WS with domain fronting",
    "st.connections_bad": "Wrong secret",
    "st.connections_untrusted": "Refused: LAN-only user from outside",
    "st.connections_masked": "Relayed to the masking site (Fake TLS)",
    "st.ws_errors": "WS errors",
    "st.bytes_up": "Sent",
    "st.bytes_down": "Received",
    "st.pool": "WS pool hits",
    "st.cf_pool": "CF worker pool hits",
    "st.h2_requests": "HTTP/2 requests",
    "st.h2_errors": "HTTP/2 errors",
    "st.h2_tcp_connections": "HTTP/2 connections",
    "st.gateway_total": "Gateway: connections",
    "st.gateway_active": "Gateway: active",
    "st.gateway_denied": "Gateway: denied",
    "st.gateway_via_cf": "Gateway: via CF worker",
    "st.gateway_failures": "Gateway: connect errors",
    "st.gateway_bytes": "Gateway: traffic ↑/↓",
    "sec.mtproto": "MTProto connection",
    "sec.dc": "Telegram datacenters (DC → IP)",
    "sec.cfproxy": "Cloudflare Proxy",
    "sec.cfworker": "Cloudflare Worker",
    "sec.access": "Access",
    "sec.gateway": "Bot API gateway (HTTP CONNECT and SOCKS5)",
    "sec.web": "Web interface",
    "sec.logs": "Logs",
    "f.host": "IP address",
    "f.port": "Port",
    "f.public_host": "External address in links",
    "f.fake_tls_domain": "Fake TLS domain (ee-secret)",
    "f.proxy_protocol": "PROXY protocol (behind nginx/haproxy)",
    "f.dc_ip": "One rule per line, format number:IP",
    "f.pool_size": "WebSocket pool per DC",
    "f.buf_kb": "Socket buffer, KB",
    "f.cfproxy": "Enable CF proxy for unavailable DCs",
    "f.h2": "Media through Cloudflare over a shared HTTP/2 connection",
    "f.cf_custom": "Own domains instead of the automatic choice",
    "f.cf_domains": "Own domains (comma separated)",
    "f.cf_url": "URL of the domain list (refreshed every hour, empty = built-in list only)",
    "f.worker_enabled": "Use Cloudflare Worker",
    "f.worker_domains": "Worker domains (comma separated)",
    "f.no_secure": "Disable TLS for CF proxy and CF worker (port 80)",
    "f.force_test_dc": "All traffic to test DCs",
    "f.trusted": "Trusted networks: no password (one per line)",
    "f.username": "Username",
    "f.new_password": "New password",
    "f.clear_password": "Remove password",
    "f.gw_enabled": "Enable gateway",
    "f.gw_allowed": "Allowed destinations (host:port, *.domain:port; one per line)",
    "f.gw_via_worker": "If direct connections fail, go through the CF Worker",
    "f.language": "Default language",
    "f.gw_fallback": "Fallback IPs of hosts (host=IP, IP; one host per line)",
    "tip.gw_fallback": "If the DNS address of a host is unreachable, the gateway tries these IPs. Telegram frontends serve api.telegram.org too; TLS stays end to end",
    "err.fallback": "Wrong host=IP line: {entry}",
    "f.tls_cert": "TLS certificate (file)",
    "f.tls_key": "TLS key (file)",
    "f.web_allowed": "Extra host names of the web interface (comma separated)",
    "f.verbose": "Verbose logging",
    "f.log_file": "Log file (empty = don't write)",
    "f.log_max_mb": "Max log size, MB",
    "ph.auto": "automatic",
    "ph.off": "off",
    "ph.unchanged": "keep current",
    "ph.cf_domains": "own CF domains, comma separated",
    "tip.host": "Address the proxy listens on. 0.0.0.0 means all interfaces",
    "tip.port": "Proxy port, the same port is set in Telegram",
    "tip.public_host": "Server address in links of users allowed from outside. Empty means the address the page is opened with",
    "tip.fake_tls": "Disguise traffic as TLS to this site (ee-secret). Connections with a wrong secret are relayed to the site",
    "tip.dc": "DC number and IP for the direct WebSocket route. Other DCs go through Cloudflare or TCP",
    "tip.pool": "How many ready WebSocket connections to keep per DC. 0 disables the direct route",
    "tip.buf_kb": "Socket receive/send buffer in kilobytes",
    "tip.h2": "One HTTP/2 connection per datacenter and domain carries all media downloads. Needs the CF proxy and TLS.",
    "tip.trusted": "Keywords: loopback (this computer), lan (private subnets of this machine's interfaces; public ones aren't included), private (all private addresses), any (every address; the password is never asked). CIDR ranges and single IPs work too.",
    "tip.web_restart": "Address and TLS of the web interface are applied after the process restarts.",
    "hint.iface": "Now \"lan\" is: {nets}",
    "hint.password_set": "The password is set.",
    "hint.password_none": "No password: the gateway and password login don't work from untrusted addresses.",
    "msg.saved": "Saved, the proxy restarted.",
    "msg.saved_web": "Saved. The address of the web interface changes after the process restarts.",
    "msg.restarted": "Proxy restarted",
    "msg.copied": "Copied",
    "msg.copy_failed": "Couldn't copy",
    "msg.invalid": "Check the highlighted fields",
    "msg.reset": "Changes discarded",
    "err.secret": "Secret: exactly 32 hex characters",
    "err.port": "Port: a number from 1 to 65535",
    "err.ip": "IP address expected",
    "err.dc": "Wrong DC:IP line: {entry}",
    "test.title": "Route tests",
    "test.secure": "With TLS (port 443)",
    "test.hint": "A test opens a WebSocket to every DC. The fields are filled with domains from the settings.",
    "test.direct": "Direct route DC → IP",
    "test.cfproxy": "CF proxy: own domains",
    "test.cfauto": "CF proxy: built-in",
    "test.cfworker": "CF Worker",
    "test.running": "Testing…",
    "test.ok_all": "All {total} DCs are available",
    "test.partial": "{ok} of {total} available",
    "test.none": "Unavailable",
    "test.no_domains": "Enter domains",
    "test.via": "route",
    "test.result": "Result",
    "test.auto_best": "Built-in domains",
    "logs.title": "Log",
    "logs.follow": "Follow",
    "logs.clear": "Clear screen",
    "logs.download": "Download file",
    "time.d": "d",
    "time.h": "h",
    "time.m": "min",
    "time.s": "s",
    "connect.access_closed": "Trusted addresses need no password. From outside, MTProto lets in only users allowed from outside, the gateway asks for the password, and administration is closed.",
    "connect.access_password": "Trusted addresses need no password. From outside, MTProto lets in only users allowed from outside; the gateway and administration ask for the password.",
    "st.connections_limited": "Refused: limit or disabled user",
  },
};

const $ = (sel, root = document) => root.querySelector(sel);
const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

const store = {
  get(key) { try { return localStorage.getItem(key); } catch { return null; } },
  set(key, value) { try { localStorage.setItem(key, value); } catch { /* private mode */ } },
};

const state = {
  lang: new URLSearchParams(location.search).get("lang") || store.get("lmp.lang") || null,
  session: null,
  status: null,
  configView: null,
  activeTab: "connect",
  users: null,
  editingUser: -1,
  logSeq: 0,
  logEntries: [],
  timers: [],
};

function t(key, params) {
  const dict = I18N[state.lang] || I18N.ru;
  let s = dict[key] ?? I18N.ru[key] ?? key;
  if (params) for (const [k, v] of Object.entries(params)) s = s.split("{" + k + "}").join(v);
  return s;
}

function applyI18n() {
  document.documentElement.lang = state.lang;
  $$("[data-i18n]").forEach((el) => { el.textContent = t(el.dataset.i18n); });
  $$("[data-i18n-ph]").forEach((el) => { el.placeholder = t(el.dataset.i18nPh); });
  $$("[data-i18n-title]").forEach((el) => { el.title = t(el.dataset.i18nTitle); });
  $("#lang-toggle").textContent = state.lang === "ru" ? "EN" : "RU";
}

function el(tag, attrs = {}, ...children) {
  const node = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (v === undefined || v === null || v === false) continue;
    if (k === "class") node.className = v;
    else if (k === "text") node.textContent = v;
    else if (k.startsWith("on")) node.addEventListener(k.slice(2), v);
    else node.setAttribute(k, v === true ? "" : v);
  }
  for (const c of children.flat()) {
    if (c === null || c === undefined || c === false) continue;
    node.append(c instanceof Node ? c : document.createTextNode(String(c)));
  }
  return node;
}

async function api(method, path, body) {
  const opts = { method, headers: { "X-Laser-Mtproxy": "1" }, credentials: "same-origin" };
  if (body !== undefined) {
    opts.headers["Content-Type"] = "application/json";
    opts.body = JSON.stringify(body);
  }
  const resp = await fetch(path, opts);
  let data = null;
  try { data = await resp.json(); } catch { data = null; }
  if (resp.status === 401 && path !== "api/login") {
    showLogin();
    throw new Error(t("login.title"));
  }
  if (!resp.ok) throw new Error((data && data.error) || resp.statusText);
  return data;
}

let toastTimer = null;
function toast(msg) {
  const node = $("#toast");
  node.textContent = msg;
  node.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { node.hidden = true; }, 2200);
}

async function copyText(text) {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
    } else {
      const area = el("textarea", { class: "mono" });
      area.value = text;
      document.body.append(area);
      area.select();
      const ok = document.execCommand("copy");
      area.remove();
      if (!ok) throw new Error("copy");
    }
    toast(t("msg.copied"));
  } catch {
    toast(t("msg.copy_failed"));
  }
}

function humanBytes(n) {
  const units = ["B", "KB", "MB", "GB", "TB"];
  let v = n, i = 0;
  while (Math.abs(v) >= 1024 && i < units.length - 1) { v /= 1024; i++; }
  return (i === 0 ? v : v.toFixed(1)) + " " + units[i];
}

function humanDuration(sec) {
  sec = Math.max(0, Math.floor(sec));
  const d = Math.floor(sec / 86400), h = Math.floor(sec % 86400 / 3600), m = Math.floor(sec % 3600 / 60), s = sec % 60;
  if (d) return `${d} ${t("time.d")} ${h} ${t("time.h")}`;
  if (h) return `${h} ${t("time.h")} ${m} ${t("time.m")}`;
  if (m) return `${m} ${t("time.m")} ${s} ${t("time.s")}`;
  return `${s} ${t("time.s")}`;
}

/* ---------- theme ---------- */
function applyTheme(mode) {
  if (mode === "light" || mode === "dark") document.documentElement.dataset.theme = mode;
  else delete document.documentElement.dataset.theme;
}
function cycleTheme() {
  const order = ["auto", "light", "dark"];
  const cur = store.get("lmp.theme") || "auto";
  const next = order[(order.indexOf(cur) + 1) % order.length];
  store.set("lmp.theme", next);
  applyTheme(next);
}

/* ---------- session ---------- */
function showLogin() {
  stopTimers();
  $("#view-app").hidden = true;
  $("#view-login").hidden = false;
  $("#logout").hidden = true;
  const s = state.session || {};
  let reason = t("login.reason", { ip: s.client_ip || "?" });
  if (s.trusted_ip && !s.host_allowed) reason = t("login.reason_host");
  else if (s.admin_blocked) reason = t("login.admin_blocked", { ip: s.client_ip || "?" });
  else if (!s.has_password) reason = t("login.no_password", { ip: s.client_ip || "?" });
  $("#login-reason").textContent = reason;
  // the form is useless when the password can't be accepted from this address
  for (const id of ["#login-user", "#login-pass"]) $(id).closest("label").hidden = !!s.admin_blocked;
  $("#login-form button[type=submit]").hidden = !!s.admin_blocked;
  if (s.username) $("#login-user").value = s.username;
  setPill("muted", "…");
}

async function loadSession() {
  state.session = await (await fetch("api/session", { credentials: "same-origin" })).json();
  if (state.lang !== "ru" && state.lang !== "en") state.lang = state.session.language === "en" ? "en" : "ru";
  applyI18n();
  if (!state.session.authorized) { showLogin(); return; }
  $("#view-login").hidden = true;
  $("#view-app").hidden = false;
  $("#logout").hidden = state.session.trusted;
  await refreshStatus();
  await loadConfig();
  startTimers();
}

/* ---------- status ---------- */
function setPill(kind, text) {
  const pill = $("#status-pill");
  pill.className = "pill pill-" + kind;
  pill.textContent = text;
}

async function refreshStatus() {
  try {
    state.status = await api("GET", "api/status");
  } catch { return; }
  renderStatus();
  if (state.activeTab === "stats") renderStats();
}

function kv(list, items) {
  list.replaceChildren(...items.flatMap(([k, v]) => [el("dt", { text: k }), el("dd", {}, v)]));
}

function codebox(text) {
  return el("div", { class: "codebox" }, el("code", { text }),
    el("button", { class: "btn btn-sm", type: "button", text: t("btn.copy"), onclick: () => copyText(text) }));
}

function renderStatus() {
  const st = state.status;
  if (!st) return;
  const now = Date.now();
  if (st.running) setPill("ok", t("status.running"));
  else setPill("err", st.proxy_error ? t("status.error") : t("status.stopped"));

  const banner = $("#proxy-error");
  const errors = [st.proxy_error, st.gateway_error].filter(Boolean);
  banner.hidden = errors.length === 0;
  banner.textContent = errors.join(" · ");

  const uptime = st.running && st.proxy_started ? humanDuration((now - Date.parse(st.proxy_started)) / 1000) : "—";
  let gw = t("v.disabled");
  if (st.gateway_enabled) gw = st.gateway_running ? st.gateway_listen : t("status.error");
  kv($("#state-list"), [
    [t("k.proxy"), st.running ? t("status.running") : t("status.stopped")],
    [t("k.listen"), st.listen],
    [t("k.uptime"), uptime],
    [t("k.gateway"), gw],
    [t("k.fake_tls"), st.fake_tls_domain || t("v.off")],
    [t("k.dc"), (st.dc_ip && st.dc_ip.length) ? st.dc_ip.join(", ") : t("v.direct_only")],
    [t("k.cf_domains"), String(st.cf_domains || 0)],
    [t("k.version"), st.version],
  ]);

  const s = state.session;
  kv($("#access-list"), [
    [t("k.your_ip"), `${s.client_ip} (${s.trusted ? t("v.trusted") : t("v.untrusted")})`],
    [t("k.trusted"), el("span", { class: "mono small", text: (st.trusted_networks || []).join(", ") || "—" })],
    [t("k.password"), st.has_password ? t("v.set") : t("v.not_set")],
  ]);
  $("#users-summary").textContent = t("connect.users_summary", { total: st.users, enabled: st.users_enabled, active: st.users_active });
  const adminClosed = !state.configView || state.configView.config.web.admin_trusted_only !== false;
  $("#access-hint").textContent = t(adminClosed ? "connect.access_closed" : "connect.access_password");

  const gwCard = $("#gateway-card");
  if (!st.gateway_enabled) {
    gwCard.replaceChildren(el("p", { class: "muted", text: t("gw.off") }));
  } else {
    const addr = st.gateway_url.replace(/^http:\/\//, "");
    const user = (state.configView && state.configView.config.access.username) || "admin";
    const code = `laser_tele.DoLaserTeleInit(laser_tele.LaserTeleConfigT{\n\tProxy: "${st.gateway_url}",\n})`;
    gwCard.replaceChildren(...[
      st.gateway_error ? el("p", { class: "error", text: t("gw.error", { error: st.gateway_error }) }) : null,
      el("p", { text: t("gw.intro", { allowed: (st.gateway_allowed || []).join(", ") }) }),
      el("p", { class: "muted", text: t("gw.code") }),
      el("pre", { class: "snippet", text: code }),
      el("p", { class: "muted", text: t("gw.env") }),
      codebox(`TG_PROXY=${st.gateway_url}`),
      el("p", { class: "hint", text: t("gw.auth", { user, addr }) }),
    ].filter(Boolean));
  }
}

const STAT_KEYS = ["connections_total", "connections_active", "connections_ws", "connections_cfproxy", "connections_h2",
  "connections_tcp_fallback", "connections_fronting", "connections_bad", "connections_untrusted", "connections_limited", "connections_masked",
  "ws_errors", "h2_requests", "h2_errors", "h2_tcp_connections",
  "gateway_total", "gateway_active", "gateway_denied", "gateway_via_cf", "gateway_failures"];

function renderStats() {
  const st = state.status;
  if (!st) return;
  const s = st.stats;
  const ratio = (h, m) => (h + m ? `${h}/${h + m}` : "—");
  const tiles = [
    ["st.bytes_up", humanBytes(s.bytes_up)],
    ["st.bytes_down", humanBytes(s.bytes_down)],
    ["st.pool", ratio(s.pool_hits, s.pool_misses)],
    ["st.cf_pool", ratio(s.cf_pool_hits, s.cf_pool_misses)],
    ...STAT_KEYS.map((k) => ["st." + k, String(s[k] ?? 0)]),
    ["st.gateway_bytes", `${humanBytes(s.gateway_bytes_up)} / ${humanBytes(s.gateway_bytes_down)}`],
  ];
  $("#stat-grid").replaceChildren(...tiles.map(([k, v]) =>
    el("div", { class: "stat" }, el("div", { class: "v", text: v }), el("div", { class: "k", text: t(k) }))));
}

/* ---------- settings ---------- */
const form = () => $("#settings-form");
function field(name) { return form().elements.namedItem(name); }
function setVal(name, value) {
  const f = field(name);
  if (!f) return;
  if (f.type === "checkbox") f.checked = !!value;
  else f.value = value ?? "";
}
function getVal(name) {
  const f = field(name);
  return f.type === "checkbox" ? f.checked : f.value.trim();
}
const lines = (s) => s.split(/\r?\n/).map((x) => x.trim()).filter(Boolean);
const commaList = (s) => s.split(/[\s,;]+/).map((x) => x.trim()).filter(Boolean);

async function loadConfig() {
  try {
    state.configView = await api("GET", "api/config");
  } catch { return; }
  fillForm();
  renderStatus();
}

function fillForm() {
  const c = state.configView.config;
  for (const k of ["host", "port", "public_host", "fake_tls_domain", "proxy_protocol",
    "pool_size", "buf_kb", "cfproxy", "h2", "cfproxy_user_domain_enabled", "cfproxy_domains_url",
    "cfproxy_worker_enabled", "no_secure", "force_test_dc", "verbose", "log_file", "log_max_mb"]) setVal(k, c[k]);
  setVal("dc_ip", (c.dc_ip || []).join("\n"));
  setVal("cfproxy_user_domain", (c.cfproxy_user_domain || []).join(", "));
  setVal("cfproxy_worker_domain", (c.cfproxy_worker_domain || []).join(", "));
  setVal("trusted_networks", (c.access.trusted_networks || []).join("\n"));
  setVal("username", c.access.username);
  setVal("new_password", "");
  setVal("clear_password", false);
  setVal("gw_enabled", c.gateway.enabled);
  setVal("gw_host", c.gateway.host);
  setVal("gw_port", c.gateway.port);
  setVal("gw_allowed", (c.gateway.allowed_hosts || []).join("\n"));
  setVal("gw_via_worker", c.gateway.via_cfproxy_worker);
  setVal("gw_fallback", Object.entries(c.gateway.fallback_ips || {}).map(([h, ips]) => `${h}=${ips.join(", ")}`).join("\n"));
  setVal("web_host", c.web.host);
  setVal("web_port", c.web.port);
  setVal("web_language", c.web.language || "ru");
  setVal("web_tls_cert", c.web.tls_cert);
  setVal("web_tls_key", c.web.tls_key);
  setVal("web_allowed_hosts", (c.web.allowed_hosts || []).join(", "));
  setVal("web_admin_trusted_only", c.web.admin_trusted_only);
  $("#iface-nets").textContent = t("hint.iface", { nets: (state.configView.interface_networks || []).join(", ") || "—" });
  $("#password-state").textContent = state.configView.has_password ? t("hint.password_set") : t("hint.password_none");
  $$(".invalid", form()).forEach((n) => n.classList.remove("invalid"));
  $("#settings-msg").hidden = true;
  syncDependent();
  if (!$("#test-worker-domains").value) $("#test-worker-domains").value = (c.cfproxy_worker_domain || []).join(", ");
  if (!$("#test-cf-domains").value) $("#test-cf-domains").value = (c.cfproxy_user_domain || []).join(", ");
  $("#test-secure").checked = !c.no_secure;
}

function syncDependent() {
  const cf = getVal("cfproxy");
  field("h2").disabled = !cf || getVal("no_secure") || getVal("force_test_dc");
  field("cfproxy_user_domain").disabled = !getVal("cfproxy_user_domain_enabled");
  field("cfproxy_worker_domain").disabled = !getVal("cfproxy_worker_enabled");
  field("new_password").disabled = getVal("clear_password");
  for (const n of ["gw_host", "gw_port", "gw_allowed", "gw_fallback", "gw_via_worker"]) field(n).disabled = !getVal("gw_enabled");
}

const IPV4 = /^(25[0-5]|2[0-4]\d|1?\d?\d)(\.(25[0-5]|2[0-4]\d|1?\d?\d)){3}$/;
const isIP = (s) => IPV4.test(s) || (s.includes(":") && /^[0-9a-fA-F:.]+$/.test(s));
const isHex32 = (s) => /^[0-9a-fA-F]{32}$/.test(s);

function validateForm() {
  const errors = [];
  const bad = (name, msg) => { field(name).classList.add("invalid"); errors.push(msg); };
  $$(".invalid", form()).forEach((n) => n.classList.remove("invalid"));
  for (const n of ["host", "gw_host", "web_host"]) if (!isIP(getVal(n))) bad(n, t("err.ip"));
  for (const n of ["port", "gw_port", "web_port"]) {
    const p = Number(getVal(n));
    if (!Number.isInteger(p) || p < 1 || p > 65535) bad(n, t("err.port"));
  }
  for (const entry of lines(getVal("dc_ip"))) {
    const [dc, ip] = entry.split(":");
    if (!/^\d+$/.test(dc || "") || !IPV4.test((ip || "").trim())) { bad("dc_ip", t("err.dc", { entry })); break; }
  }
  for (const entry of lines(getVal("gw_fallback"))) {
    const [host, ips] = entry.split("=");
    if (!host || !ips || !commaList(ips).every(isIP)) { bad("gw_fallback", t("err.fallback", { entry })); break; }
  }
  return errors;
}

function parseFallback(text) {
  const result = {};
  for (const entry of lines(text)) {
    const [host, ips] = entry.split("=");
    result[host.trim().toLowerCase()] = commaList(ips || "");
  }
  return result;
}

function collectConfig() {
  const c = structuredClone(state.configView.config);
  c.host = getVal("host");
  c.port = Number(getVal("port"));
  c.public_host = getVal("public_host");
  delete c.secret;
  delete c.external_secret;
  c.fake_tls_domain = getVal("fake_tls_domain");
  c.proxy_protocol = getVal("proxy_protocol");
  c.dc_ip = lines(getVal("dc_ip"));
  c.pool_size = Number(getVal("pool_size"));
  c.buf_kb = Number(getVal("buf_kb"));
  c.cfproxy = getVal("cfproxy");
  c.h2 = getVal("h2");
  c.cfproxy_user_domain_enabled = getVal("cfproxy_user_domain_enabled");
  c.cfproxy_user_domain = commaList(getVal("cfproxy_user_domain"));
  c.cfproxy_domains_url = getVal("cfproxy_domains_url");
  c.cfproxy_worker_enabled = getVal("cfproxy_worker_enabled");
  c.cfproxy_worker_domain = commaList(getVal("cfproxy_worker_domain"));
  c.no_secure = getVal("no_secure");
  c.force_test_dc = getVal("force_test_dc");
  c.verbose = getVal("verbose");
  c.log_file = getVal("log_file");
  c.log_max_mb = Number(getVal("log_max_mb"));
  c.access.trusted_networks = lines(getVal("trusted_networks"));
  c.access.username = getVal("username");
  c.gateway.enabled = getVal("gw_enabled");
  c.gateway.host = getVal("gw_host");
  c.gateway.port = Number(getVal("gw_port"));
  c.gateway.allowed_hosts = lines(getVal("gw_allowed"));
  c.gateway.via_cfproxy_worker = getVal("gw_via_worker");
  c.gateway.fallback_ips = parseFallback(getVal("gw_fallback"));
  c.web.host = getVal("web_host");
  c.web.port = Number(getVal("web_port"));
  c.web.language = getVal("web_language");
  c.web.tls_cert = getVal("web_tls_cert");
  c.web.tls_key = getVal("web_tls_key");
  c.web.allowed_hosts = commaList(getVal("web_allowed_hosts"));
  c.web.admin_trusted_only = getVal("web_admin_trusted_only");
  return c;
}

function showSettingsMsg(kind, text) {
  const m = $("#settings-msg");
  m.className = "msg " + kind;
  m.textContent = text;
  m.hidden = false;
}

async function saveSettings(ev) {
  ev.preventDefault();
  const errors = validateForm();
  if (errors.length) { showSettingsMsg("err", t("msg.invalid") + ": " + errors[0]); return; }
  const btn = $("button[type=submit]", form());
  btn.disabled = true;
  try {
    const res = await api("PUT", "api/config", {
      config: collectConfig(),
      new_password: getVal("clear_password") ? "" : field("new_password").value,
      clear_password: getVal("clear_password"),
    });
    await loadConfig();
    await refreshStatus();
    showSettingsMsg("ok", res.web_restart_required ? t("msg.saved_web") : t("msg.saved"));
  } catch (e) {
    showSettingsMsg("err", e.message);
  } finally {
    btn.disabled = false;
  }
}

/* ---------- tests ---------- */
function resultTable(results, showVia) {
  return el("div", { class: "table-wrap" }, el("table", { class: "result-table" },
    el("thead", {}, el("tr", {}, el("th", { text: "DC" }), el("th", { text: t("test.result") }),
      showVia ? el("th", { text: t("test.via") }) : null)),
    el("tbody", {}, results.map((r) => el("tr", {},
      el("td", { text: "DC" + r.dc }),
      el("td", {}, r.ok ? el("span", { class: "ok-text", text: "OK" }) : el("span", { class: "err-text", text: r.error || "—" })),
      showVia ? el("td", { class: "muted", text: r.via || "" }) : null)))));
}

function summaryPill(ok, total) {
  if (ok === total && total > 0) return el("span", { class: "pill pill-ok", text: t("test.ok_all", { total }) });
  if (ok > 0) return el("span", { class: "pill pill-warn", text: t("test.partial", { ok, total }) });
  return el("span", { class: "pill pill-err", text: t("test.none") });
}

async function runTest(kind, button) {
  const out = $("#test-results");
  const secure = $("#test-secure").checked;
  const body = { kind, secure };
  if (kind === "cfproxy") body.domains = commaList($("#test-cf-domains").value);
  if (kind === "cfworker") body.domains = commaList($("#test-worker-domains").value);
  if (kind === "direct") body.dc_ip = lines(getVal("dc_ip"));
  if ((kind === "cfproxy" || kind === "cfworker") && !body.domains.length) { toast(t("test.no_domains")); return; }
  const card = el("div", { class: "card" }, el("p", { class: "muted", text: `${button.textContent}: ${t("test.running")}` }));
  out.prepend(card);
  $$("[data-test]").forEach((b) => { b.disabled = true; });
  try {
    const res = await api("POST", "api/test", body);
    const parts = [];
    if (res.domains) {
      for (const d of res.domains) {
        parts.push(el("div", { class: "card-head" }, el("h3", { text: d.domain }), summaryPill(d.ok_count, d.total)));
        parts.push(resultTable(d.results, false));
      }
    } else if (kind === "cfproxy_auto") {
      parts.push(el("div", { class: "card-head" }, el("h3", { text: t("test.auto_best") }), summaryPill(res.ok_count, res.total)));
      parts.push(resultTable(res.results, false));
    } else {
      const ok = res.results.filter((r) => r.ok).length;
      parts.push(el("div", { class: "card-head" }, el("h3", { text: button.textContent }), summaryPill(ok, res.results.length)));
      parts.push(resultTable(res.results, true));
    }
    card.replaceChildren(...parts);
  } catch (e) {
    card.replaceChildren(el("p", { class: "error", text: e.message }));
  } finally {
    $$("[data-test]").forEach((b) => { b.disabled = false; });
  }
}

/* ---------- logs ---------- */
const LEVELS = { DEBUG: 0, INFO: 1, WARN: 2, ERROR: 3 };
function renderLogs() {
  const min = LEVELS[$("#log-level").value];
  const view = $("#log-view");
  const nodes = state.logEntries.filter((e) => LEVELS[e.level] >= min).map((e) => {
    const time = new Date(e.time).toLocaleTimeString();
    return el("div", { class: "lv-" + e.level, text: `${time}  ${e.level.padEnd(5)}  ${e.msg}` });
  });
  view.replaceChildren(...nodes);
  if ($("#log-follow").checked) view.scrollTop = view.scrollHeight;
}

async function pollLogs() {
  try {
    const res = await api("GET", "api/logs?after=" + state.logSeq);
    if (res.entries.length) {
      state.logSeq = res.entries[res.entries.length - 1].seq;
      state.logEntries.push(...res.entries);
      if (state.logEntries.length > 3000) state.logEntries.splice(0, state.logEntries.length - 3000);
      if (state.activeTab === "logs") renderLogs();
    }
  } catch { /* shown by other requests */ }
}


/* ---------- users ---------- */
const userForm = () => $("#user-form");
const ufield = (name) => userForm().elements.namedItem(name);

async function loadUsers() {
  try {
    const res = await api("GET", "api/users");
    state.users = res.users;
    state.publicHost = res.public_host;
  } catch { return; }
  renderUsers();
}

function pageURL(u) {
  return new URL(u.page, location.origin + location.pathname.replace(/[^/]*$/, "")).href;
}

function renderUsers() {
  const table = $("#users-table");
  const users = state.users || [];
  $("#users-public-hint").hidden = !(users.some((u) => u.allow_external && u.max_connections > 0) && !state.publicHost);
  if (!users.length) {
    table.replaceChildren(el("tbody", {}, el("tr", {}, el("td", { class: "muted", text: t("users.none") }))));
    return;
  }
  table.replaceChildren(
    el("thead", {}, el("tr", {}, ["users.col_name", "users.col_now", "users.col_where", "users.col_comment", "users.col_actions"]
      .map((k) => el("th", { text: t(k) })))),
    el("tbody", {}, users.map((u, i) => el("tr", {},
      el("td", {}, el("b", { text: u.name })),
      el("td", { class: "num" }, u.max_connections === 0
        ? el("span", { class: "pill pill-muted", text: t("users.off") })
        : `${u.active} / ${u.max_connections}`),
      el("td", {}, el("span", { class: "pill " + (u.allow_external ? "pill-warn" : "pill-ok"),
        text: u.allow_external ? t("users.where_any") : t("users.where_lan") })),
      el("td", { class: "wrap-cell muted", text: u.comment || "" }),
      el("td", {}, el("div", { class: "row-actions" },
        u.max_connections > 0 ? el("a", { class: "btn btn-sm btn-primary", href: pageURL(u), target: "_blank", rel: "noopener", text: t("users.page") }) : null,
        el("button", { class: "btn btn-sm", type: "button", text: t("users.copy_page"),
          onclick: () => copyText(pageURL(u)).then(() => toast(t("users.page_copied"))) }),
        el("button", { class: "btn btn-sm", type: "button", text: t("users.edit_btn"), onclick: () => editUser(i) }),
        el("button", { class: "btn btn-sm", type: "button", text: t("users.new_secret"), onclick: () => newSecret(i) }),
        el("button", { class: "btn btn-sm", type: "button", text: t("users.delete"), onclick: () => deleteUser(i) })))))));
}

function plainUsers() {
  return (state.users || []).map((u) => ({ name: u.name, secret: u.secret, max_connections: u.max_connections,
    allow_external: u.allow_external, comment: u.comment || "" }));
}

async function saveUsers(list) {
  const res = await api("PUT", "api/users", { users: list });
  state.users = res.users;
  renderUsers();
  await loadConfig();
  refreshStatus();
}

function showUserMsg(kind, text) {
  const m = $("#user-msg");
  m.className = "msg " + kind;
  m.textContent = text;
  m.hidden = false;
}

function resetUserForm() {
  state.editingUser = -1;
  userForm().reset();
  ufield("max_connections").value = "20";
  $("#user-form-title").textContent = t("users.add");
  $("#user-submit").textContent = t("users.add_btn");
  $("#user-cancel").hidden = true;
  $$(".invalid", userForm()).forEach((n) => n.classList.remove("invalid"));
}

function editUser(i) {
  const u = state.users[i];
  state.editingUser = i;
  ufield("name").value = u.name;
  ufield("max_connections").value = String(u.max_connections);
  ufield("allow_external").checked = u.allow_external;
  ufield("comment").value = u.comment || "";
  ufield("secret").value = u.secret;
  $("#user-form-title").textContent = t("users.edit", { name: u.name });
  $("#user-submit").textContent = t("users.save_btn");
  $("#user-cancel").hidden = false;
  $("#user-msg").hidden = true;
  userForm().scrollIntoView({ behavior: "smooth" });
}

async function submitUser(ev) {
  ev.preventDefault();
  $$(".invalid", userForm()).forEach((n) => n.classList.remove("invalid"));
  const name = ufield("name").value.trim();
  const max = Number(ufield("max_connections").value);
  const secret = ufield("secret").value.trim().toLowerCase();
  if (!name) { ufield("name").classList.add("invalid"); showUserMsg("err", t("err.user_name")); return; }
  if (!Number.isInteger(max) || max < 0) { ufield("max_connections").classList.add("invalid"); showUserMsg("err", t("err.user_max")); return; }
  if (secret && !isHex32(secret)) { ufield("secret").classList.add("invalid"); showUserMsg("err", t("err.secret")); return; }
  const user = { name, secret, max_connections: max, allow_external: ufield("allow_external").checked, comment: ufield("comment").value.trim() };
  const list = plainUsers();
  if (state.editingUser >= 0) list[state.editingUser] = user;
  else list.push(user);
  try {
    await saveUsers(list);
    resetUserForm();
    showUserMsg("ok", t("users.saved"));
  } catch (e) {
    showUserMsg("err", e.message);
  }
}

async function newSecret(i) {
  const u = state.users[i];
  if (!confirm(t("users.confirm_secret", { name: u.name }))) return;
  const list = plainUsers();
  list[i].secret = "";
  try { await saveUsers(list); toast(t("users.saved")); } catch (e) { toast(e.message); }
}

async function deleteUser(i) {
  const u = state.users[i];
  if (!confirm(t("users.confirm_delete", { name: u.name }))) return;
  const list = plainUsers();
  list.splice(i, 1);
  try { await saveUsers(list); toast(t("users.saved")); } catch (e) { toast(e.message); }
}

/* ---------- report ---------- */
function fmtTime(s) {
  if (!s || s.startsWith("0001")) return "";
  return new Date(s).toLocaleString();
}

async function loadReport() {
  let r;
  try { r = await api("GET", "api/report"); } catch { return; }
  $("#report-since").textContent = t("report.since", { since: fmtTime(r.since) });
  const sum = { active: 0, connections: 0, rl: 0, rd: 0, ru: 0, up: 0, down: 0 };
  const rows = r.users.map((u) => {
    sum.active += u.active; sum.connections += u.connections; sum.rl += u.refused_limit;
    sum.rd += u.refused_disabled; sum.ru += u.refused_untrusted; sum.up += u.bytes_up; sum.down += u.bytes_down;
    const peak = u.peak ? `${u.peak}` + (u.peak_at && !u.peak_at.startsWith("0001") ? ` (${fmtTime(u.peak_at)})` : "") : "0";
    const last = u.last_seen && !u.last_seen.startsWith("0001") ? `${fmtTime(u.last_seen)}, ${u.last_ip}` : t("report.never");
    return el("tr", {},
      el("td", {}, el("b", { text: u.name }), u.allow_external ? el("span", { class: "muted small", text: " · " + t("users.where_any") }) : null),
      el("td", { class: "num" }, u.max_connections === 0 ? t("users.off") : `${u.active} / ${u.max_connections}`),
      el("td", { class: "num", text: peak }),
      el("td", { class: "num", text: String(u.connections) }),
      el("td", { class: "num", text: `${u.refused_limit} / ${u.refused_disabled} / ${u.refused_untrusted}` }),
      el("td", { class: "num", text: `${humanBytes(u.bytes_up)} / ${humanBytes(u.bytes_down)}` }),
      el("td", { class: "wrap-cell", text: last }),
      el("td", { class: "wrap-cell mono small", text: (u.active_ips || []).join(", ") }));
  });
  $("#report-table").replaceChildren(
    el("thead", {}, el("tr", {}, ["report.user", "report.now", "report.peak", "report.connections", "report.refused",
      "report.traffic", "report.last", "report.ips"].map((k) => el("th", { text: t(k) })))),
    el("tbody", {}, rows),
    el("tfoot", {}, el("tr", {},
      el("td", { text: t("report.total") }), el("td", { class: "num", text: String(sum.active) }), el("td", {}),
      el("td", { class: "num", text: String(sum.connections) }),
      el("td", { class: "num", text: `${sum.rl} / ${sum.rd} / ${sum.ru}` }),
      el("td", { class: "num", text: `${humanBytes(sum.up)} / ${humanBytes(sum.down)}` }), el("td", {}), el("td", {}))));
}

/* ---------- tabs and timers ---------- */
function showTab(name) {
  state.activeTab = name;
  $$(".tab").forEach((b) => b.classList.toggle("active", b.dataset.tab === name));
  $$(".panel").forEach((p) => { p.hidden = p.id !== "panel-" + name; });
  if (name === "stats") renderStats();
  if (name === "logs") { renderLogs(); pollLogs(); }
  if (name === "users") loadUsers();
  if (name === "report") loadReport();
  store.set("lmp.tab", name);
}

function startTimers() {
  stopTimers();
  state.timers.push(setInterval(refreshStatus, 5000));
  state.timers.push(setInterval(pollLogs, 2000));
  state.timers.push(setInterval(() => {
    if (state.activeTab === "report") loadReport();
    if (state.activeTab === "users") loadUsers();
  }, 5000));
  pollLogs();
}
function stopTimers() {
  state.timers.forEach(clearInterval);
  state.timers = [];
}

/* ---------- init ---------- */
document.addEventListener("DOMContentLoaded", () => {
  applyTheme(store.get("lmp.theme") || "auto");
  $("#theme-toggle").addEventListener("click", cycleTheme);
  $("#lang-toggle").addEventListener("click", () => {
    state.lang = state.lang === "ru" ? "en" : "ru";
    store.set("lmp.lang", state.lang);
    applyI18n();
    renderStatus();
    if (state.configView) fillForm();
    if (state.activeTab === "stats") renderStats();
    if (state.activeTab === "users") { renderUsers(); if (state.editingUser < 0) resetUserForm(); }
    if (state.activeTab === "report") loadReport();
  });
  $("#logout").addEventListener("click", async () => {
    try { await api("POST", "api/logout", {}); } catch { /* ignore */ }
    location.reload();
  });
  $("#login-form").addEventListener("submit", async (ev) => {
    ev.preventDefault();
    const err = $("#login-error");
    err.hidden = true;
    try {
      await api("POST", "api/login", { username: $("#login-user").value, password: $("#login-pass").value });
      $("#login-pass").value = "";
      await loadSession();
    } catch (e) {
      err.textContent = e.message;
      err.hidden = false;
    }
  });
  $$(".tab").forEach((b) => b.addEventListener("click", () => showTab(b.dataset.tab)));
  $("#restart").addEventListener("click", async (ev) => {
    ev.target.disabled = true;
    try { await api("POST", "api/restart", {}); toast(t("msg.restarted")); await refreshStatus(); }
    catch (e) { toast(e.message); }
    finally { ev.target.disabled = false; }
  });
  form().addEventListener("submit", saveSettings);
  form().addEventListener("change", syncDependent);
  $("#settings-reset").addEventListener("click", () => { fillForm(); toast(t("msg.reset")); });
  $$("[data-gen]").forEach((b) => b.addEventListener("click", async () => {
    try { setVal(b.dataset.gen, (await api("GET", "api/random-secret")).secret); } catch (e) { toast(e.message); }
  }));
  $$("[data-clear]").forEach((b) => b.addEventListener("click", () => setVal(b.dataset.clear, "")));
  $$("[data-test]").forEach((b) => b.addEventListener("click", () => runTest(b.dataset.test, b)));
  $$("[data-goto]").forEach((b) => b.addEventListener("click", () => showTab(b.dataset.goto)));
  userForm().addEventListener("submit", submitUser);
  $("#user-cancel").addEventListener("click", () => { resetUserForm(); $("#user-msg").hidden = true; });
  $("#user-gen").addEventListener("click", async () => {
    try { ufield("secret").value = (await api("GET", "api/random-secret")).secret; } catch (e) { toast(e.message); }
  });
  $("#report-reset").addEventListener("click", async () => {
    if (!confirm(t("report.confirm_reset"))) return;
    try { await api("POST", "api/report/reset", {}); loadReport(); } catch (e) { toast(e.message); }
  });
  $("#log-level").addEventListener("change", renderLogs);
  $("#log-clear").addEventListener("click", () => { state.logEntries = []; renderLogs(); });

  const hashTab = location.hash.slice(1);
  const savedTab = $(`.tab[data-tab="${hashTab}"]`) ? hashTab : store.get("lmp.tab");
  if (savedTab && $(`.tab[data-tab="${savedTab}"]`)) showTab(savedTab);
  loadSession().catch((e) => { setPill("err", e.message); });
});
