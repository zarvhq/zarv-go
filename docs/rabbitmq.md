# RabbitMQ

Cliente para operações com RabbitMQ, suportando producers e consumers com reconnection automática.

## 📦 Instalação

```bash
go get github.com/zarvhq/zarv-go/pkg/rabbitmq
```

```go
import "github.com/zarvhq/zarv-go/pkg/rabbitmq"
```

## 🚀 Uso Básico

### Producer

```go
package main

import (
    "context"
    "github.com/zarvhq/zarv-go/pkg/rabbitmq"
)

func main() {
    ctx := context.Background()
    
    // Conectar ao RabbitMQ
    client, err := rabbitmq.NewClient(ctx, "amqp://guest:guest@localhost:5672/")
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // Criar producer
    producer, err := client.NewProducer()
    if err != nil {
        panic(err)
    }
    defer producer.Close()
    
    // Publicar mensagem
    err = producer.Publish("orders-queue", map[string]interface{}{
        "order_id": 12345,
        "status":   "pending",
        "amount":   99.99,
    })
    if err != nil {
        panic(err)
    }
}
```

### Consumer

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "github.com/zarvhq/zarv-go/pkg/rabbitmq"
)

// Implementar handler personalizado
type OrderHandler struct{}

func (h *OrderHandler) HandleMessage(data []byte) error {
    var order map[string]interface{}
    if err := json.Unmarshal(data, &order); err != nil {
        return err
    }
    
    log.Printf("Processing order: %v", order)
    
    // Processar ordem aqui
    
    return nil // nil = Ack, error = Nack
}

func main() {
    ctx := context.Background()
    
    // Conectar ao RabbitMQ
    client, err := rabbitmq.NewClient(ctx, "amqp://guest:guest@localhost:5672/")
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // Criar consumer com handler
    handler := &OrderHandler{}
    consumer, err := client.NewConsumer("order-processor", "orders-queue", handler)
    if err != nil {
        panic(err)
    }
    
    // Iniciar consumo com 5 workers concorrentes
    err = consumer.Consume(5)
    if err != nil {
        panic(err)
    }
}
```

## 🔧 Interface QueueHandler

```go
type QueueHandler interface {
    HandleMessage(data []byte) error
}
```

**Comportamento:**
- Retornar `nil` → mensagem é confirmada (Ack)
- Retornar `error` → mensagem é rejeitada (Nack) e volta para a fila

## 🔄 Reconnection Automática

### Producer

O producer reconecta automaticamente quando o canal fecha:

```go
producer, _ := client.NewProducer()

// Se o canal fechar durante Publish(), 
// o producer tenta reconectar automaticamente
err := producer.Publish("queue", data)
if err != nil {
    // Se falhar após retry, retorna erro
    log.Printf("Failed to publish: %v", err)
}
```

**Comportamento:**
- ✅ Detecta canal fechado automaticamente
- ✅ Recria canal em caso de falha
- ✅ Retry transparente
- ✅ Thread-safe
- ❌ Se a reconexão falhar, retorna erro

### Consumer

O consumer **não** reconecta automaticamente. Em caso de falha, retorna erro:

```go
err := consumer.Consume(5)
if err != nil {
    // Canal/conexão fechou com erro
    log.Printf("Consumer failed: %v", err)
    
    // Cabe à aplicação decidir se reconecta
}
```

## 🛑 Graceful Shutdown

O consumer suporta graceful shutdown via contexto:

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "github.com/zarvhq/zarv-go/pkg/rabbitmq"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Capturar sinais do OS
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    
    client, _ := rabbitmq.NewClient(ctx, "amqp://localhost:5672/")
    defer client.Close()
    
    consumer, _ := client.NewConsumer("consumer-1", "orders", handler)
    
    // Executar consumer em goroutine
    errChan := make(chan error, 1)
    go func() {
        errChan <- consumer.Consume(10)
    }()
    
    // Aguardar sinal de shutdown
    select {
    case <-sigChan:
        log.Println("Shutting down gracefully...")
        cancel() // Cancela o context
        
        // Consumer irá:
        // 1. Parar de aceitar novas mensagens
        // 2. Aguardar mensagens em processamento
        // 3. Retornar nil
        
        if err := <-errChan; err != nil {
            log.Printf("Error: %v", err)
            os.Exit(1)
        }
        log.Println("Shutdown complete")
        os.Exit(0)
        
    case err := <-errChan:
        // Erro inesperado (canal/conexão fechou)
        log.Printf("Consumer stopped with error: %v", err)
        os.Exit(1)
    }
}
```

## 📋 Estratégias de Recovery

### Opção 1: Loop de Retry na Aplicação

```go
func main() {
    for {
        err := runConsumer()
        if err == nil {
            // Shutdown graceful
            break
        }
        
        log.Printf("Consumer error: %v. Reconnecting in 5s...", err)
        time.Sleep(5 * time.Second)
    }
}

func runConsumer() error {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Setup de sinais...
    
    client, _ := rabbitmq.NewClient(ctx, url)
    defer client.Close()
    
    consumer, _ := client.NewConsumer("consumer", "queue", handler)
    return consumer.Consume(10)
}
```

### Opção 2: Graceful Shutdown + Supervisor Externo (Recomendado)

Deixe supervisores externos (systemd, Docker, Kubernetes) gerenciarem restarts:

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    fatalErr := make(chan error, 1)
    
    go func() {
        client, _ := rabbitmq.NewClient(ctx, url)
        defer client.Close()
        
        consumer, _ := client.NewConsumer("consumer", "queue", handler)
        fatalErr <- consumer.Consume(10)
    }()
    
    select {
    case <-sigChan:
        cancel()
        <-fatalErr
        os.Exit(0) // Shutdown normal
        
    case err := <-fatalErr:
        log.Printf("Fatal error: %v", err)
        os.Exit(1) // Supervisor irá reiniciar
    }
}
```

**Configuração Systemd:**
```ini
[Unit]
Description=Order Consumer
After=network.target

[Service]
Type=simple
Restart=always
RestartSec=5
ExecStart=/usr/local/bin/consumer
```

**Configuração Docker Compose:**
```yaml
services:
  consumer:
    image: myapp:latest
    restart: unless-stopped
```

**Configuração Kubernetes:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: consumer
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: consumer
        image: myapp:latest
        restartPolicy: Always
```

## 📋 Funcionalidades

- ✅ Producer com reconnection automática
- ✅ Consumer com graceful shutdown
- ✅ Serialização automática para JSON
- ✅ Processamento concorrente configurável
- ✅ Handlers personalizáveis
- ✅ Panic recovery automático
- ✅ Thread-safe
- ✅ QoS configurável
- ✅ Suporte a contexto

## ⚙️ Concorrência

```go
// 1 worker (sequencial)
consumer.Consume(1)

// 5 workers (concorrente)
consumer.Consume(5)

// 50 workers (alta concorrência)
consumer.Consume(50)
```

## 🧪 Testando

### Usando Docker

```bash
docker run -d --name rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  rabbitmq:3-management
```

### Acessar Management UI

```
http://localhost:15672
Usuário: guest
Senha: guest
```

## 🔗 Referências

- [Documentação Completa](../pkg/rabbitmq/README.md)
- [Documentação GoDoc](https://pkg.go.dev/github.com/zarvhq/zarv-go/pkg/rabbitmq)
- [RabbitMQ Official Docs](https://www.rabbitmq.com/documentation.html)
