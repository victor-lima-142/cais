# Cais

Uma API HTTP pequena para Go, com rotas, grupos, middleware e respostas JSON.

```go
package main

import (
	"log"
	"net/http"

	cais "github.com/victor-lima-142/cais"
)

func main() {
	engine := cais.NewEngine()
	engine.Get("/hello", func(_ cais.Request, response cais.Response) {
		response.Respond(map[string]string{"message": "Olá!"}).Status(http.StatusOK)
	})
	log.Fatal(engine.Run(":8080"))
}
```

Para testar o pacote, execute `go test ./...`.
