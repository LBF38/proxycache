# ProxyCache

A simple HTTP Reverse Proxy with caching.

This is a small challenge for building a reverse proxy from scratch in Go.

## Goals

Here are the main goals of the project. It may evolve across the project's lifetime.

### Minimal

- HTTP reverse proxy
- Caching

### Enhancements

- Static configuration
- CLI
- Load balancing
- Rate limit
- Observability

## Inspirations/Ressources

Some ressources I found useful referencing to:

- <https://www.rfc-editor.org/rfc/rfc9110.html>
- <https://www.rfc-editor.org/rfc/rfc9111.html>
- [FOSDEM 2019: How to write a reverse proxy with Go in 25 minutes](https://youtu.be/tWSmUsYLiE4)
- [Traefik Proxy](https://github.com/traefik/traefik)
- [Traefik EE docs](https://doc.traefik.io/traefik-enterprise/middlewares/http-cache/)
- ... (and more)
