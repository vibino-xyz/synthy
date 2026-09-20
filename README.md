<br />
<div align="center" id="readme-top">

  <h1 align="center">Synthy</h1>

  <p align="center">
    The indexing and retrieval engine of <a href="https://github.com/vibino-xyz">Vibino</a> — it turns a repository into a searchable graph of code, and answers retrieval queries over it.
    <br />
  </p>
</div>

<details>
  <summary>Table of Contents</summary>
  <ol>
    <li>
      <a href="#introduction">Introduction</a>
      <ul>
        <li><a href="#what-synthy-does">What Synthy does</a></li>
        <li><a href="#where-synthy-fits-in-vibino">Where Synthy fits in Vibino</a></li>
        <li><a href="#built-with">Built With</a></li>
      </ul>
    </li>
    <li><a href="#screenshots">Screenshots</a></li>
    <li>
      <a href="#architecture">Architecture</a>
      <ul>
        <li><a href="#the-analysis-pipeline">The analysis pipeline</a></li>
        <li><a href="#the-embedding-fan-out">The embedding fan-out</a></li>
        <li><a href="#the-retrieval-pipeline">The retrieval pipeline</a></li>
        <li><a href="#project-layout">Project layout</a></li>
      </ul>
    </li>
    <li>
      <a href="#getting-started">Getting Started</a>
      <ul>
        <li><a href="#prerequisites">Prerequisites</a></li>
        <li><a href="#installation">Installation</a></li>
        <li><a href="#environment-variables">Environment variables</a></li>
        <li><a href="#running-the-stack">Running the stack</a></li>
      </ul>
    </li>
    <li><a href="#services-ports">Services Ports</a></li>
    <li>
      <a href="#usage">Usage</a>
      <ul>
        <li><a href="#http-api">HTTP API</a></li>
        <li><a href="#grpc-api">gRPC API</a></li>
        <li><a href="#queues-and-exchanges">Queues and exchanges</a></li>
      </ul>
    </li>
    <li><a href="#data-model">Data Model</a></li>
    <li><a href="#testing">Testing</a></li>
    <li><a href="#roadmap">Roadmap</a></li>
    <li><a href="#contributing">Contributing</a></li>
  </ol>
</details>

## Introduction

Vibino is a company brain: connect your sources, and anyone on the team can ask questions and get answers grounded in what the company has actually written. **Synthy is the service that builds and reads that brain for code.**

### What Synthy does

Synthy owns the whole path from "a repository exists" to "here is the code that answers this question":

* **Connects GitHub.** Serves the dashboard's GitHub App flow — install link, installation callback, repository/branch listing, and disconnection — and records which installation belongs to which organization.
* **Clones and parses.** Shallow-clones a repository with a short-lived installation token and walks it with Go's `go/ast` and `golang.org/x/tools/go/packages`.
* **Builds a code graph.** Persists files, symbols (functions, types, interfaces, fields…), call edges, and import edges into PostgreSQL — not just flat text.
* **Chunks and embeds.** Splits code into semantic chunks (one per function / struct / interface), embeds them with [Voyage AI][Voyage-url] `voyage-code-3` and upserts the vectors into [Pinecone][Pinecone-url].
* **Summarises.** In parallel with embedding, an LLM writes a natural-language summary for each chunk and stores it alongside.
* **Indexes incrementally.** A push webhook only reprocesses the files that changed — every untouched chunk (and the embedding already paid for) stays in place.
* **Retrieves.** Exposes a gRPC `Retrieve` RPC that embeds a query, finds the nearest chunks, walks their call and import edges, and returns an LLM-ready context block with the chunks that back it.

Synthy does **no generation**. It hands assembled context back to its caller; the LLM step belongs to whoever asked.

### Where Synthy fits in Vibino

Vibino is split into small Go services that share a `commons` library and a `protos` contract module. Synthy is one of them.

