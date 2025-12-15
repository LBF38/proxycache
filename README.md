# ProxyCache

A simple HTTP Reverse Proxy with caching.

This is a small challenge for building a reverse proxy from scratch in Go.

> [!WARNING]
> This work is far from being perfect.
> It works but might not be fully compatible to HTTP standards on caching and other details.

## Goals

Here are the main goals of the project. It may evolve across the project's lifetime.

### Minimal

- HTTP reverse proxy
  - has minimal support for the following features:
    - all HTTP methods
    - forwarded headers (`X-Forwarded-*`)
    - streaming data
    - trailer headers
    - middlewares (custom or predefined, to extend the proxy behaviour)
  - enhancements:
    - support for more protocols (websocket, tcp, udp, HTTP/2, HTTP/3)
    - enhance the routing engine
    - entrypoints/middlewares/servers architecture
- Caching
  - current features:
    - bypass rules (partially supported, see tests + RFC for more) from requests or response and `Cache-Control` directives
    - cache response from method and status code (heuristical caching)
    - adapter pattern to add any cache implementation in the proxy (in-memory, redis, ...)
    - ETag (partially supported)
  - enhancements:
    - ETag (full support)
    - `Last-Modified` / `Expires`
    - `If-None-Match` / `If-Match`
    - Fresh/Stale using `Age` or `max-age`
    - Validation mechanism

### Enhancements

- Static configuration
  - => need to refactor the `Proxy` struct
- CLI
  - minimal version for now.
  - can be enhanced with config management + other flags for configuring the app
- Load balancing
  - would have to create a proper router between endpoints and servers
- Authentication
- Rate limit
- Observability

These could be implemented using middlewares.

## Inspirations/Ressources

Some ressources I found useful referencing to:

- <https://www.rfc-editor.org/rfc/rfc9110.html>
- <https://www.rfc-editor.org/rfc/rfc9111.html>
- <https://web.dev/articles/http-cache?hl=en>
- [Mozilla docs - HTTP Caching](https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/Caching)
- [FOSDEM 2019: How to write a reverse proxy with Go in 25 minutes](https://youtu.be/tWSmUsYLiE4)
- [Traefik Proxy](https://github.com/traefik/traefik)
- [Traefik EE docs](https://doc.traefik.io/traefik-enterprise/middlewares/http-cache/)
- ... (and more)
