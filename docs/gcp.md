# Google Cloud Platform (GCP)

Clientes para serviços do Google Cloud Platform.

## 📦 Pacotes Disponíveis

- [Google Cloud Storage (GCS)](#google-cloud-storage-gcs)
- [Document AI](#document-ai)
- [Pub/Sub](#pubsub)
- [Metrics](#metrics)

---

## Google Cloud Storage (GCS)

Cliente para operações com Google Cloud Storage.

### 📦 Instalação

```bash
go get github.com/zarvhq/zarv-go/pkg/gcp/gcs
```

```go
import "github.com/zarvhq/zarv-go/pkg/gcp/gcs"
```

### 🚀 Uso

```go
package main

import (
    "context"
    "github.com/zarvhq/zarv-go/pkg/gcp/gcs"
)

func main() {
    ctx := context.Background()
    
    // Criar cliente (usa Workload Identity por padrão)
    client, err := gcs.NewClient(ctx, &gcs.Cfg{
        BucketName: "my-bucket",
        // CredentialsJSON: []byte(`{...}`), // Opcional
    })
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // Upload de objeto
    err = client.PutObject(&gcs.Object{
        Name: "path/to/file.txt",
        Data: []byte("Hello, GCS!"),
    })
    
    // Download de objeto
    obj, err := client.GetObject(&gcs.Object{
        Name: "path/to/file.txt",
    })
    
    // Signed URL para upload
    signedURL, err := client.PutObjectSignedURL(&gcs.SignedURL{
        ObjectName: "uploads/file.pdf",
        ExpiresIn:  3600, // 1 hora
    })
}
```

### 📋 Interface

```go
type Client interface {
    GetObject(*Object) (*Object, error)
    PutObject(*Object) error
    GetObjectSignedURL(*SignedURL) (*SignedURL, error)
    PutObjectSignedURL(*SignedURL) (*SignedURL, error)
    Close() error
}
```

### 🔗 Referências

- [Documentação GoDoc](https://pkg.go.dev/github.com/zarvhq/zarv-go/pkg/gcp/gcs)
- [GCS Official Docs](https://cloud.google.com/storage/docs)

---

## Document AI

Cliente para processamento de documentos com Google Cloud Document AI.

### 📦 Instalação

```bash
go get github.com/zarvhq/zarv-go/pkg/gcp/documentai
```

```go
import "github.com/zarvhq/zarv-go/pkg/gcp/documentai"
```

### 🚀 Uso

```go
package main

import (
    "context"
    "github.com/zarvhq/zarv-go/pkg/gcp/documentai"
)

func main() {
    ctx := context.Background()
    
    // Criar cliente
    client, err := documentai.NewClient(ctx, &documentai.Cfg{
        ProjectID: "my-project",
        Location:  "us", // ou "eu"
        // CredentialsJSON: []byte(`{...}`), // Opcional
    })
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // Processar documento
    document, err := client.ProcessDocument(
        ctx,
        fileBytes,
        "application/pdf",
        "projects/123/locations/us/processors/abc",
    )
    
    // Extrair texto
    text := document.GetText()
}
```

### 🔗 Referências

- [Documentação GoDoc](https://pkg.go.dev/github.com/zarvhq/zarv-go/pkg/gcp/documentai)
- [Document AI Official Docs](https://cloud.google.com/document-ai/docs)

---

## Pub/Sub

Cliente para operações com Google Cloud Pub/Sub (tópicos e subscriptions).

### 📦 Instalação

```bash
go get github.com/zarvhq/zarv-go/pkg/gcp/pubsub
```

```go
import "github.com/zarvhq/zarv-go/pkg/gcp/pubsub"
```

### 🚀 Uso Básico

#### Publisher

```go
package main

import (
    "context"
    "github.com/zarvhq/zarv-go/pkg/gcp/pubsub"
)

func main() {
    ctx := context.Background()
    
    // Conectar ao Pub/Sub
    cfg := &pubsub.Cfg{
        ProjectID: "my-project-id",
        // CredentialsJSON: []byte(`{...}`), // Opcional - usa Workload Identity
    }
    
    client, err := pubsub.NewClient(ctx, cfg)
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // Criar tópico (se não existir)
    err = client.CreateTopic("orders")
    if err != nil {
        panic(err)
    }
    
    // Criar publisher
    publisher, err := client.NewPublisher("orders")
    if err != nil {
        panic(err)
    }
    defer publisher.Stop()
    
    // Publicar mensagem
    messageID, err := publisher.Publish(ctx, map[string]interface{}{
        "order_id": 12345,
        "status":   "pending",
        "amount":   99.99,
    })
    
    // Publicar com atributos
    messageID, err = publisher.PublishWithAttributes(ctx,
        map[string]string{"data": "value"},
        map[string]string{
            "priority": "high",
            "region":   "us-east1",
        },
    )
}
```

#### Subscriber

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "github.com/zarvhq/zarv-go/pkg/gcp/pubsub"
)

// Implementar handler personalizado
type OrderHandler struct{}

func (h *OrderHandler) HandleMessage(data []byte, attributes map[string]string) error {
    var order map[string]interface{}
    if err := json.Unmarshal(data, &order); err != nil {
        return err
    }
    
    log.Printf("Processing order: %v", order)
    log.Printf("Attributes: %v", attributes)
    
    // Processar ordem aqui
    
    return nil // nil = Ack, error = Nack
}

func main() {
    ctx := context.Background()
    
    cfg := &pubsub.Cfg{
        ProjectID: "my-project-id",
    }
    
    client, err := pubsub.NewClient(ctx, cfg)
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // Criar subscription (se não existir)
    err = client.CreateSubscription("orders", "orders-subscription")
    if err != nil {
        panic(err)
    }
    
    // Criar subscriber com handler
    handler := &OrderHandler{}
    subscriber, err := client.NewSubscriber("orders-subscription", handler)
    if err != nil {
        panic(err)
    }
    
    // Iniciar recebimento com 10 workers concorrentes
    err = subscriber.Receive(10)
    if err != nil {
        panic(err)
    }
}
```

### 🔧 Interface SubscriberHandler

```go
type SubscriberHandler interface {
    HandleMessage(data []byte, attributes map[string]string) error
}
```

**Comportamento:**
- Retornar `nil` → mensagem é confirmada (Ack)
- Retornar `error` → mensagem é rejeitada (Nack) e será reenviada

### 🛑 Graceful Shutdown

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Capturar sinais do OS
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

client, _ := pubsub.NewClient(ctx, cfg)
defer client.Close()

subscriber, _ := client.NewSubscriber("subscription-id", handler)

// Executar subscriber em goroutine
errChan := make(chan error, 1)
go func() {
    errChan <- subscriber.Receive(10)
}()

// Aguardar sinal de shutdown
select {
case <-sigChan:
    log.Println("Shutting down gracefully...")
    cancel() // Cancela o context
    
    // Subscriber irá:
    // 1. Parar de aceitar novas mensagens
    // 2. Aguardar mensagens em processamento
    // 3. Retornar nil
    
    if err := <-errChan; err != nil {
        log.Printf("Error: %v", err)
    }
    log.Println("Shutdown complete")
    
case err := <-errChan:
    log.Printf("Subscriber stopped: %v", err)
}
```

### 📋 Funcionalidades

- ✅ Publicação de mensagens em tópicos
- ✅ Consumo de mensagens de subscriptions
- ✅ Serialização automática para JSON
- ✅ Atributos customizados nas mensagens
- ✅ Processamento concorrente configurável
- ✅ Handlers personalizáveis
- ✅ Panic recovery automático
- ✅ Thread-safe
- ✅ Graceful shutdown via context
- ✅ Suporte a Workload Identity

### 🔄 Tópicos vs Subscriptions

#### Tópicos
- Endpoint para publicar mensagens
- Uma mensagem em um tópico pode ser recebida por múltiplas subscriptions
- Similar a "exchanges" no RabbitMQ

#### Subscriptions
- Representa uma fila de mensagens de um tópico
- Cada subscription recebe uma cópia de cada mensagem do tópico
- Similar a "queues" no RabbitMQ

```go
// Um tópico pode ter múltiplas subscriptions
client.CreateTopic("orders")
client.CreateSubscription("orders", "email-service-sub")
client.CreateSubscription("orders", "analytics-sub")
client.CreateSubscription("orders", "billing-sub")

// Cada subscription recebe todas as mensagens do tópico
publisher.Publish(ctx, order) // Vai para todas as 3 subscriptions
```

### ⚙️ Concorrência

```go
// 1 worker (sequencial)
subscriber.Receive(1)

// 10 workers (concorrente)
subscriber.Receive(10)

// 100 workers (alta concorrência)
subscriber.Receive(100)
```

### 🔗 Referências

- [Documentação Completa](../pkg/gcp/pubsub/README.md)
- [Documentação GoDoc](https://pkg.go.dev/github.com/zarvhq/zarv-go/pkg/gcp/pubsub)
- [Pub/Sub Official Docs](https://cloud.google.com/pubsub/docs)

---

## Metrics

Cliente simples para enviar métricas customizadas ao Cloud Monitoring.

### 📦 Instalação

```bash
go get github.com/zarvhq/zarv-go/pkg/gcp/metrics
```

```go
import "github.com/zarvhq/zarv-go/pkg/gcp/metrics"
```

### 🚀 Uso

```go
ctx := context.Background()

client, err := metrics.NewClient(ctx, &metrics.Cfg{
    ProjectID: "my-project",
    // CredentialsJSON: []byte(`{...}`), // opcional, usa WI se vazio
    // Timeout: 5 * time.Second,        // opcional, default 5s se omitido
})
if err != nil { panic(err) }
defer client.Close()

// Enviar um gauge customizado
err = client.WriteGauge(ctx,
    "custom.googleapis.com/myapp/queue_depth",
    map[string]string{"queue": "emails"},
    42,
)
```

### 📋 Interface

```go
type Client interface {
    WriteGauge(ctx context.Context, metricType string, labels map[string]string, value float64) error
    Close() error
}
```

### 🔗 Referências

- [Cloud Monitoring: custom metrics](https://cloud.google.com/monitoring/custom-metrics)
