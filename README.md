# Orbit

Orbit prepara un flujo de trabajo colaborativo para desarrollar software con agentes de IA. Su CLI inicializa memoria de proyecto en Markdown, instala la skill de Orbit para el agente que elijas y ofrece una integración opcional con Atomic Memory.

La memoria Markdown queda versionada con el proyecto y sirve como contexto compartido revisable. Atomic Memory es opcional y, en esta versión, se configura en modo personal para Codex o Claude Code; no habilita por sí sola una memoria compartida entre colaboradores.

## Requisitos

- Go 1.24 o posterior para instalar o compilar el CLI.
- Git y un proyecto que ya sea un repositorio Git para ejecutar `orbit init` e instalar la skill.
- Codex, Claude Code o Antigravity instalado si quieres que Orbit prepare la skill para ese agente.

## Instalar el CLI

Con Go instalado, ejecuta:

```powershell
go install github.com/Cris-17-21/Orbit/cmd/orbit@latest
```

Comprueba la instalación:

```powershell
orbit help
```

Si PowerShell no encuentra `orbit`, agrega la carpeta de binarios de Go a `PATH`. Puedes consultarla con `go env GOPATH`; en Windows normalmente es la carpeta `bin` dentro de esa ruta (por ejemplo, `$env:USERPROFILE\go\bin`). Abre una terminal nueva después de actualizar `PATH`.

### Compilar desde el código fuente

```powershell
git clone https://github.com/Cris-17-21/Orbit.git
Set-Location Orbit
go build -o orbit.exe ./cmd/orbit
.\orbit.exe help
```

## Preparar un proyecto

Desde la raíz de tu proyecto Git, inicializa el contexto versionado de Orbit:

```powershell
orbit init
```

Revisa el plan que muestra el CLI y confirma la creación de los archivos. Para aplicar sin confirmación interactiva, usa `orbit init --yes`.

Instala la skill de Orbit para uno o más agentes. El modo interactivo permite elegirlos:

```powershell
orbit skill install
```

O indica los agentes directamente:

```powershell
orbit skill install --codex
orbit skill install --claude
orbit skill install --antigravity
```

Orbit mantiene estos archivos dentro del proyecto: Codex y Antigravity usan `.agents/skills/orbit/`, y Claude Code usa `.claude/skills/orbit/`. Usa `--dry-run` para revisar el plan sin escribir archivos o `--yes` para aplicarlo sin confirmación interactiva. Instalar la skill de Orbit no impide instalar ni usar otras skills.

## Atomic Memory (opcional)

Orbit puede preparar Atomic Memory para la persona actual y el agente compatible que elijas. Primero configura localmente `ATOMICMEMORY_API_KEY` y `ATOMICMEMORY_SCOPE_USER`; no pongas esos valores en el repositorio ni los compartas en archivos versionados.

Para Codex:

```powershell
orbit memory configure codex
```

Para Claude Code:

```powershell
orbit memory configure claude
```

Revisa la vista previa antes de confirmar. Puedes consultar el estado con `orbit memory status` y ejecutar `orbit doctor` para revisar la configuración y las herramientas disponibles. Estos diagnósticos no verifican que los colaboradores compartan una misma memoria. La modalidad compartida aún no está habilitada; usa los archivos Markdown versionados como contexto común del proyecto.

## Comandos disponibles

```text
orbit help                         Muestra la ayuda
orbit init                         Inicializa la memoria Markdown del proyecto
orbit skill install                Instala la skill de Orbit para agentes elegidos
orbit memory status                Consulta el estado de Atomic Memory
orbit memory configure codex       Configura Atomic Memory para Codex
orbit memory configure claude      Configura Atomic Memory para Claude Code
orbit memory disable               Desactiva la integración de Atomic Memory del proyecto
orbit doctor                       Diagnostica la configuración local
```

Agrega los documentos Markdown de contexto y reglas del proyecto a Git para que el equipo pueda revisarlos y compartirlos mediante sus PRs. Consulta el [contrato de memoria](docs/memory-contract.md), la [especificación del producto](docs/product-spec.md) y el [flujo de orquestación de tareas](docs/task-orchestration.md) para conocer el diseño de Orbit.
