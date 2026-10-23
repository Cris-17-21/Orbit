# Plan de implementación: Atomic Memory para Orbit

## Resumen

Integrar Atomic Memory mediante su servidor MCP oficial, con memoria Markdown como respaldo y autoridad revisable. La configuración de proyecto definirá solo modalidad y datos no secretos; cada colaborador mantendrá credenciales e identidad en su entorno local. Se habilitará memoria de equipo únicamente para un backend/alcance que pase el piloto con dos colaboradores.

## Decisiones de arquitectura

- Reutilizar `@atomicmemory/mcp-server`; Orbit no implementará un cliente ni almacenamiento paralelo.
- Mantener los adaptadores como recursos namespaced que Orbit pueda planificar, aplicar, verificar y quitar según su manifiesto.
- No escribir API keys ni identidad personal en archivos versionados. Los secretos se resuelven en la configuración local del host o por variables de entorno, una vez verificado el mecanismo de cada host.
- No marcar una conexión como compartida/verificada solo por existir una entrada MCP.
- Iniciar con Codex y Claude Code, cuyas guías oficiales describen MCP. Antigravity se agrega en una entrega posterior, si pasa la validación genérica MCP.
- Hosted Cloud podrá configurarse en modo personal. No se ofrecerá como memoria compartida verificada mientras la autorización por colaborador y el comportamiento del namespace no estén demostrados; omitir `ATOMICMEMORY_SCOPE_NAMESPACE` según la guía actual.
- Comando propuesto para la interfaz CLI: `orbit memory configure|status|disable`; confirmarlo contra las convenciones existentes al implementar el primer corte.

## Dependencias

```text
Piloto de identidad, alcance y MCP
                 │
                 ▼
Configuración no secreta del proyecto y modelo interno
        ┌────────┴─────────┐
        ▼                  ▼
Adaptador Codex      Adaptador Claude Code
        └────────┬─────────┘
                 ▼
Skill de protocolo de memoria + diagnóstico/propiedad
                 │
                 ▼
Piloto de aceptación con dos colaboradores
```

## Tareas

### Fase 0: validar el backend y el alcance compartido

- [ ] Tarea 1: probar identidad, aislamiento y superficie MCP con dos colaboradores usando datos sintéticos. Registrar el backend y versión elegidos, comportamiento de memoria personal/compartida y limitaciones.
  - Bloquea: habilitar la modalidad compartida en Orbit.
  - Verificación: A ingiere una memoria personal y otra compartida; B no recupera la personal y sí recupera la compartida cuando está autorizado; ambos usan credenciales propias; probar los scopes expuestos por el servidor MCP, no solo la API Core.
  - Si falla: conservar la modalidad personal y dejar la compartida como no compatible hasta resolver la causa; no compartir una API key como sustituto de identidad.
  - Archivos: informe de piloto en `docs/atomic-memory-evaluation.md`.
  - Alcance: pequeño, validación manual externa.

### Punto de revisión: decisión de soporte

- [ ] Aprobar backend/modos que pasaron el piloto y las limitaciones documentadas antes de habilitar memoria de equipo en el CLI.

### Fase 1: configuración y propiedad de Orbit

- [x] Tarea 2: definir y probar el modelo/configuración no secreta de Atomic Memory en `.orbit`, integración con manifiesto y vista previa del plan.
  - Aceptación: modalidad apagada o personal (la compartida queda bloqueada hasta validar el piloto); endpoint/provider si aplica; ningún secreto ni identidad personal en el repo; archivos existentes se preservan.
  - Verificación: pruebas de serialización, detección de conflictos, preservación e inspección de que no serializa credenciales.
  - Dependencias: Tarea 1 para habilitar campos de modalidad compartida; la configuración personal puede desarrollarse antes si el piloto tarda. El primer corte modela solo apagado/personal para Hosted Cloud.
  - Archivos probables: paquete nuevo `internal/atomicmemory`, cambios acotados a CLI/manifiesto y pruebas.
  - Alcance: mediano.

### Fase 2: una integración vertical por agente

- [ ] Tarea 3: implementar `orbit memory configure` para Codex mediante su configuración MCP oficial y un plan revisable.
  - Aceptación: la persona ve destinos/cambios; la API key no se muestra, registra ni escribe en el proyecto; colisiones se preservan y reportan; `orbit memory status` distingue configurada de verificada.
  - Verificación: pruebas de generación, configuración preexistente preservada, clave omitida de salida, y comprobación manual con Codex disponible.
  - Dependencias: Tarea 2; Tarea 1 para modo compartido.
  - Archivos probables: adaptador Codex, dispatcher CLI y pruebas correspondientes (máximo cinco archivos por corte).
  - Alcance: mediano.

