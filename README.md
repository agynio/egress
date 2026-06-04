# Egress Service

Egress is a gRPC service that owns EgressRule and EgressRuleAttachment
resources for the egress gateway v1 program. It stores rule configuration in
PostgreSQL, provisions OpenZiti services and dial policies via Ziti Management,
validates secret references through Secrets, and publishes cache-invalidation
events through Notifications.

## Build

```sh
make proto
go build ./...
```

## Run

```sh
export DATABASE_URL='postgres://user:pass@localhost:5432/egress?sslmode=disable'
export GRPC_ADDRESS=':50051'
go run ./cmd/egress
```

## Configuration

| Environment variable | Required | Default | Description |
| --- | --- | --- | --- |
| `DATABASE_URL` | Yes | - | PostgreSQL connection string. |
| `GRPC_ADDRESS` | No | `:50051` | gRPC listen address. |
| `ZITI_MANAGEMENT_ADDRESS` | No | `ziti-management:50051` | Ziti Management gRPC target. |
| `AUTHORIZATION_SERVICE_ADDRESS` | No | `authorization:50051` | Authorization gRPC target. |
| `SECRETS_SERVICE_ADDRESS` | No | `secrets:50051` | Secrets gRPC target. |
| `NOTIFICATIONS_ADDRESS` | No | `notifications:50051` | Notifications gRPC target. |
| `RECONCILIATION_INTERVAL` | No | `60s` | Reconciliation interval. |

## Helm validation

```sh
helm dependency update charts/egress
helm lint charts/egress
helm template egress charts/egress
```

The chart includes Istio `AuthorizationPolicy` rules for internal-only RPCs:
`ListEgressRulesByAgent` is limited to the Egress Gateway service account, and
`CountRulesReferencingSecret` is limited to the Secrets service account.
