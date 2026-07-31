# goexptAuction

Desafio de fechamento automático de leilões da pós Go Expert.

A base é o [repositório do curso](https://github.com/devfullcycle/labs-auction-goexpert), que já implementa criação de leilões, lances e a validação que recusa lance em leilão encerrado.O leilão nunca expira. 

O acréscimo aqui é a Goroutine que fecha o leilão sozinho depois da duração configurada, assim como a revalidação de leilões ativos na inicialização da aplicação, para não perder nenhum agendamento caso o serviço seja reiniciado.

## Como rodar

O compose lê `cmd/auction/.env`. 

Crie a partir do template:

```bash
cp cmd/auction/.env.example cmd/auction/.env
docker compose up --build
```

A API sobe em `http://localhost:8080` e o Mongo em `localhost:27017`. A app só inicia depois que o Mongo responde ao healthcheck.

Para derrubar tudo:

```bash
docker compose down -v
```

Sem Docker, `MONGODB_URL` para `localhost` e rode **a partir da raiz**, porque o `main.go` usa `cmd/auction/.env` caminho relativo:

```bash
docker compose up -d mongodb
MONGODB_URL=mongodb://admin:admin@localhost:27017/auctions?authSource=admin go run cmd/auction/main.go
```

## Variáveis de ambiente

Ficam em `cmd/auction/.env` leve como base -> `cmd/auction/.env.example`.

| Variável | Descrição | Exemplo | Fallback |
| --- | --- | --- | --- |
| `AUCTION_INTERVAL` | **Duração do leilão**, do momento da criação até o fechamento automático | `20s` | `5m` |
| `BATCH_INSERT_INTERVAL` | Intervalo de flush do lote de lances | `20s` | `3m` |
| `MAX_BATCH_SIZE` | Quantidade de lances que dispara a gravação do lote | `4` | `5` |
| `MONGODB_URL` | Conexão do Mongo | `mongodb://admin:admin@mongodb:27017/auctions?authSource=admin` | — |
| `MONGODB_DB` | Nome do banco | `auctions` | — |
| `MONGO_INITDB_ROOT_USERNAME` / `_PASSWORD` | Credenciais criadas pelo container do Mongo | `admin` | — |

As durações usam o formato do [`time.ParseDuration`](https://pkg.go.dev/time#ParseDuration): `30s`, `1m`, `1h30m` com valor default igual tabela. 

Para um leilão de um minuto:

```
AUCTION_INTERVAL=1m
```

## Rotas

| Método | Rota | O que faz |
| --- | --- | --- |
| `POST` | `/auction` | Cria um leilão |
| `GET` | `/auction?status=&category=&productName=` | Lista leilões por filtro |
| `GET` | `/auction/:auctionId` | Busca leilão por id |
| `GET` | `/auction/winner/:auctionId` | Lance vencedor do leilão |
| `POST` | `/bid` | Registra um lance |
| `GET` | `/bid/:auctionId` | Lances de um leilão |
| `GET` | `/user/:userId` | Busca usuário por id |

```bash
curl -s -X POST http://localhost:8080/auction \
  -H 'Content-Type: application/json' \
  -d '{"product_name":"HB20","category":"Automovel",
       "description":"Completo 4 portas, bom estado de conservacao","condition":1}'
```

| Campo | Valor | Significado |
| --- | --- | --- |
| `condition` | `1` | New |
| `condition` | `2` | Used |
| `condition` | `3` | Refurbished |


| Campo | Valor | Significado |
| --- | --- | --- |
| `status` | `0` | Active |
| `status` | `1` | Completed |

## Fechamento automático

Ao criar um leilão, uma Goroutine é disparada e aguarda `AUCTION_INTERVAL` e então grava `Completed` no Mongo quando o intervalo termina.

O prazo é `timestamp_de_criação + AUCTION_INTERVAL`, a mesma fórmula que o `BidRepository` já usa para recusar lances vencidos. Como a validação vive só em memória, um restart perderia os agendamentos, então adicionado na inicialização, `ScheduleActiveAuctions` varre os leilões ainda abertos, fecha os vencidos e reagenda novamente o resto.

| Arquivo | Papel |
| --- | --- |
| `internal/infra/database/auction/create_auction.go` | Agenda a goroutine e lê `AUCTION_INTERVAL` |
| `internal/infra/database/auction/update_auction.go` | `CloseAuction` e a varredura de inicialização |

Verificando pela API, com `AUCTION_INTERVAL=20s`:

```bash
curl -s "http://localhost:8080/auction?status=0"   # logo após criar: aparece aqui
sleep 25
curl -s "http://localhost:8080/auction?status=1"   # passado o intervalo: fechado
```
Log do fechamento automático:

```json
{"level":"info","time":"2026-07-31T16:52:56.693Z","message":"Auction 2c6e1c4c-a891-42e4-9f18-a9b2f242f8f7 closed automatically"}
```

## Testes


```bash
go test -race ./...
```

Os de integração comprovam o cenário do desafio .

Criar, aguardar o intervalo, conferir `Completed`. 

`Precisam de um Mongo acessível!`

```bash
docker compose up -d mongodb
go test -race -v ./internal/infra/database/auction/
```


## Stack

Go 1.20 com [Gin](https://github.com/gin-gonic/gin), MongoDB, Docker Compose e [testify](https://github.com/stretchr/testify) nos testes.
