# Fiber Middleware

Middlewares para aplicações Fiber com autenticação e autorização Zarv.

## 📦 Instalação

```bash
go get github.com/zarvhq/zarv-go/pkg/fiber/v2/middleware
```

```go
import "github.com/zarvhq/zarv-go/pkg/fiber/v2/middleware"
```

## 🚀 Uso Básico

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
            "userId":      profile.UserID,
            "role":        profile.Role,
            "accessLevel": profile.AccessLevel,
        })
    })

    app.Listen(":3000")
}
```

## 📋 Headers Esperados

O middleware `Authenticate` valida os seguintes headers HTTP:

| Header | Descrição | Exemplo |
|--------|-----------|---------|
| `X-Issuer` | Identificador do emissor | `ultron-app`, `vision-app` |
| `X-Workspace-Id` | ID do workspace Zarv | `ws_123456` |
| `X-User-Id` | ID do usuário Zarv | `usr_789012` |
| `X-Zarv-Role` | Role do usuário | `zarver`, `user` |
| `X-Access-Level` | Nível de acesso | `viewer`, `user`, `supervisor`, `admin` |
| `X-Internal` | (Opcional) Requisição interna | `true`, `false` |

### Validação

Se algum header obrigatório estiver faltando, o middleware retorna:

```json
{
  "error": "Missing required header: X-Issuer"
}
```

Status: `401 Unauthorized`

## 🔐 Níveis de Acesso

| Nível | Descrição |
|-------|-----------|
| `viewer` | Acesso somente leitura |
| `user` | Acesso de usuário padrão |
| `supervisor` | Acesso de supervisor com permissões elevadas |
| `admin` | Acesso administrativo completo |

## 🛠️ Métodos do AuthProfile

O `AuthProfile` fornece métodos auxiliares para verificar permissões:

```go
profile := middleware.GetAuthProfile(c)

// Verificar se é administrador Zarv (role zarver + admin/supervisor)
if profile.IsZarvAdmin() {
    // Usuário tem privilégios máximos
}

// Verificar se é administrador do workspace
if profile.IsUserAdmin() {
    // Usuário é admin ou supervisor do workspace
}

// Verificar se tem apenas acesso de visualização
if profile.IsViewer() {
    // Usuário só pode ler
}
```

### Estrutura AuthProfile

```go
type AuthProfile struct {
    Issuer      string // Emissor da requisição
    WorkspaceID string // ID do workspace
    UserID      string // ID do usuário
    Role        string // Role (zarver, user, etc.)
    AccessLevel string // Nível de acesso
    IsInternal  bool   // Se é requisição interna
}
```

## 📝 Exemplos Avançados

### Proteção de Rotas por Nível

```go
// Rota apenas para admins
app.Get("/api/admin/*", func(c *fiber.Ctx) error {
    profile := middleware.GetAuthProfile(c)
    
    if !profile.IsUserAdmin() {
        return c.Status(403).JSON(fiber.Map{
            "error": "Admin access required",
        })
    }
    
    return c.JSON(fiber.Map{"status": "ok"})
})

// Rota que bloqueia viewers
app.Post("/api/resource", func(c *fiber.Ctx) error {
    profile := middleware.GetAuthProfile(c)
    
    if profile.IsViewer() {
        return c.Status(403).JSON(fiber.Map{
            "error": "Cannot modify resources with viewer access",
        })
    }
    
    // Processar criação
    return c.JSON(fiber.Map{"created": true})
})
```

### Rota apenas para Zarv Admins

```go
app.Delete("/api/dangerous-action", func(c *fiber.Ctx) error {
    profile := middleware.GetAuthProfile(c)
    
    if !profile.IsZarvAdmin() {
        return c.Status(403).JSON(fiber.Map{
            "error": "Zarv admin privileges required",
        })
    }
    
    // Ação sensível
    return c.JSON(fiber.Map{"deleted": true})
})
```

### Logging de Requisições

```go
app.Use(middleware.Authenticate)

app.Use(func(c *fiber.Ctx) error {
    profile := middleware.GetAuthProfile(c)
    
    log.Printf("Request from user %s in workspace %s (role: %s, level: %s)",
        profile.UserID,
        profile.WorkspaceID,
        profile.Role,
        profile.AccessLevel,
    )
    
    return c.Next()
})
```

## 🧪 Testando

### Headers de Teste

```bash
curl -X GET http://localhost:3000/api/resource \
  -H "X-Issuer: test-app" \
  -H "X-Workspace-Id: ws_test123" \
  -H "X-User-Id: usr_test456" \
  -H "X-Zarv-Role: user" \
  -H "X-Access-Level: admin"
```

### Mock de AuthProfile em Testes

```go
// Em seus testes
func TestProtectedRoute(t *testing.T) {
    app := fiber.New()
    
    // Configurar middleware de teste
    app.Use(func(c *fiber.Ctx) error {
        c.Locals("authProfile", &middleware.AuthProfile{
            Issuer:      "test",
            WorkspaceID: "ws_test",
            UserID:      "usr_test",
            Role:        "user",
            AccessLevel: "admin",
            IsInternal:  false,
        })
        return c.Next()
    })
    
    app.Get("/api/test", yourHandler)
    
    // Testar...
}
```

## 🔗 Referências

- [Fiber Framework](https://gofiber.io/)
- [Documentação GoDoc](https://pkg.go.dev/github.com/zarvhq/zarv-go/pkg/fiber/v2/middleware)
