# fctech-rate-limiter

Rate limiter middleware in Go with Redis storage. Supports limiting by IP or API token. Token-specific limits override IP limits.

Descrição
---------

Implementação simples de um rate limiter em Go que permite limitar requisições por endereço IP ou por token de acesso (header `API_KEY`). As configurações podem ser feitas via variáveis de ambiente ou arquivo `.env` na raiz do projeto. As informações de contagem e bloqueio são persistidas no Redis.

Onde colocar o `.env`
---------------------

Coloque o arquivo `.env` na raiz do repositório (mesmo diretório do `go.mod` e do `docker-compose.yml`). O servidor carrega `./.env` automaticamente (loader minimalista). Há um arquivo de exemplo em `.env.example`.

Pré-requisitos
--------------

- Go (compatível com `go.mod`)
- Docker + Docker Compose (opcional, para subir Redis)
- curl.exe ou PowerShell (Invoke-RestMethod) para testar

Como rodar localmente
---------------------

1) Subir o Redis (opcional, recomendado):

```powershell
cd d:\Workspace\FCTECH\fctech-rate-limiter
docker-compose up -d
```

2) Ajuste o `.env` (ou copie do exemplo):

```powershell
Copy-Item .env.example .env
# edite .env para ajustar limites e tokens
```

3) Rodar o servidor (desenvolvimento):

```powershell
go run ./cmd/server
```

ou gerar binário:

```powershell
go build -o bin/server ./cmd/server
.\bin\server
```

Testes manuais (PowerShell)
---------------------------

Observação: no PowerShell o comando `curl` é um alias para `Invoke-WebRequest`; para usar a sintaxe tradicional `-H "Name: Value"` chame `curl.exe` explicitamente (se disponível). Alternativamente, use `Invoke-RestMethod` com um hashtable de headers.

Exemplo com `curl.exe` (se disponível):

```powershell
for ($i = 1; $i -le 6; $i++) {
	curl.exe -H "X-Forwarded-For: 1.2.3.4" http://localhost:8080/ping
	Write-Host ""
}

# com token
curl.exe -H "API_KEY: abc123" http://localhost:8080/ping
```

Exemplo com PowerShell nativo (`Invoke-RestMethod`):

```powershell
$headers = @{ 'X-Forwarded-For' = '1.2.3.4' }
for ($i = 1; $i -le 6; $i++) {
	Invoke-RestMethod -Uri 'http://localhost:8080/ping' -Headers $headers -Method Get
	Write-Host ""
}

# com API key
$h = @{ 'API_KEY' = 'abc123' }
Invoke-RestMethod -Uri 'http://localhost:8080/ping' -Headers $h -Method Get
```

Inspecionando o Redis
---------------------

Se o Redis estiver rodando via docker-compose com o serviço `redis`, você pode conectar ao CLI:

```powershell
docker exec -it fctech_redis redis-cli
# então dentro do redis-cli:
KEYS *
TTL ip:1.2.3.4
TTL blocked:ip:1.2.3.4
```

As chaves usadas são `ip:<ip>` e `token:<token>`; chaves de bloqueio são `blocked:<key>`.

Executando os testes automatizados
---------------------------------

Rode todos os testes do projeto:

```powershell
go test ./... -v
```

Isso executa os testes unitários para `internal/limiter` e `pkg/middleware` (já incluídos no repositório).

Formato de `TOKEN_LIMITS`
-----------------------

Variável `TOKEN_LIMITS` no `.env` tem o formato:

```
TOKEN_LIMITS=TOKEN:LIMIT:WINDOW_SECONDS:BLOCK_SECONDS,TOKEN2:...
```

Exemplo:

```
TOKEN_LIMITS=abc123:100:1:300,def456:50:1:60
```

Observações e recomendações
---------------------------

- O loader de `.env` é minimalista (só `KEY=VALUE`). Se precisar de suporte a aspas, `export` ou valores complexos, recomendo integrar `github.com/joho/godotenv`.
- A extração de IP usa `X-Forwarded-For` e `RemoteAddr` como fallback; para produção com proxies, melhore a lógica ou configure trusted proxies.
- A estratégia de persistência é baseada em uma interface `internal/storage.Storage` — é fácil trocar o Redis por outra implementação.
- Os testes atuais cobrem a lógica do limiter e o middleware; é recomendável adicionar testes de concorrência e uma implementação `storage` em memória reutilizável.

Contribuições e próximos passos
------------------------------

Se quiser, posso:

- trocar o loader por `godotenv` para suportar casos mais complexos;
- adicionar uma `storage` em memória para executar o servidor sem Redis (útil para testes);
- adicionar scripts PowerShell em `scripts/` para automatizar os testes manuais;
- adicionar métricas/logs (Prometheus / zerolog).

Obrigado — se quiser, eu faço qualquer uma das melhorias acima.

# fctech-rate-limiter
Rate limiter em Go que possa ser utilizado para controlar o tráfego de requisições para um serviço web. 
