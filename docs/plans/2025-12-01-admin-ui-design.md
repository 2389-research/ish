# Admin UI Design (v1.5)

Consumer interface for viewing and managing fake Google API data.

## Decisions

| Decision | Choice |
|----------|--------|
| Interface | Web UI |
| Interactivity | htmx (partial updates) |
| Styling | Tailwind CSS (CDN) |
| CRUD scope | View + Create + Delete |
| URL path | `/admin/` |
| Templates | Go `html/template` |
| Build step | None (all CDN) |

## Architecture

```
ISH Server
├── /gmail/v1/...     → Gmail API handlers
├── /calendar/v3/...  → Calendar API handlers
├── /people/v1/...    → People API handlers
└── /admin/           → Admin UI handlers
    └── internal/admin/
        ├── handlers.go
        ├── templates.go
        └── templates/
```

## Routes

| Route | Method | Description |
|-------|--------|-------------|
| `/admin/` | GET | Dashboard with data counts |
| `/admin/gmail` | GET | List messages |
| `/admin/gmail/new` | GET | Create message form |
| `/admin/gmail` | POST | Create message (htmx) |
| `/admin/gmail/{id}` | DELETE | Delete message (htmx) |
| `/admin/calendar` | GET | List events |
| `/admin/calendar/new` | GET | Create event form |
| `/admin/calendar` | POST | Create event (htmx) |
| `/admin/calendar/{id}` | DELETE | Delete event (htmx) |
| `/admin/people` | GET | List contacts |
| `/admin/people/new` | GET | Create contact form |
| `/admin/people` | POST | Create contact (htmx) |
| `/admin/people/{id}` | DELETE | Delete contact (htmx) |

## htmx Behavior

- POST returns new row HTML (appended to table)
- DELETE returns empty (htmx removes row via `hx-swap="delete"`)
- No full page reloads for create/delete

## File Structure

```
internal/admin/
├── handlers.go      # Routes + handlers
├── templates.go     # Template loading
└── templates/
    ├── layout.html
    ├── dashboard.html
    ├── gmail/list.html
    ├── gmail/form.html
    ├── gmail/row.html
    ├── calendar/list.html
    ├── calendar/form.html
    ├── calendar/row.html
    ├── people/list.html
    ├── people/form.html
    └── people/row.html
```

## UI Layout

Dashboard shows counts and quick links. List pages show tables with delete buttons. Forms are simple HTML with htmx submit.

## Dependencies

- htmx via CDN: `https://unpkg.com/htmx.org`
- Tailwind via CDN: `https://cdn.tailwindcss.com`
