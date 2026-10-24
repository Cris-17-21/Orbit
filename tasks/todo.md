# Tareas: Atomic Memory para Orbit

- [ ] **Tarea 1 — Piloto de identidad, aislamiento y MCP**
  - Aceptación: dos colaboradores acceden con identidades propias; se comprueba aislamiento personal y recuperación compartida autorizada; queda registrado backend, versión y alcance real expuesto por MCP.
  - Verificar: usar datos sintéticos, probar personal y compartido, limpiar el espacio temporal; no inferir identidad a partir de compartir una API key.
  - Dependencias: acceso a Atomic Memory y colaboración de una segunda persona.
  - Archivos: `docs/atomic-memory-evaluation.md`.
  - Alcance: pequeño, requiere validación manual externa.

- [ ] **Revisión humana — modalidades que Orbit puede declarar compatibles**
  - Aceptación: confirmar qué backend/modos pasaron el piloto; si el modo compartido no pasa, mantener solo configuración personal.
  - Verificar: revisar el informe de Tarea 1.
  - Dependencias: Tarea 1.

- [x] **Tarea 2 — Configuración no secreta y propiedad en `.orbit`**
  - Aceptación: modelo versionable solo con campos seguros, validación y vista previa; no serializa token ni identidad personal; deshabilitada por defecto.
  - Verificar: pruebas de serialización, round trip, conflictos y secretos.
  - Dependencias: Tarea 1 para modo compartido.
  - Archivos probables: paquete `internal/atomicmemory`, CLI/manifiesto y pruebas (máximo cinco por corte).
  - Alcance: mediano.
  - Ajuste por revisión oficial de Hosted Cloud: primera versión permite apagar/uso personal; no presenta uso compartido Cloud como validado. La clave del proyecto y el `user_id` suministrado en petición no acreditan identidad o membresía de la persona.
  - [x] Subcorte 2.1 — modelo tipado, validación de backend/URL, JSON estricto y pruebas de ausencia de credenciales/identidad.
  - [x] Subcorte 2.2 — crear `.orbit/atomic-memory.json` deshabilitado, registrar su propiedad en el manifiesto y mostrarlo en la vista previa; preservar cualquier archivo preexistente.

- [ ] **Tarea 3 — Configuración MCP de Codex**
  - Aceptación: agrega el servidor oficial de forma namespaced y reversible; confirma cambios antes de aplicarlos; protege credenciales y ajustes ajenos.
  - Verificar: pruebas del plan y redacción de secretos; comprobar manualmente en una instalación Codex autenticada.
  - Dependencias: Tarea 2.
  - Archivos probables: adaptador Codex, dispatcher CLI y pruebas (máximo cinco).
  - Alcance: mediano.
  - [x] Subcorte 3.1 — comando `orbit memory configure codex`, detección segura de una entrada existente, plan con confirmación, generación de `.codex/config.toml` de proyecto sin credenciales y propiedad en el manifiesto; pruebas automatizadas pasan.
  - [ ] Subcorte 3.2 — validación manual en Codex autenticado y revisión del comportamiento de las variables locales en otra cuenta.

- [ ] **Tarea 4 — Configuración MCP de Claude Code**
  - Aceptación: agrega el servidor oficial sin reemplazar `.mcp.json` ni configuración ajena; informa aprobación de confianza pendiente.
  - Verificar: pruebas de merge/preservación; `claude mcp list`; confirmar herramientas tras confianza MCP.
  - Dependencias: Tarea 2.
  - Archivos probables: adaptador Claude Code, dispatcher CLI y pruebas (máximo cinco).
  - Alcance: mediano.

- [ ] **Checkpoint — Hosts**
  - Aceptación: `go test ./...` y `go vet ./...` pasan; Codex y Claude Code muestran estado real sin exponer secretos.
  - Dependencias: Tareas 3 y 4.

- [ ] **Tarea 5 — Protocolo de memoria del orquestador**
  - Aceptación: recuperación opcional y acotada; recuerdos son referencias, no instrucciones; ingestión solo de resúmenes o hechos aceptados; fallo MCP usa Markdown sin fingir persistencia.
  - Verificar: revisar casos de recuerdo propuesto, vigente, reemplazado, conflicto, irrelevante y servidor caído.
  - Dependencias: Tareas 2-4.
  - Archivos: skill fuente de Orbit y sus referencias.
  - Alcance: pequeño/mediano.

- [ ] **Tarea 6 — Estado, doctor y desactivación**
  - Aceptación: distingue configuración, conexión y alcance compartido verificado; quita solo entradas de Orbit no modificadas por la persona.
  - Verificar: pruebas de diagnóstico, configuración alterada, recursos ajenos y desinstalación.
  - Dependencias: Tareas 2-5.
  - Archivos: CLI, adaptadores, manifiesto y pruebas (dividir en PR si excede cinco archivos).
  - Alcance: mediano.
  - [x] **Subcorte 6.1 — `orbit memory status`:** informa modalidad del proyecto y presencia de entradas MCP en Codex/Claude; no revela variables configuradas ni afirma conectividad o memoria compartida.
  - [x] Subcorte 6.2 — `orbit doctor`: detecta los CLI disponibles, consulta listas MCP de Codex/Claude, muestra presencia de variables requeridas sin sus valores y separa evidencia reportada de memoria compartida verificada.
  - [x] Subcorte 6.3 — `orbit memory disable`: desactivación revisable que solo quite recursos propiedad de Orbit y no modificados por la persona.

- [ ] **Checkpoint final — Aceptación del piloto**
  - Aceptación: validación automática pasa; memoria personal queda aislada; modo compartido solo se declara si prueba con dos identidades pasa; secretos ausentes del diff; fallback Markdown comprobado.
  - Dependencias: Tareas 1-6.
