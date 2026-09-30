# Zarv Go

[![CI](https://github.com/zarvhq/zarv-go/actions/workflows/ci.yml/badge.svg)](https://github.com/zarvhq/zarv-go/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/zarvhq/zarv-go)](https://goreportcard.com/report/github.com/zarvhq/zarv-go)
[![GoDoc](https://pkg.go.dev/badge/github.com/zarvhq/zarv-go)](https://pkg.go.dev/github.com/zarvhq/zarv-go)

Biblioteca de módulos compartilhados em Go para projetos Zarv.

## 📦 Instalação

```bash
go get github.com/zarvhq/zarv-go
```

## 📚 Documentação

### ☁️ [Google Cloud Platform (GCP)](docs/gcp.md)

Clientes para serviços do Google Cloud Platform.

**Pacotes disponíveis:**

#### GCS (Google Cloud Storage)
```go
import "github.com/zarvhq/zarv-go/pkg/gcp/gcs"
```
Upload/download de objetos e geração de signed URLs.

#### Document AI
```go
import "github.com/zarvhq/zarv-go/pkg/gcp/documentai"
```
Processamento de documentos com IA.

#### Pub/Sub
```go
import "github.com/zarvhq/zarv-go/pkg/gcp/pubsub"
```
Publicação e consumo de mensagens em tópicos e subscriptions.

**Funcionalidades:**
- Suporte a Workload Identity
- Graceful shutdown
- Processamento concorrente
- Serialização automática JSON
- Panic recovery

**[📖 Ver documentação completa →](docs/gcp.md)**

---

### 🐰 [RabbitMQ](docs/rabbitmq.md)

Cliente para operações com RabbitMQ.

```go
import "github.com/zarvhq/zarv-go/pkg/rabbitmq"
```

**Funcionalidades:**
- Producer com reconnection automática
- Consumer com graceful shutdown
- Handlers personalizáveis
- Processamento concorrente configurável
- Serialização automática JSON
- Thread-safe

**[📖 Ver documentação completa →](docs/rabbitmq.md)**

---

### 📥 [Ingestion](pkg/ingestion/README.md)

Envio de eventos ao gateway de ingestão, que grava cada um na sua tabela `bronze`.

```go
import "github.com/zarvhq/zarv-go/pkg/ingestion"
```

**Funcionalidades:**
- Configuração por `ZARV_INGESTION_URL` e `ZARV_INGESTION_KEY`, validada na partida
- Novas tentativas seguras, respeitando `Retry-After`
- Recusas do gateway como erros tipados, inclusive dentro de um 202
- Uma linha de log por falha, sem registro nem chave

**[📖 Ver documentação completa →](pkg/ingestion/README.md)**

## 🤝 Contribuindo

1. Faça um fork do projeto
2. Crie uma branch para sua feature (`git checkout -b feature/nova-feature`)
3. Commit suas mudanças (`git commit -am 'Adiciona nova feature'`)
4. Push para a branch (`git push origin feature/nova-feature`)
5. Abra um Pull Request

## 📝 Versionamento

Este projeto segue o [Semantic Versioning](https://semver.org/). Para as versões disponíveis, veja as [tags neste repositório](https://github.com/zarvhq/zarv-go/tags).

### Publicando uma nova versão

Para criar uma nova release:

```bash
# Criar e enviar uma tag seguindo semantic versioning
git tag v1.0.0
git push origin v1.0.0
```

O GitHub Actions automaticamente:
- Valida o formato da tag
- Executa os testes
- Cria a release no GitHub
- Publica o módulo

## 📄 Licença

Este projeto está sob a licença MIT. Veja o arquivo [LICENSE](LICENSE) para mais detalhes.

## 🔗 Links

- [Documentação Go](https://pkg.go.dev/github.com/zarvhq/zarv-go)
- [Issues](https://github.com/zarvhq/zarv-go/issues)
- [Pull Requests](https://github.com/zarvhq/zarv-go/pulls)
