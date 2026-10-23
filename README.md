# Orbit

Orbit es un conjunto de herramientas colaborativas para trabajar con agentes de programación mediante desarrollo guiado por especificaciones (SDD).

El proyecto se diseña y construye por etapas. La dirección prevista incluye un CLI en Go, flujos y skills compartidos para SDD, adaptadores para Codex, Claude Code y Antigravity, roles especializados de subagentes, memoria de proyecto en Markdown e integración opcional con Atomic Memory.

Consulta la [especificación del producto](docs/product-spec.md) para conocer el diseño consolidado de Orbit, la [orquestación de tareas](docs/task-orchestration.md) para el flujo propuesto y el [contrato de memoria](docs/memory-contract.md) para el formato inicial del contexto persistente del proyecto.

## Compilar el CLI desde el código fuente

Para compilar desde el código fuente se requiere Go 1.24 o posterior. Ejecuta `go test ./...` y `go build -o <ruta-de-salida>/orbit ./cmd/orbit`. Desde un proyecto Git, ejecuta `orbit init` para revisar los archivos que se crearán. Añade `--yes` solo si quieres aplicar el plan sin una confirmación interactiva.