- [ ] Tarea 4: implementar la misma configuración para Claude Code, manteniendo su mecanismo de confianza/aprobación MCP.
  - Aceptación: integración namespaced; la confianza pendiente se informa como acción manual; entradas ajenas se conservan; secretos siguen fuera de Git.
  - Verificación: pruebas de configuración y preservación, `claude mcp list` y confirmación manual de herramientas en un entorno autenticado.
  - Dependencias: Tarea 2.
  - Archivos probables: adaptador Claude Code, dispatcher CLI y pruebas correspondientes (máximo cinco archivos por corte).
  - Alcance: mediano.

### Punto de revisión: integración por host

- [ ] `go test ./...` y `go vet ./...` pasan; revisión manual confirma que los dos hosts cargan el servidor y que el CLI distingue configuración de conexión.

### Fase 3: uso seguro desde el orquestador

- [ ] Tarea 5: agregar la skill fuente de protocolo Atomic Memory y enlazarla a la skill de entrada de Orbit.
  - Aceptación: búsqueda al iniciar solo cuando esté habilitada; contexto devuelto es no confiable; ingestión se limita a resúmenes/hechos aceptados; no se guardan transcripciones completas automáticamente; falla MCP y el agente continúa con Markdown.
  - Verificación: revisión de instrucciones con casos de memoria propuesta, vigente, reemplazada, irrelevante, servidor no disponible y regla contradictoria.
  - Dependencias: Tareas 2-4.
  - Archivos: skill fuente de Orbit y referencia de instalación/paquete.
  - Alcance: pequeño/mediano.

- [ ] Tarea 6: agregar `orbit memory status`/`orbit doctor` y desactivación segura con manifiesto de propiedad.
  - Aceptación: informa estado configurada/conectividad/alcance verificado por separado; desactiva solo recursos administrados por Orbit; no elimina skills ni servidores ajenos.
  - Verificación: pruebas de doctor, modificaciones de usuario, eliminación segura y escenarios sin credenciales.
  - Dependencias: Tareas 2-5.
  - Archivos: comandos, adaptadores, manifiesto y pruebas dentro del límite de archivos por PR.
  - Alcance: mediano.

### Punto de revisión final

- [ ] Pruebas automatizadas y `go vet ./...` pasan.
- [ ] Piloto con dos identidades pasa para todo modo llamado compartido.
- [ ] Conflictos, salida sin conexión y fallback Markdown comprobados.
- [ ] Revisión del diff confirma ausencia de secretos y cambios ajenos.

## Riesgos y mitigaciones

| Riesgo | Impacto | Mitigación |
| --- | --- | --- |
| El MCP oficial no expone scopes workspace o no autentica a cada colaborador como necesita el equipo | No se puede ofrecer memoria compartida segura desde el adaptador | Probar la interfaz MCP real con dos colaboradores antes de habilitar el modo; mantener personal disponible |
| Hosted Cloud presenta asimetría de namespace/ingesta documentada | Recuerdos ingeridos pueden no recuperarse por filtro de proyecto | Omitir namespace en Cloud mientras persista el defecto; no prometer aislamiento por proyecto |
| Cada host almacena MCP y secretos con convenciones distintas | Filtración o configuración dañada | Verificar documentación oficial actual; preferir CLI/almacén local; vista previa, merge acotado y pruebas de no serialización |
| El contenido recuperado contradice reglas aprobadas | Agente aplica comportamiento obsoleto | Markdown/SDD aprobados son autoridad; conservar estado/procedencia y pedir revisión humana ante conflicto |
| PR2 se integró en la rama de PR1 después del merge de PR1 | `main` puede no tener las guías españolas | Antes de abrir PR de implementación, comparar el árbol de la rama de trabajo con `main` y llevar primero la traducción a `main` en un PR pequeño |

## Preguntas abiertas

- ¿Qué backend y cuenta de Atomic Memory están disponibles para el piloto de dos colaboradores?
- ¿El servidor MCP admite los scopes compartidos requeridos por Core o solo scopes user/agent/namespace/thread?
- ¿Qué forma de secreto por host se puede usar sin guardarlo en el proyecto?
- ¿Se separa el PR para recuperar primero las guías en español a `main`? Recomendación: sí.

## Comandos de verificación

- `go test ./...`
- `go vet ./...`
- `git diff --check`
- Verificaciones manuales documentadas en cada tarea y en el piloto.
