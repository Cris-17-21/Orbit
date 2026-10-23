# Tareas: instalación de skills de Orbit por proyecto

Referencia: [especificación](../docs/specs/orbit-project-skill-install.md) · [plan](orbit-skill-install-plan.md)

## T-2.1 — Planificación y aplicación genérica de archivos administrados

- [x] **Aceptación:** Planifica recursos de archivo sin escrituras; identifica nuevas rutas, rutas Orbit intactas y conflictos por archivo ajeno, modificación o propiedad incompatible.
- [x] **Aceptación:** La aplicación exige que el estado coincida con el plan; valida la raíz, rutas y symlinks; escribe recursos y manifiesto con recuperación ante error.
- [x] **Aceptación:** Puede crear un manifiesto si no existe y extender uno existente sin perder recursos de `orbit init` o Atomic Memory.
- [x] **Verificar:** pruebas del paquete `internal/initmemory` para estados, manifiestos, rutas inseguras, conflictos, cambios entre plan y aplicación, rollback y preservación byte por byte.
- [x] **Dependencias:** especificación y plan aceptados.
- [x] **Archivos probables:** `internal/initmemory/managed.go`, `internal/initmemory/managed_test.go`.
- [x] **PR:** servicio genérico y pruebas; PR #9 integrado en `main`.

## Checkpoint T-2.1

- [x] Revisar que el manifiesto se pueda crear y extender sin cambiar las garantías de `orbit init` ni de configuración MCP.
- [x] Revisar y aceptar la interfaz reutilizable antes de añadir adaptadores; aceptación registrada al integrar el PR #9.

## T-2.2 — Fuente empaquetada y destinos por agente

Estado actual: integrado en `main` mediante PR #10. Rutas revisadas en documentación oficial de Codex, Antigravity y Claude Code el 2026-10-23.

- [x] **Aceptación:** El CLI puede leer del binario una copia byte por byte de `skills/orbit/SKILL.md` sin red ni fuente duplicada.
- [x] **Aceptación:** Codex/Antigravity comparten y deduplican `.agents/skills/orbit/SKILL.md`; Claude Code usa `.claude/skills/orbit/SKILL.md`.
- [x] **Verificar:** pruebas de contenido embebido, rutas, IDs y combinaciones de agentes.
- [x] **Dependencias:** T-2.1, integrada en `main` mediante PR #9.
- [x] **Archivos probables:** `skills/embed.go`, `internal/skillinstall/resolve.go` y su prueba.
- [x] **PR:** PR #10 integrado en `main`; sin modificar archivos de proyectos.

## Checkpoint T-2.2

- [x] Revisar los destinos frente a la documentación oficial actual de los tres hosts (2026-10-23; enlaces en el plan y la especificación).
- [x] Aceptar el plan completo por agente antes de conectar la interfaz CLI.

## T-2.3 — Comando `orbit skill install`

Estado actual: implementado y verificado localmente; pendiente revisión del PR.

- [x] **Aceptación:** Selección interactiva o por flags, vista previa, confirmación y mensajes de resultado en español.
- [x] **Aceptación:** `--dry-run` no escribe; modo no interactivo necesita `--yes`; cancelar no escribe.
- [x] **Aceptación:** Conflictos se conservan sin reemplazos; destinos independientes se pueden instalar y registrar.
- [x] **Verificar:** pruebas de argumentos, selección, cancelación, plan seco, aplicación, conflicto parcial e integración de `orbit init` antes y después; `go test ./... -count=1`, `go vet ./...` y `git diff --check`.
- [x] **Dependencias:** T-2.1 y T-2.2 integradas en `main` mediante PR #9 y PR #10.
- [x] **Archivos probables:** `cmd/orbit/main.go`, `cmd/orbit/main_test.go`.
- [ ] **PR:** entrega enfocada del comando pendiente de creación/revisión.

## Checkpoint final

- [ ] AC-1 a AC-10 de la especificación tienen evidencia.
- [x] Los archivos instalados y el manifiesto quedan correctos tanto antes como después de `orbit init`, mediante prueba de comandos en ambos órdenes.
- [ ] El PR enumera criterios, validación, hosts comprobados y límites de la verificación runtime.
