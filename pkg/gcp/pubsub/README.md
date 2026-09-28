# Google Cloud Pub/Sub Package

Cliente para operações com Google Cloud Pub/Sub, suportando tanto tópicos quanto filas (subscriptions).

## 📦 Instalação

```bash
go get github.com/zarvhq/zarv-go/pkg/gcp/pubsub
```

```go
import "github.com/zarvhq/zarv-go/pkg/gcp/pubsub"
```

## 🚀 Uso

### Publisher (Publicar mensagens em tópicos)

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
        // CredentialsJSON: []byte(`{...}`), // Opcional - usa Workload Identity se omitido
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
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Published message: %s\n", messageID)
    
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

#### Publicação assíncrona (sem bloquear a request)

`PublishAsync` enfileira a mensagem e retorna imediatamente, sem esperar o
round-trip com o servidor. O resultado (ID ou erro) chega pelo callback. Ideal
para caminhos sensíveis a latência, como emissão de eventos de auditoria, onde
a publicação não pode adicionar latência à request. Chame `Stop()` no shutdown
para drenar as mensagens pendentes.

```go
publisher.PublishAsync(ctx,
    map[string]string{"data": "value"},
    nil, // atributos
    func(id string, err error) {
        if err != nil {
            log.Printf("falha ao publicar: %v", err)
            return
        }
        log.Printf("publicado: %s", id)
    },
)
// callback pode ser nil (fire-and-forget)
```

#### Publisher sem checagem de existência do tópico

Por padrão `NewPublisher` valida que o tópico existe, o que exige a permissão
`pubsub.topics.get`. Uma service account somente de publicação
(`roles/pubsub.publisher`) não tem essa permissão; passe
`WithoutTopicExistsCheck()` para pular a checagem quando o tópico já é conhecido:

```go
publisher, err := client.NewPublisher("meu-topico", pubsub.WithoutTopicExistsCheck())
```

#### Batching (flush agrupado)

O cliente já agrupa mensagens por padrão: cada `Publish`/`PublishAsync` enfileira
no batch atual, que é enviado quando algum limite é atingido — `DelayThreshold`
(10ms), `CountThreshold` (100 mensagens) ou `ByteThreshold` (1MB). Basta manter
**um** publisher vivo (não crie um por mensagem) para que muitas mensagens virem
poucas requisições; `Stop()` faz o flush final. Para volumes altos (ex.: eventos
de auditoria por request), aumente os limites com `WithBatchSettings`, que parte
dos defaults e só altera o que você tocar:

```go
publisher, err := client.NewPublisher("meu-topico",
    pubsub.WithBatchSettings(func(s *pubsub.PublishSettings) {
        s.DelayThreshold = 100 * time.Millisecond // espera até 100ms para formar batches maiores
        s.CountThreshold = 500                     // ou 500 mensagens
    }),
)
```

### Subscriber (Consumir mensagens de subscriptions)

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "github.com/zarvhq/zarv-go/pkg/gcp/pubsub"
)

// Implementar o handler personalizado
type OrderHandler struct{}

func (h *OrderHandler) HandleMessage(data []byte, attributes map[string]string) error {
    var order map[string]interface{}
    if err := json.Unmarshal(data, &order); err != nil {
        return err
    }
    
    log.Printf("Processing order: %v", order)
    log.Printf("Attributes: %v", attributes)
    
    // Processar a ordem aqui
    
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

## 🔧 Interface SubscriberHandler

Para consumir mensagens, implemente a interface `SubscriberHandler`:

```go
type SubscriberHandler interface {
    HandleMessage(data []byte, attributes map[string]string) error
}
```

**Comportamento:**
- Retornar `nil` → mensagem é confirmada (Ack)
- Retornar `error` → mensagem é rejeitada (Nack) e será reenviada

## 📋 Funcionalidades

- ✅ Publicação de mensagens em tópicos
- ✅ Consumo de mensagens de subscriptions
- ✅ Serialização automática para JSON
- ✅ Atributos customizados nas mensagens
- ✅ Processamento concorrente
- ✅ Handlers personalizáveis
- ✅ Panic recovery automático
- ✅ Thread-safe
- ✅ Graceful shutdown via context

## 🛑 Graceful Shutdown

O subscriber suporta graceful shutdown via contexto:

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "github.com/zarvhq/zarv-go/pkg/gcp/pubsub"
)

func main() {
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
}
```

## 🔄 Tópicos vs Subscriptions

### Tópicos
- Endpoint para publicar mensagens
- Uma mensagem em um tópico pode ser recebida por múltiplas subscriptions
- Similar a "exchanges" no RabbitMQ

### Subscriptions
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

## ⚙️ Concorrência

O método `Receive(concurrency int)` controla quantos workers processarão mensagens:

```go
// 1 worker (sequencial)
subscriber.Receive(1)

// 10 workers (concorrente)
subscriber.Receive(10)

// 100 workers (alta concorrência)
subscriber.Receive(100)
```

## 🔐 Autenticação

Por padrão, usa **Workload Identity** no GKE. Para usar credenciais JSON:

```go
cfg := &pubsub.Cfg{
    ProjectID:       "my-project",
    CredentialsJSON: []byte(`{"type": "service_account", ...}`),
}
```

## 🛡️ Tratamento de Erros

- Se `HandleMessage` retornar erro, a mensagem será rejeitada (Nack) e reenviada
- Se `HandleMessage` retornar `nil`, a mensagem será confirmada (Ack)
- Panics são capturados automaticamente e a mensagem é rejeitada

## 🔒 Thread Safety

- **Publisher.Publish()**: Thread-safe, pode ser chamado por múltiplas goroutines
- **Subscriber.Receive()**: Deve ser chamado apenas uma vez por subscriber
- **Client**: Thread-safe para criar múltiplos publishers/subscribers
