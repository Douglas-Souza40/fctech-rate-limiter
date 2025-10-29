# fctech-rate-limiter

Rate limiter middleware in Go with Redis storage. Supports limiting by IP or API token. Token-specific limits override IP limits.

Como usar (exemplo):

1. Copie `.env.example` para `.env` e ajuste valores.
2. Suba o Redis com docker-compose:

```powershell
docker-compose up -d
```

3. Execute o servidor:

```powershell
go build ./...; .\cmd\server\server.exe
```

4. Teste:

```powershell
# sem token (limite por IP)
for ($i=0; $i -lt 10; $i++) { curl http://localhost:8080/ping }

# com token
curl -H "API_KEY: abc123" http://localhost:8080/ping
```

Arquitetura e decisões:
- `internal/storage` contém a interface `Storage` e implementação Redis.
- `internal/limiter` contém a lógica de limitação (separada do middleware).
- `pkg/middleware` contém middleware HTTP para injetar o limiter no handler.

Próximos passos (sugestões):
- adicionar testes unitários e mocks para `storage.Storage`.
- implementar outra estratégia (memória, banco SQL) seguindo a `Storage` interface.
# fctech-rate-limiter
Rate limiter em Go que possa ser utilizado para controlar o tráfego de requisições para um serviço web. 
