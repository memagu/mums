<div align="center"> <img src="/web/static/icons/mums.svg" alt="mums logo"
   title="mums logo" width="256"> <h1>mums</h1> <h4> Serving beverages since
      1337! <br /> <a href="https://www.mumsa.nu/">mumsa.nu</a> </h4> </div>

## About

A web application designed to easily track and manage _mums_.

This project is built with simplicity at its core, deliberately avoiding the
build steps that are common in modern web development. It combines
[HTMX](https://htmx.org/) and [Tailwind CSS](https://tailwindcss.com/) on the
frontend with [Echo](https://echo.labstack.com/) and [SQLite](https://www.sqlite.org/)
on the backend to create a lightweight, server-driven architecture.

Nearly everything above the framework is built in-house: session management with
sliding expiry, role-based access control for both user accounts and per-group roles,
password-reset and invite tokens, the database access layer (plain SQL, no ORM), and a
database event bus that streams live page updates over SSE.

The dependencies are intentionally kept few and minimal, with no scaffolding or code
generation, so every layer stays small and readable, making the setup easy to run and
maintain.

## Development

### Prerequisites

- [Go](https://go.dev/) 1.26+

### Setup

1. Clone or fork this repository.

#### Optional setup

- Copy the environment template: `cp .env.example .env` and customize values.

### Running

1. From the project root, run `go run cmd/mums/main.go`; the app serves
   <http://127.0.0.1:11337>.
   > Alternatively, run `go tool air` for live reloading on file changes.

### Verifying

A change isn't done until all three are clean:

- `go build ./...`
- `gofmt -l .`
- `go vet ./...`

## License

[GPL-3.0](LICENSE)
