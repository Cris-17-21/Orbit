# Orbit

Orbit es un conjunto de herramientas colaborativas para trabajar con agentes de programación mediante desarrollo guiado por especificaciones (SDD).

El proyecto se diseña y construye por etapas. La dirección prevista incluye un CLI en Go, flujos y skills compartidos para SDD, adaptadores para Codex, Claude Code y Antigravity, roles especializados de subagentes, memoria de proyecto en Markdown e integración opcional con Atomic Memory.

Consulta la [especificación del producto](docs/product-spec.md) para conocer el diseño consolidado de Orbit, la [orquestación de tareas](docs/task-orchestration.md) para el flujo propuesto y el [contrato de memoria](docs/memory-contract.md) para el formato inicial del contexto persistente del proyecto.

## Compilar el CLI desde el código fuente

Para compilar desde el código fuente se requiere Go 1.24 o posterior. Ejecuta `go test ./...` y `go build -o <ruta-de-salida>/orbit ./cmd/orbit`. Desde un proyecto Git, ejecuta `orbit init` para revisar los archivos que se crearán. Añade `--yes` solo si quieres aplicar el plan sin una confirmación interactiva.

### Atomic Memory con Codex

Después de `orbit init`, ejecuta `orbit memory configure codex` para revisar y activar el modo personal de Atomic Memory. El comando detecta un servidor Codex llamado `atomicmemory` para evitar duplicarlo. Si no existe, propone `.codex/config.toml`, que Codex solo carga en proyectos de confianza.

Usa `orbit memory status` para ver la modalidad configurada y si los archivos MCP del proyecto contienen la entrada `atomicmemory` para Codex o Claude Code. Este diagnóstico no comprueba la conexión del agente ni valida que la memoria sea compartida entre colaboradores.

Para desactivar la integración del proyecto, ejecuta `orbit memory disable`. Orbit muestra primero qué configuraciones intactas de su propiedad quitará; los archivos o entradas que hayan cambiado se conservan. La memoria Markdown del proyecto no se elimina. En entornos no interactivos, confirma la vista previa con `orbit memory disable --yes`.

`orbit doctor` amplía el diagnóstico: detecta si están disponibles los CLI de Codex y Claude Code, consulta sus listas MCP y muestra únicamente si las variables locales requeridas están definidas, nunca sus valores. Codex informa servidores registrados; Claude Code informa el estado que muestra `claude mcp list`. Ninguno de esos resultados valida que la memoria sea compartida.

Orbit nunca guarda la clave API ni la identidad del colaborador en el repositorio. Cada persona configura localmente `ATOMICMEMORY_API_KEY` y `ATOMICMEMORY_SCOPE_USER` antes de iniciar Codex. La configuración de proyecto declara esas variables como referencias del entorno. Para elegir otro backend, usa `--backend connected-local` o `--backend self-hosted --api-url https://tu-servidor`; `--yes` aplica el plan sin confirmación interactiva.
