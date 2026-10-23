# Plan de implementación: instalación de skills de Orbit por proyecto

- especificación: [instalar la skill de Orbit en un proyecto](../docs/specs/orbit-project-skill-install.md)
- estado: T-2.1 integrada en main mediante PR #9; T-2.2 en progreso
- lista de tareas: [orbit-skill-install-todo.md](orbit-skill-install-todo.md)
- alcance: T-2, instalar la skill de entrada en proyectos con Codex, Claude Code y/o Antigravity
- separación: este plan no reemplaza [el plan de Atomic Memory](plan.md) ni su lista de tareas.

## Resultado

Entregar `orbit skill install` como una instalación local al proyecto, revisable y administrada por manifiesto. La skill canónica se empaqueta junto con el CLI, sin solicitar red en tiempo de instalación. Se admiten los destinos de proyecto de Codex, Claude Code y Antigravity; la ruta compartida de Codex y Antigravity se deduplica.

El plan cubre solamente skills. No configura Atomic Memory, perfiles de subagentes ni ajustes personales. La salida confirma presencia de archivos en el proyecto, no activación comprobada dentro de un host.

## Decisiones propuestas

- Requerir una raíz Git y usarla como límite del alcance proyecto, siguiendo el comportamiento actual de `orbit init` y permitiendo versionar juntos la skill y su manifiesto.
- Mantener `skills/orbit/SKILL.md` como fuente canónica única. Empaquetarla con `go:embed` desde un paquete pequeño en el directorio `skills/`, evitando copiarla a otra carpeta fuente o descargarla al instalar.
- Resolver los destinos con adaptadores de skill: Codex y Antigravity → `.agents/skills/orbit/SKILL.md`; Claude Code → `.claude/skills/orbit/SKILL.md`. Seleccionar Codex y Antigravity produce un solo recurso.
- Añadir una operación genérica de planificación/aplicación de archivos administrados a `internal/initmemory`, reutilizando formato y esquema de `.orbit/manifest.json`. La operación debe poder crear un manifiesto válido si aún no existe o extenderlo si `orbit init` ya creó uno.
- Resolver propiedad por ruta y `resource_id` estable. Un recurso nuevo se instala; un recurso intacto de Orbit se reporta como ya presente; un destino preexistente ajeno, modificado, huérfano o con propiedad incompatible se conserva y se reporta como conflicto.
- Mostrar planes interactivos antes de escribir. `--dry-run` muestra el plan sin escrituras; el uso no interactivo requiere `--yes`. Los destinos sin conflicto pueden instalarse aunque otro agente seleccionado tenga un conflicto.
- No detectar binarios de agentes ni requerir que Codex, Claude Code o Antigravity estén instalados para copiar la skill del proyecto. Evita que el instalador afirme compatibilidad de runtime no comprobada.

## Flujo de componentes

```text
skills/orbit/SKILL.md
          │ go:embed
          ▼
Paquete de assets de skills ──► Adaptadores: host → destino
                                         │
                                         ▼
                            Plan genérico de archivo administrado
                                         │
                           confirmación / dry-run / --yes
                                         ▼
                       Aplicación transaccional + manifiesto
                                         │
                                         ▼
                               Resumen en español
```

La planificación inspecciona el manifiesto, los archivos destino y las carpetas padre sin escribir. La aplicación vuelve a verificar el estado observado antes de modificarlo. El manifiesto solo registra rutas propias de la skill, nunca la carpeta completa `.agents/skills` o `.claude/skills`.

## Tareas

### T-2.1 — Planificación y aplicación genérica de archivos de skill

**Resultado:** Orbit puede planificar y aplicar un conjunto de archivos administrados, creando o extendiendo el manifiesto sin reclamar ni reemplazar archivos ajenos.

**Criterios:**
- La planificación representa destinos ausentes, ya presentes sin cambios, modificados, ajenos y con propiedad incompatible.
- El plan incluye hashes/estado observado para rechazar cambios ocurridos entre vista previa y aplicación.
- La aplicación vuelve a validar rutas dentro de la raíz, evita symlinks y cambios externos, escribe recursos y manifiesto de forma recuperable.
- Puede crear el manifiesto cuando el usuario todavía no ejecutó `orbit init`; después `orbit init` lo reconoce y añade su memoria sin perder el registro de la skill.
- Los conflictos no impiden preparar recursos independientes; ningún archivo ajeno cambia.

**Verificación:** pruebas unitarias de planner y escritura en proyectos Git temporales, cubriendo manifiesto ausente/válido/inválido, destinos nuevos/presentes/alterados, colisiones, rutas inseguras, carrera tras el plan y preservación de recursos existentes.

**Archivos probables:** `internal/initmemory/managed.go`, `internal/initmemory/managed_test.go` y, si hace falta separar la API, un archivo adicional del mismo paquete.

**Dependencias:** ninguna aparte de la especificación aceptada.

**Alcance:** mediano, máximo cinco archivos. Sin interfaz de usuario todavía.

### Punto de revisión — servicio de propiedad administrada

