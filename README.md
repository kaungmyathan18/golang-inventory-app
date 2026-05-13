# golang-inventory-app

Generated HTTP service (`github.com/kaungmyathan18/golang-inventory-app`).

## Run locally

```bash
make tidy
make run
```

Live reload with [Air](https://github.com/air-verse/air) (install once: `go install github.com/air-verse/air@latest`):

```bash
make dev
```

## Docker Compose

```bash
docker compose -f compose.yaml up --build
```

### Observability stack (Prometheus, Grafana, Pyroscope, Tempo, OpenTelemetry Collector)

| Service | URL |
|---------|-----|
| Grafana | http://localhost:3000 — `admin` / `admin`; datasources and dashboards are provisioned automatically |
| Prometheus | http://localhost:9090 |
| Loki | http://localhost:3100 (container logs, queried from Grafana) |
| Pyroscope | http://localhost:4040 (continuous CPU/heap profiles from the app) |
| Tempo | trace backend (query from Grafana Explore or the **Traces overview** dashboard) |

With the stack enabled, the app sends OTLP to `otel-collector:4317` and profiles to Pyroscope via `PYROSCOPE_SERVER_ADDRESS` (set in `compose.yaml`).

### Logs in Grafana

Open Grafana → **Explore** → select **Loki**.

Useful LogQL queries:

```logql
{service="app"}
{service="app"} | json | level=~"warn|error|fatal|panic"
{compose_project=~".+"} |= "error"
```

The **HTTP service overview** dashboard includes request, latency, runtime, and app log panels. The **Observability overview** dashboard includes container log volume and cross-container error panels.

## CI/CD

GitHub Actions (`.github/workflows/ci.yml`) runs on push and pull requests: `go mod verify`, `go vet`, `go test`, a local **Docker** build, and **Trivy** scans (filesystem and image). [Dependabot](.github/dependabot.yml) opens weekly PRs for Go modules and Actions.

### ECS deploy

`.github/workflows/deploy-ecs.yml` is included because you enabled ECS during scaffolding. Configure **Actions** before running it:

| Kind | Name | Purpose |
|------|------|---------|
| Secret | `AWS_ROLE_ARN` | IAM role for GitHub OIDC (`sts:AssumeRoleWithWebIdentity`) |
| Variable | `AWS_REGION` | e.g. `us-east-1` |
| Variable | `ECR_REPOSITORY` | ECR repo name (not the full URI) |
| Variable | `ECS_CLUSTER` | ECS cluster name |
| Variable | `ECS_SERVICE` | ECS service name |

The workflow pushes an image tagged with the commit SHA and `:latest`, then calls `ecs update-service --force-new-deployment`. Your task definition should use the same ECR repository (typically the `:latest` tag). See [GitHub OIDC with AWS](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/configuring-openid-connect-in-amazon-web-services).

## Features

- Database: SQLite
- Cache: Redis
- Queue: none
- Observability: OTel Prometheus metrics structured logs (zap)
