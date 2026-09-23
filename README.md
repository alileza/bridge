# Bridge 
<img width="193" alt="bridge (1)" src="https://github.com/alileza/bridge/assets/1962129/3469338d-c2c5-4446-9dc4-669ee1dd6b24">


Bridge is a lightweight URL shortener server designed for simplicity, ease of operation, and minimal operational costs. With its minimalist design and efficient architecture, Bridge offers a straightforward solution for shortening URLs without the overhead of complex systems.

## Key Features 

- **Minimalist Design:** Bridge is built with simplicity in mind, offering essential features without unnecessary bloat.
  
- **Easy to Operate:** Setting up and managing Bridge is effortless, making it ideal for both beginners and experienced users alike.
  

## Getting Started 

To deploy Bridge and start shortening URLs, follow these simple steps:







```sh
$ go install .

$ bridge
2024/03/20 05:09:29 Storage: error reading file: open ./bridgedata/routes.json: no such file or directory
2024/03/20 05:09:29 Storage: creating new file: ./bridgedata/routes.json
portal: 2024/03/20 05:09:29 Listening on 0.0.0.0:8080
```

### In action


https://github.com/alileza/bridge/assets/1962129/e3da4868-2a72-40ae-876a-69560364d5a5




## Configuration

| Flag | Env | Default | Description |
| --- | --- | --- | --- |
| `--listen-address`, `-l` | `LISTEN_ADDRESS` | `0.0.0.0:80` | HTTP listen address |
| `--storage-dir`, `-s` | | `./bridgedata` | Routes are stored in `<storage-dir>.json` |
| `--metrics`, `-m` | `METRICS_ENABLED` | `false` | Expose Prometheus metrics at `/metrics` |
| `--metrics-address` | `METRICS_ADDRESS` | | Serve `/metrics` on a separate address instead (e.g. `0.0.0.0:9090`), keeping it off the public port |

`GET /healthz` returns `200` while the route storage is readable, `503` otherwise.

## Monitoring

With `--metrics` (or `--metrics-address`), bridge exposes Prometheus metrics with no extra dependencies:

- **Activity:** `bridge_redirects_total{host}`, `bridge_routes_forwarded_total{key}`, `bridge_redirect_misses_total{host}`, `bridge_route_changes_total{op}`
- **Health:** `bridge_storage_up`, `bridge_routes{host}`, `bridge_storage_errors_total{op}`, `bridge_build_info{version}`, `bridge_start_time_seconds`
- **HTTP:** `bridge_http_requests_total{handler,code}`, `bridge_http_request_duration_seconds{handler}`
- **Runtime:** `go_goroutines`, `go_memstats_heap_alloc_bytes`, `go_memstats_sys_bytes`, `go_gc_cycles_total`

A ready-made Grafana dashboard lives in [`contrib/grafana/bridge-dashboard.json`](contrib/grafana/bridge-dashboard.json). Import it and pick your Prometheus data source.

## Releasing

Every merge to `main` cuts a release automatically: a semver tag, a GitHub release with binaries, and a `ghcr.io/alileza/bridge` image.

- Default bump is **patch**; add the `minor` or `major` label to the PR to bump further.
- Put `[skip release]` in the merge commit message to skip a release.
- To release manually, run the **Release** workflow from the Actions tab and pick the bump.

## Contributing 

Contributions to Bridge are welcome! Whether it's bug fixes, feature enhancements, or documentation improvements, feel free to submit pull requests or open issues on the GitHub repository.

## Feedback and Support 📧

For any questions, feedback, or support inquiries, please don't hesitate to reach out through GitHub Issues or contact us directly at [bridge@alileza.me](mailto:bridge@alileza.me).

Start shortening your URLs effortlessly with Bridge today!
