# goexptAuction

Sistema de leilões em Go com fechamento automático via Goroutines — desafio da pós Go Expert.

A base do projeto é o repositório do curso ([devfullcycle/labs-auction-goexpert](https://github.com/devfullcycle/labs-auction-goexpert)), que já implementa criação de leilões, lances (bids) e a validação que impede lances em leilões encerrados. O acréscimo deste repositório é a rotina de **fechamento automático** do leilão após a duração configurada.

## Stack

- Go 1.20 (API REST com [Gin](https://github.com/gin-gonic/gin))
- MongoDB
- Docker / Docker Compose

## Como rodar

O `docker-compose.yml` lê as variáveis de `cmd/auction/.env`, que **não é versionado**. Crie-o a partir do template antes de subir a stack:

```bash
cp cmd/auction/.env.example cmd/auction/.env
docker compose up --build
```

A API sobe em `http://localhost:8080` e o MongoDB em `localhost:27017`. O container da aplicação só inicia depois que o MongoDB responde ao healthcheck.

Para derrubar tudo (o `-v` também remove o volume de dados do Mongo):

```bash
docker compose down -v
```

## Variáveis de ambiente

Todas ficam em `cmd/auction/.env` — veja `cmd/auction/.env.example`.

| Variável | Descrição | Exemplo | Fallback no código |
| --- | --- | --- | --- |
| `AUCTION_INTERVAL` | **Duração do leilão.** Define quanto tempo após a criação o leilão é fechado automaticamente. | `20s` | `5m` |
| `BATCH_INSERT_INTERVAL` | Intervalo de flush do lote de lances. | `20s` | `3m` |
| `MAX_BATCH_SIZE` | Quantidade de lances que dispara a gravação do lote. | `4` | `5` |
| `MONGODB_URL` | String de conexão do MongoDB. | `mongodb://admin:admin@mongodb:27017/auctions?authSource=admin` | — |
| `MONGODB_DB` | Nome do banco. | `auctions` | — |
| `MONGO_INITDB_ROOT_USERNAME` | Usuário root criado pelo container do Mongo. | `admin` | — |
| `MONGO_INITDB_ROOT_PASSWORD` | Senha do usuário root. | `admin` | — |

As durações usam o formato do [`time.ParseDuration`](https://pkg.go.dev/time#ParseDuration): `30s`, `5m`, `1h30m`. Valores inválidos não derrubam a aplicação — ela cai silenciosamente no fallback da tabela acima.

Para um leilão de 1 minuto, por exemplo:

```
AUCTION_INTERVAL=1m
```

## Rodando fora do Docker

Requer um MongoDB acessível. Ajuste o host em `MONGODB_URL` de `mongodb` (nome do serviço no Compose) para `localhost`:

```
MONGODB_URL=mongodb://admin:admin@localhost:27017/auctions?authSource=admin
```

Execute **a partir da raiz do repositório** — o `main.go` carrega `cmd/auction/.env` por caminho relativo:

```bash
docker compose up -d mongodb   # apenas o banco
go run cmd/auction/main.go
```

## Endpoints

| Método | Rota | Descrição |
| --- | --- | --- |
| `POST` | `/auction` | Cria um leilão |
| `GET` | `/auction?status=&category=&productName=` | Lista leilões por filtro |
| `GET` | `/auction/:auctionId` | Busca leilão por id |
| `GET` | `/auction/winner/:auctionId` | Retorna o lance vencedor do leilão |
| `POST` | `/bid` | Registra um lance |
| `GET` | `/bid/:auctionId` | Lista os lances de um leilão |
| `GET` | `/user/:userId` | Busca usuário por id |

Criando um leilão:

```bash
curl -i -X POST http://localhost:8080/auction \
  -H 'Content-Type: application/json' \
  -d '{
    "product_name": "Notebook",
    "category": "Eletronicos",
    "description": "Notebook usado em bom estado de conservacao",
    "condition": 1
  }'
```

`condition`: `1` = New, `2` = Used, `3` = Refurbished.
`status` (na resposta): `0` = Active, `1` = Completed.

Consultando os leilões abertos:

```bash
curl -s "http://localhost:8080/auction?status=0"
```

## Testes

```bash
go test ./...
go test -race ./...                  # recomendado: o projeto é concorrente
go test -v -run TestNome ./caminho/do/pacote/
```

## Fechamento automático

> **TODO** — a implementar. Esta seção deve documentar a goroutine de fechamento
> em `internal/infra/database/auction/create_auction.go` e o teste automatizado
> que comprova a transição do status para `Completed` após `AUCTION_INTERVAL`.
