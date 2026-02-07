# Zarv Go

[![CI](https://github.com/zarvhq/zarv-go/actions/workflows/ci.yml/badge.svg)](https://github.com/zarvhq/zarv-go/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/zarvhq/zarv-go)](https://goreportcard.com/report/github.com/zarvhq/zarv-go)
[![GoDoc](https://pkg.go.dev/badge/github.com/zarvhq/zarv-go)](https://pkg.go.dev/github.com/zarvhq/zarv-go)

Biblioteca de módulos compartilhados em Go para projetos Zarv.

## 📦 Instalação

```bash
go get github.com/zarvhq/zarv-go
```

## 📚 Pacotes Disponíveis

### Fiber Middleware

Middlewares para aplicações Fiber com autenticação e autorização Zarv. Veja [documentação completa](pkg/fiber/README.md).

#### Instalação

```go
import "github.com/zarvhq/zarv-go/pkg/fiber/v2/middleware"
```

#### Uso

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/zarvhq/zarv-go/pkg/fiber/v2/middleware"
)

func main() {
    app := fiber.New()

    // Adicionar middleware de autenticação
    app.Use(middleware.Authenticate)

    app.Get("/api/resource", func(c *fiber.Ctx) error {
        // Obter perfil de autenticação
        profile := middleware.GetAuthProfile(c)
        
        // Verificar permissões
        if profile.IsViewer() {
            return c.Status(403).JSON(fiber.Map{
                "error": "Permission denied",
            })
        }

        return c.JSON(fiber.Map{
            "workspaceId": profile.WorkspaceID,
            "userId": profile.UserID,
        })
    })

    app.Listen(":3000")
}
```

### GCP (Google Cloud Platform)

Clientes para serviços do Google Cloud Platform. Veja [documentação completa](pkg/gcp/README.md).

#### GCS (Google Cloud Storage)

```go
import "github.com/zarvhq/zarv-go/pkg/gcp/gcs"
```

#### Document AI

```go
import "github.com/zarvhq/zarv-go/pkg/gcp/documentai"
```

#### Pub/Sub

Cliente para operações com Google Cloud Pub/Sub (tópicos e filas). Veja [documentação completa](pkg/gcp/pubsub/README.md).

```go
import "github.com/zarvhq/zarv-go/pkg/gcp/pubsub"
```

**Exemplo Rápido:**

```go
// Publisher
client, _ := pubsub.NewClient(ctx, &pubsub.Cfg{ProjectID: "my-project"})
publisher, _ := client.NewPublisher("topic-name")
messageID, _ := publisher.Publish(ctx, map[string]string{"msg": "hello"})

// Subscriber
type Handler struct{}
func (h *Handler) HandleMessage(data []byte, attributes map[string]string) error {
    return nil
}

subscriber, _ := client.NewSubscriber("subscription-id", &Handler{})
subscriber.Receive(10) // 10 workers concorrentes
```

### RabbitMQ

Cliente para operações com RabbitMQ. Veja [documentação completa](pkg/rabbitmq/README.md).

#### Instalação

```go
import "github.com/zarvhq/zarv-go/pkg/rabbitmq"
```

#### Exemplo Rápido

```go
// Producer
client, _ := rabbitmq.NewClient(ctx, "amqp://localhost:5672/")
producer, _ := client.NewProducer()
producer.Publish("queue-name", map[string]string{"msg": "hello"})

// Consumer
type Handler struct{}
func (h *Handler) HandleMessage(data []byte) error { return nil }

consumer, _ := client.NewConsumer("consumer-name", "queue-name", &Handler{})
consumer.Consume(5) // 5 workers concorrentes
```
            })
        }

        return c.JSON(fiber.Map{
            "workspaceId": profile.WorkspaceID,
            "userId": profile.UserID,
        })
    })

    app.Listen(":3000")
}
```

#### Headers Esperados

O middleware `Authenticate` valida os seguintes headers:

- `X-Issuer`: Identificador do emissor (ex: "ultron-app", "vision-app")
- `X-Workspace-Id`: ID do workspace Zarv
- `X-User-Id`: ID do usuário Zarv
- `X-Zarv-Role`: Role do usuário
- `X-Access-Level`: Nível de acesso (viewer, user, supervisor, admin)
- `X-Internal`: (Opcional) Indica requisição interna

#### Níveis de Acesso

- `viewer`: Acesso somente leitura
- `user`: Acesso de usuário padrão
- `supervisor`: Acesso de supervisor
- `admin`: Acesso administrativo completo

#### Métodos do AuthProfile

```go
profile := middleware.GetAuthProfile(c)

// Verificar se é administrador Zarv (role zarver + admin/supervisor)
profile.IsZarvAdmin() // bool

// Verificar se é administrador do workspace
profile.IsUserAdmin() // bool

// Verificar se tem apenas acesso de visualização
profile.IsViewer() // bool
```

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