- [x] Los planes y escrituras conservan recursos existentes y las pruebas del paquete pasan.
- [x] Revisar el manejo de manifiesto ausente y el comportamiento cuando la escritura del manifiesto falla.
- [x] Aceptar la abstracción en revisión antes de continuar a T-2.2; integrada en `main` mediante PR #9 sin cambiar el comportamiento de `orbit init` ni de la configuración MCP actual.

### T-2.2 — Fuente empaquetada y resolución de destinos por agente

**Resultado:** La skill fuente canónica está disponible desde el binario de Orbit y cada agente objetivo produce sus destinos de proyecto correctos.

**Criterios:**
- `go:embed` empaqueta `skills/orbit/SKILL.md` sin duplicar su fuente ni usar red al instalar.
- Codex y Antigravity resuelven la misma ruta de proyecto y se deduplican al seleccionarlos juntos.
- Claude Code resuelve su carpeta de proyecto documentada.
- Los adaptadores producen recursos con IDs estables y descriptivos que el plan de propiedad puede registrar.

**Verificación:** pruebas de assets y destinos; comprobar bytes empaquetados contra el archivo fuente y probar las combinaciones de agentes.

**Archivos probables:** `skills/embed.go`, prueba correspondiente y paquete nuevo `internal/skillinstall` con su prueba.

**Dependencias:** T-2.1.

**Alcance:** mediano, máximo cinco archivos. Todavía sin comando CLI.

### T-2.3 — Comando `orbit skill install`

**Resultado:** La persona elige los agentes y puede revisar o aplicar la instalación al proyecto actual.

**Criterios:**
- `orbit skill install` ofrece elegir Codex, Claude Code, Antigravity o combinaciones; flags de agente pueden seleccionar destinos de forma directa.
- La vista previa lista cada destino, acción y conflicto. `--dry-run` nunca escribe; `--yes` aplica exactamente el plan presentado en modo no interactivo.
- Antes de aplicar, se acepta el plan interactivo; cancelar no deja cambios parciales.
- Se requieren raíz Git y manifiesto interpretable; un manifiesto incompatible detiene el comando antes de escribir.
- Se informan instalados, ya presentes, omitidos y conflictos con sus rutas y mensajes en español. No se asegura que un agente haya cargado la skill.
- El plan y la salida permanecen coherentes cuando existe un conflicto para un host y destinos libres para otro.

**Verificación:** pruebas del dispatcher/flags, confirmación, `--dry-run`, `--yes`, selección combinada y errores. Ejecución de `go test ./...` y `go vet ./...` tras completar el corte, además de una inspección manual de los archivos resultantes en un repo temporal.

**Archivos probables:** `cmd/orbit/main.go`, `cmd/orbit/main_test.go` y ajustes mínimos a README/ayuda solo si son necesarios para documentar el comando.

**Dependencias:** T-2.1 y T-2.2.

**Alcance:** mediano, máximo cinco archivos. Un PR de instalación CLI.

## Checkpoint final

- [ ] AC-1 a AC-10 de la especificación están ligados a evidencia.
- [ ] `go test ./...` y `go vet ./...` pasan.
- [ ] `orbit init` conserva sus recursos cuando la skill se instala antes o después.
- [ ] Los tres hosts detectan la skill en el destino de proyecto correspondiente según sus ubicaciones documentadas; si no se dispone de un host para verificación runtime, se registra esa limitación y no se presenta como validado.
- [ ] Revisión del diff, idioma y conflictos; PR pequeño con enlace a especificación y evidencias.

## Riesgos y mitigaciones

| Riesgo | Impacto | Mitigación |
| --- | --- | --- |
| La operación genérica de archivos altera comportamiento de memoria/MCP | Regresión en comandos existentes | T-2.1 cambia solo interfaz reutilizable; pruebas de regresión para `orbit init` y `orbit memory configure`. |
| Diferencia entre vista previa y aplicación | Se sobrescribe un archivo cambiado por la persona | Snapshot de manifiesto/destinos y revalidación bajo bloqueo antes de escribir. |
| Fallo al escribir el manifiesto después de los archivos | Recursos instalados sin propiedad trazable | Aplicación transaccional y rollback de archivos creados/modificados. |
| Skill empaquetada se separa de la fuente canónica | El binario instala texto antiguo | `go:embed` directo y prueba que compara contenido embebido con la fuente. |
| Agente no disponible durante CI | Matriz runtime parcial | Separar pruebas de rutas/formato de las comprobaciones manuales de descubrimiento; informar alcance de evidencia. |

## Preguntas para la revisión del plan

- El plan propone exigir una raíz Git para mantener el límite de proyecto y el manifiesto coherentes con `orbit init`.
- El plan propone una selección interactiva explícita cuando no se especifican agentes, sin autodetectar instalaciones locales.
- El empaquetado propuesto mantiene el archivo fuente canónico en `skills/orbit/SKILL.md` con un paquete Go en `skills/` que lo incluye mediante `go:embed`.

## Aceptación del plan

- aceptado por: Cris (responsable de Orbit)
- aceptado el: 2026-10-23
- nota: se autoriza implementar los cortes secuencialmente, empezando por T-2.1. Los cambios técnicos propuestos no amplían la especificación aceptada.