| Service | Role |
| --- | --- |
| [**synthy**](https://github.com/vibino-xyz/synthy) | **This repo.** Code indexing, the code graph, embeddings, and retrieval. |
| [nexy](https://github.com/vibino-xyz/nexy) | Auth and authz — users, OTP signup, organizations, members, invitations, RBAC, and the JWTs every other service trusts. |
| [wolfy](https://github.com/vibino-xyz/wolfy) | Webhook gateway. Receives GitHub push events and publishes them as `RepositoryEventMessage` onto the queue Synthy consumes. |
| compass | Query engine. Calls Synthy's `Retrieve` over gRPC, then asks an LLM to answer from the returned context. Powers the chat experience. |
| [vibino-fe](https://github.com/vibino-xyz/vibino-fe) | Next.js frontend — landing page, auth, dashboard, and chat. Proxies `/api/nexy`, `/api/synthy` and `/api/compass` so the browser stays same-origin. |
| [commons](https://github.com/vibino-xyz/commons) | Shared `jwtauth`, `vhttp`, `vgrpc`, `ratelimit`, and `id` packages, plus the local `docker-compose` for Postgres and RabbitMQ. |
| [protos](https://github.com/vibino-xyz/protos) | Protobuf contracts — the `synthy.v1` gRPC API and the `contracts.v1` queue messages. |

> Everything else lives under **[github.com/vibino-xyz](https://github.com/vibino-xyz)**. Features owned by another service (sign-in, organizations, members, chat answers) are documented in that service's own repository.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Built With

* [![Go][Go]][Go-url]
* [![gRPC][gRPC]][gRPC-url]
* [![Protobuf][Protobuf]][Protobuf-url]
* [![PostgreSQL][Postgresql]][Postgresql-url]
* [![RabbitMQ][RabbitMQ]][RabbitMQ-url]
* [![Pinecone][Pinecone]][Pinecone-url]
* [![Docker][Docker]][Docker-url]

Plus [Uber fx][fx-url] for dependency injection, [Echo v5][echo-url] for HTTP, [golang-migrate][migrate-url] for embedded migrations, and [Voyage AI][Voyage-url] for code embeddings.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Screenshots

The dashboard below is served by [vibino-fe](https://github.com/vibino-xyz/vibino-fe); the GitHub connection, the repository list, and the indexing state on it are all Synthy's endpoints.

<p align="center">
    <img src="docs/images/dashboard.png" alt="Dashboard — GitHub connected, three repositories indexed by Synthy" width="49%">
    <img src="docs/images/chat.png" alt="Chat — questions answered from Synthy's indexed sources" width="49%">
</p>
<p align="center">
    <img src="docs/images/landing.png" alt="Vibino landing page" width="49%">
    <img src="docs/images/members.png" alt="Workspace members" width="49%">
</p>

And the part that belongs to this repo alone — `Retrieve`, answered from a live index of `vibino-xyz/nexy`:

```console
$ grpcurl -plaintext -import-path ./api -proto synthy/v1/synthy.proto \
    -d '{"query":"where is the JWT access token signed and verified","top_k":2}' \
    localhost:50051 synthy.v1.SynthyService/Retrieve

=== Retrieved Code Context ===

--- Chunk 1 | Type: TYPE | Language: GO | Lines: 12-15 ---
Content:
Service struct {
	signer     *jwtauth.Signer
	refreshTTL time.Duration
}

--- Chunk 2 | Type: TYPE | Language: GO | Lines: 11-15 ---
Content:
Signer struct {
	secret    []byte
	issuer    string
	accessTTL time.Duration
}
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Architecture

Synthy follows a ports-and-adapters layout: `core` holds the domain and its interfaces, `usecase` orchestrates them, `infra` implements them against Postgres / RabbitMQ / Pinecone / GitHub, and `interface` exposes them over HTTP, gRPC, and the message queue. Everything is wired in [`cmd/synthy/main.go`](cmd/synthy/main.go) with `fx`.

```mermaid
flowchart LR
    GH["GitHub push"] --> W["wolfy"]
    DASH["Dashboard: POST /index"] --> Q1
    W -->|RepositoryEventMessage| Q1[["repository_event_queue"]]
    Q1 --> AP["Analysis pipeline<br/>clone, parse, chunk"]
    AP --> PG[("PostgreSQL<br/>files, symbols, edges, chunks")]
    AP -->|EmbeddingEventMessage| EX{{"embedding_exchange<br/>fanout"}}
    EX --> Q2[["embedding_queue"]]
    EX --> Q3[["summary_queue"]]
    Q2 --> VO["Voyage embeddings"] --> PC[("Pinecone")]
    Q3 --> LLM["LLM summary"] --> PG
    CO["compass"] -->|gRPC Retrieve| RP["Retrieval pipeline"]
    RP --> PC
    RP --> PG
```

### The analysis pipeline

[`internal/usecase/analysis`](internal/usecase/analysis/analysisPipeline.go) is the write path.

1. Upsert the repository row, reusing the existing id — regenerating it would orphan every Pinecone vector, whose metadata records the repository and chunk ids at upsert time.
2. For a **full index**, replace everything the repository owns. For an **incremental index**, touch only the paths the push changed; the first index of a repository is always full, even if the event asks for incremental.
3. Per file: checksum it, record it, parse it, extract symbols, resolve call edges and import edges, and build one chunk per function / struct / interface.
4. Publish one `EmbeddingEventMessage` per chunk and return. Embedding and summarisation happen off the critical path.

### The embedding fan-out

`embedding_exchange` is a **fanout** exchange, so every chunk event reaches both consumers:

* [`EmbeddingEventController`](internal/interface/mq/embeddingEventController.go) embeds the chunk with Voyage, upserts the vector into Pinecone with the repository/file/symbol ids as metadata, and writes the vector id back onto the chunk.
* [`SummaryEventController`](internal/interface/mq/summaryEventController.go) asks the LLM for a summary and stores it on the chunk.

Both queues are declared with a dead-letter exchange, so a chunk that cannot be processed lands in `embedding_queue_error` / `summary_queue_error` instead of spinning forever.

### The retrieval pipeline

[`internal/usecase/retrieval`](internal/usecase/retrieval/retrievalPipeline.go) is the read path: embed the query (as an `InputTypeQuery`, not a document), query Pinecone for the top *k* vectors, load the matching chunks from Postgres, then enrich each one with its symbol's outgoing call edges and its file's import edges before formatting the context block.

### Project layout

```
cmd/synthy/                 fx wiring and process entrypoint
internal/
  core/                     domain models + repository/service interfaces
    repository/  file/  symbol/  calls/  imports/  chunker/
    embedding/  github/  llm/  organization/
  usecase/
    analysis/               indexing pipeline (full + incremental)
    retrieval/              query -> context assembly
    github/                 GitHub App connection orchestration
  infra/
    psql/                   pgx repositories + embedded migrations
    pinecone/               vector index
    rabbimq/                publishers, subscribers, exchange/DLX setup
    github/  git/           App client, token minting, shallow clone
    llm/voyage/             embeddings
    llm/ollama/             summaries
    llm/claude/             alternative summary client (opt-in)
  interface/
    api/                    Echo HTTP controllers
    rpc/                    gRPC server
    mq/                     queue consumers
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Getting Started

### Prerequisites

* [Go 1.25+](https://go.dev/dl/)
* [Docker](https://www.docker.com/) — for PostgreSQL and RabbitMQ
* `git` on `PATH` — the indexer shells out to it to clone repositories
* A [Pinecone](https://www.pinecone.io/) index and a [Voyage AI](https://www.voyageai.com/) API key
* An OpenAI-compatible LLM endpoint for chunk summaries ([Ollama](https://ollama.com/) works)
* A [GitHub App](https://docs.github.com/en/apps/creating-github-apps) with its private key, if you want the GitHub integration
* Optional: [air](https://github.com/air-verse/air) for live reload, [grpcurl](https://github.com/fullstorydev/grpcurl) to poke the gRPC API

### Installation

Synthy uses local `replace` directives for `commons` and `protos`, so clone the three repositories as siblings:

```sh
git clone git@github.com:vibino-xyz/commons.git
git clone git@github.com:vibino-xyz/protos.git
git clone git@github.com:vibino-xyz/synthy.git
```

Start PostgreSQL and RabbitMQ (the compose file lives in `commons`, and its init script creates the `synthy` and `nexy` databases):

```sh
docker compose -f ../commons/deployments/docker-compose.yaml up -d
```

Copy the environment template and fill it in:

```sh
cp .env.example .env
```

Migrations are embedded in the binary and run automatically on startup — there is no separate migrate step.

### Environment variables

| Variable | Description |
| --- | --- |
| `POSTGRES_CONNECTION_STRING` | e.g. `postgres://postgres:password@localhost:5432/synthy?sslmode=disable` |
| `RABBITMQ_URL` | e.g. `amqp://guest:guest@localhost:5672/` |
| `PORT` | HTTP port (default used by the stack: `8090`) |
| `PINECONE_API_KEY`, `PINECONE_HOST` | Vector index credentials |
| `VOYAGE_API_KEY` | Voyage AI key for code embeddings |
| `VOYAGE_MODEL`, `VOYAGE_BASE_URL`, `VOYAGE_OUTPUT_DIMENSION` | Default to `voyage-code-3`, the public API, and `1024` |
| `VOYAGE_RPM`, `VOYAGE_TPM` | Client-side rate limits (default `2000` / `3000000`) |
| `LLM_CLIENT_BASE_URL`, `LLM_CLIENT_MODEL` | OpenAI-compatible endpoint used for chunk summaries |
| `EMBEDDING_CLIENT_BASE_URL`, `EMBEDDING_CLIENT_MODEL` | Only needed when using the Ollama embedding client instead of Voyage |
| `CLAUDE_CODE_OAUTH_TOKEN`, `CLAUDE_BINARY_PATH`, `CLAUDE_MODEL` | Only needed for the opt-in Claude summary client |
| `JWT_SECRET`, `JWT_ISSUER` | Must match nexy — Synthy verifies the tokens nexy issues |
| `GITHUB_APP_ID`, `GITHUB_APP_SLUG`, `GITHUB_APP_PRIVATE_KEY_PATH` | GitHub App credentials |

### Running the stack

Synthy on its own:

```sh
go run ./cmd/synthy
```

…or with live reload:

```sh
air
```

On a healthy boot you should see the three consumers attach:

```console
[Fx] RUNNING
INFO Running database migrations...
INFO Migrations complete version=4 dirty=false
INFO Connected to database successfully
INFO Starting repository event subscriber queue=repository_event_queue
INFO Starting embedding event subscriber queue=embedding_queue
INFO Starting summary event subscriber queue=summary_queue
```

To bring up the full product (the screenshots above), run each service from its own directory — `nexy` on `:8080`, `synthy` on `:8090`, `compass` on `:8100`, `wolfy` on `:8000` — and then the frontend:

```sh
npm run dev
```

The frontend proxies `/api/nexy/*`, `/api/synthy/*`, and `/api/compass/*` to those origins, so open http://localhost:3000 and nothing else needs CORS.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Services Ports

- Synthy:
    * `HTTP API` : 8090 (`PORT`)
    * `gRPC API` : 50051
<br/><br/>
- Other Vibino services:
    * `nexy` : 8080
    * `compass` : 8100
    * `wolfy` : 8000
    * `vibino-fe` : 3000
<br/><br/>
- Infrastructure:
    * `PostgreSQL` : 5432
    * `RabbitMQ` : 5672
    * `RabbitMQ management UI` : 15672

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Usage

### HTTP API

Every route is under `/synthy/v1/github` and requires a nexy-issued bearer token; the organization is taken from the token's `oid` claim, never from the request body. Routes marked *manager* additionally require an `OWNER` or `ADMIN` role.

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/connection` | Whether the App is configured and connected, and to which account |
| `GET` | `/install-url` | *manager* — GitHub App install link, carrying the org id as `state` |
| `POST` | `/connect` | *manager* — records an installation id after the GitHub callback |
| `DELETE` | `/connection` | *manager* — disconnects the installation |
| `GET` | `/repositories` | Repositories the installation can see |
| `GET` | `/repositories/indexed` | Repositories Synthy has actually indexed |
| `GET` | `/branches?repo=owner/name` | Branches of one repository |
| `POST` | `/index` | *manager* — queues a full index of every connected repository |

```sh
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8090/synthy/v1/github/repositories/indexed
```

```json
{
  "repositories": [
    {
      "id": "1b546b89aabe8cb5876be63a1ca02d73",
      "external_id": "1171109879",
      "full_name": "vibino-xyz/synthy",
      "default_branch": "main",
      "provider": "GITHUB",
      "repository_url": "https://github.com/vibino-xyz/synthy.git",
      "state": "indexed",
      "indexed_at": "2026-08-09T22:20:24.923394Z"
    }
  ]
}
```

### gRPC API

`synthy.v1.SynthyService/Retrieve` on `:50051` — defined in [protos/api/synthy/v1/synthy.proto](https://github.com/vibino-xyz/protos/blob/main/api/synthy/v1/synthy.proto).

```protobuf
rpc Retrieve(RetrieveRequest) returns (RetrieveResponse);
```

`RetrieveRequest` takes a `query`, an `organization_id`, and an optional `top_k` (defaults to 7). The response carries `formatted_context` — the block a caller feeds to an LLM — and the `chunks` behind it, so the caller can cite them.

### Queues and exchanges

| Name | Kind | Purpose |
| --- | --- | --- |
| `repository_event_exchange` | fanout exchange | Index jobs. Synthy's publisher mirrors wolfy's topology exactly, so a webhook and a `POST /index` land on the same queue |
| `repository_event_queue` | queue | What the indexer consumes |
| `repository_event_dlx_exchange` | exchange | Dead letters from the index queue |
| `embedding_exchange` | fanout exchange | One chunk event, two consumers |
| `embedding_queue` | queue | Voyage embedding → Pinecone upsert |
| `summary_queue` | queue | LLM chunk summary → Postgres |
| `embedding_dlx_exchange` | exchange | Dead letters from both chunk queues |
| `embedding_queue_error`, `summary_queue_error` | queues | Chunk messages that could not be processed |

```console
$ docker exec vibino_rabbitmq rabbitmqctl list_queues name messages consumers
name                    messages  consumers
repository_event_queue  0         1
embedding_queue         0         1
summary_queue           0         1
embedding_queue_error   0         0
summary_queue_error     0         0
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Data Model

Migrations live in [`internal/infra/psql/migrations`](internal/infra/psql/migrations) and are embedded into the binary.

| Table | What it holds |
| --- | --- |
| `repository` | One row per indexed repository, keyed to an organization and a stable `external_id` |
| `file` | Path, checksum, language, and binary flag — the checksum is what makes incremental indexing possible |
| `symbol` | Functions, types, interfaces, fields, and more, with signature, visibility, receiver kind, and position |
| `call_edge` | Caller → callee, per repository |
| `import_edge` | File → file, with the import path, alias, and whether it is external |
| `code_chunk` | The embedded unit: content, content hash, type, line range, Pinecone `embedding_id`, and LLM `summary` |
| `github_installation` | Maps a GitHub App installation to a Vibino organization |

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Testing

```sh
go test ./...
```

The unit tests around chunking, the ingestion mapper, and file handling run anywhere. The tests that exercise the pipeline, the database, and the publishers need `.env` plus a running Postgres and RabbitMQ; they skip themselves when those are missing.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Roadmap

- [x] Code graph: files, symbols, call edges, import edges
- [x] Chunking, embedding, and LLM summaries over a fanout exchange with dead-lettering
- [x] GitHub App connection and repository listing
- [x] Incremental indexing driven by push webhooks
- [x] gRPC retrieval with call/import edge enrichment
- [ ] Per-organization Pinecone namespaces — `Retrieve` accepts `organization_id` but still queries a shared namespace
- [ ] Make the gRPC port configurable instead of a hardcoded `:50051`
- [ ] Languages beyond Go — the parser, symbol extractor, and chunker are Go-only today
- [ ] Per-repository indexing state, so the dashboard can show progress rather than inferring "indexed" from a row's existence
- [ ] Hybrid retrieval: combine vector search with graph traversal and lexical search
- [ ] GitLab and Bitbucket providers (already modelled in the schema)

<p align="right">(<a href="#readme-top">back to top</a>)</p>

## Contributing

If you have a suggestion that would make this better, please fork the repo and create a pull request. You can also simply open an issue with the `enhancement` tag.

1. Fork the Project
2. Create your Feature Branch <br/>
`git checkout -b feature/new_feature`
3. Commit your Changes <br/>
`git commit -m 'Add new_feature'`
4. Push to the Branch <br/>
`git push origin feature/new_feature`
5. Open a Pull Request

<p align="right">(<a href="#readme-top">back to top</a>)</p>

[Go]: https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white
[Go-url]: https://go.dev/
[gRPC]: https://img.shields.io/badge/gRPC-244c5a?style=for-the-badge&logo=grpc&logoColor=white
[gRPC-url]: https://grpc.io/
[Protobuf]: https://img.shields.io/badge/Protobuf-0f9d58?style=for-the-badge&logo=protobuf&logoColor=white
[Protobuf-url]: https://protobuf.dev/
[Postgresql]: https://img.shields.io/badge/postgresql-white?style=for-the-badge&logo=postgresql&logoColor=blue
[Postgresql-url]: https://www.postgresql.org/
[RabbitMQ]: https://img.shields.io/badge/RabbitMQ-FF6600?style=for-the-badge&logo=rabbitmq&logoColor=white
[RabbitMQ-url]: https://www.rabbitmq.com/
[Pinecone]: https://img.shields.io/badge/Pinecone-000000?style=for-the-badge&logo=pinecone&logoColor=white
[Pinecone-url]: https://www.pinecone.io/
[Docker]: https://img.shields.io/badge/docker-blue?style=for-the-badge&logo=docker&logoColor=white
[Docker-url]: https://www.docker.com/
[Voyage-url]: https://www.voyageai.com/
[fx-url]: https://uber-go.github.io/fx/
[echo-url]: https://echo.labstack.com/
[migrate-url]: https://github.com/golang-migrate/migrate
