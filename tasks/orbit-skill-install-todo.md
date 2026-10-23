# Tareas: instalación de skills de Orbit por proyecto

Referencia: [especificación](../docs/specs/orbit-project-skill-install.md) · [plan](orbit-skill-install-plan.md)

## T-2.1 — Planificación y aplicación genérica de archivos administrados

- [x] **Aceptación:** Planifica recursos de archivo sin escrituras; identifica nuevas rutas, rutas Orbit intactas y conflictos por archivo ajeno, modificación o propiedad incompatible.
- [x] **Aceptación:** La aplicación exige que el estado coincida con el plan; valida la raíz, rutas y symlinks; escribe recursos y manifiesto con recuperación ante error.
- [x] **Aceptación:** Puede crear un manifiesto si no existe y extender uno existente sin perder recursos de `orbit init` o Atomic Memory.
- [x] **Verificar:** pruebas del paquete `internal/initmemory` para estados, manifiestos, rutas inseguras, conflictos, cambios entre plan y aplicación, rollback y preservación byte por byte.
- [ ] **Dependencias:** especificación y plan aceptados.
- [ ] **Archivos probables:** `internal/initmemory/managed.go`, `internal/initmemory/managed_test.go` (y hasta tres adicionales si la separación lo requiere).
- [ ] **PR:** servicio genérico y pruebas; sin nuevo comando de usuario.

## Checkpoint T-2.1

- [x] Revisar que el manifiesto se pueda crear y extender sin cambiar las garantías de `orbit init` ni de configuración MCP.
- [ ] Revisar y aceptar la interfaz reutilizable antes de añadir adaptadores.

## T-2.2 — Fuente empaquetada y destinos por agente

- [ ] **Aceptación:** El CLI puede leer del binario una copia byte por byte de `skills/orbit/SKILL.md` sin red ni fuente duplicada.
- [ ] **Aceptación:** Codex/Antigravity comparten y deduplican `.agents/skills/orbit/SKILL.md`; Claude Code usa `.claude/skills/orbit/SKILL.md`.
- [ ] **Verificar:** pruebas de contenido embebido, rutas, IDs y combinaciones de agentes.
- [ ] **Dependencias:** T-2.1.
- [ ] **Archivos probables:** paquete de assets dentro de `skills/`; `internal/skillinstall/` y pruebas.
- [ ] **PR:** empaquetado y resolución; sin modificar archivos del usuario.

## Checkpoint T-2.2

- [ ] Revisar los destinos frente a la documentación actual de los tres hosts.
- [ ] Aceptar el plan completo por agente antes de conectar la interfaz CLI.

## T-2.3 — Comando `orbit skill install`

- [ ] **Aceptación:** Selección de agentes, vista previa, confirmación y mensajes de resultado en español.
- [ ] **Aceptación:** `--dry-run` no escribe; modo no interactivo necesita `--yes`; cancelar no escribe.
- [ ] **Aceptación:** Conflictos se omiten sin reemplazos; destinos independientes sí pueden instalarse y registrarse.
- [ ] **Verificar:** pruebas de argumentos, selección, cancelación, plan seco, aplicación y conflicto parcial; `go test ./...`, `go vet ./...` y revisión manual de un proyecto temporal.
- [ ] **Dependencias:** T-2.1 y T-2.2.
- [ ] **Archivos probables:** `cmd/orbit/main.go`, `cmd/orbit/main_test.go`; actualización mínima del README si corresponde.
- [ ] **PR:** una entrega enfocada del comando y su documentación.

## Checkpoint final

- [ ] AC-1 a AC-10 de la especificación tienen evidencia.
- [ ] Los archivos instalados y el manifiesto quedan correctos tanto antes como después de `orbit init`.
- [ ] El PR enumera criterios, validación, hosts comprobados y límites de la verificación runtime.
