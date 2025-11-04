# Rate Limiter

Rate limiter HTTP escrito em Go 1.25 seguindo arquitetura hexagonal. A lógica de rate limit reside no domínio, é exposta via uma porta (`internal/core/ports/limiter.go`) e pode trabalhar com diferentes storages através da porta `Storage`. A implementação inicial utiliza Redis, mas outras estratégias podem ser adicionadas registrando novos adaptadores.

## Estrutura

- `cmd/server`: ponto de entrada, faz o bootstrap da aplicação.
- `internal/config`: carrega configurações via variáveis de ambiente (`godotenv`).
- `internal/core/domain`: entidades e erros do domínio.
- `internal/core/services`: lógica do rate limiter desacoplada de HTTP ou Redis.
- `internal/adapters/http`: middleware/handlers usando Chi.
- `internal/adapters/storage`: adaptadores concretos de persistência (Redis).

## Configuração

Variáveis principais (ver `.env.example`):

```
SERVER_PORT=8080
STORAGE_TYPE=redis
REDIS_HOST=redis
REDIS_PORT=6379

# Regras por IP
RATE_LIMIT_IP_REQUESTS=10
RATE_LIMIT_IP_WINDOW_SECONDS=1
RATE_LIMIT_IP_BLOCK_DURATION_MINUTES=5

# Regras por token
RATE_LIMIT_TOKEN_DEFAULT_REQUESTS=100
RATE_LIMIT_TOKEN_DEFAULT_WINDOW_SECONDS=1
RATE_LIMIT_TOKEN_DEFAULT_BLOCK_DURATION_MINUTES=5

# Overrides específicos por token (TOKEN:REQUESTS:WINDOW_SECONDS:BLOCK_MINUTES)
TOKENS=abc123:100:1:5,xyz789:50:1:10
```

O header esperado para autenticação por token é `API_KEY: <TOKEN>`. Regras de tokens têm prioridade sobre as de IP.

## Executando com Docker

```bash
cp .env.example .env
docker compose up --build
```

O serviço fica disponível em `http://localhost:8080/test`. Envie requisições com e sem o header `API_KEY` para validar a aplicação das regras.

## Execução local

```bash
go mod tidy        # uma vez, para baixar dependências
go run ./cmd/server
```

Certifique-se de ter um Redis acessível conforme configurado no `.env`.

## Testes

```bash
go test ./...
```

Os testes de unidade cobrem os principais fluxos do `RateLimiterService` e validam a separação entre domínio e adaptadores.
