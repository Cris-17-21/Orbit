# Especificación: skill de entrada del orquestador Orbit

- estado: aprobada
- responsable de la tarea: persona que inicia y acepta la tarea
- creada: 2026-10-23
- identificador: `orbit-orchestrator-entry-skill`
- depende de: [orquestación de tareas](../task-orchestration.md), [contrato de memoria](../memory-contract.md), [contrato de adaptadores](../agent-adapter-contract.md)

## Necesidad y resultado

Quien trabaja con un agente compatible necesita una entrada sencilla para iniciar el flujo Orbit en el proyecto actual. La skill debe orientar al agente a entender la petición, seleccionar solo el proceso y especialistas que aporten valor, aplicar SDD cuando corresponda y usar Markdown y Atomic Memory según su configuración y alcance.

El resultado es una skill fuente compartida, instalable por los adaptadores de Orbit, que funciona como guía de enrutamiento. No implementa un agente autónomo, un runtime de subagentes ni un cliente MCP propio.

## Alcance

### Incluye

- Definir la interfaz de invocación y las instrucciones de la skill de entrada.
- Enrutar cada petición por una ruta ligera o una ruta SDD completa según tamaño, riesgo, reglas de negocio/datos y preferencia explícita.
- Consultar la memoria Markdown del proyecto y, solo si Atomic Memory está habilitada, orientar la recuperación MCP acotada y registrar sus límites de confianza y alcance.
- Recomendar o delegar únicamente subtrabajos independientes y útiles, con contexto mínimo, permisos adecuados y resultado conciso con evidencia.
- Exigir aceptación de la especificación por la persona responsable de la tarea antes de implementaciones sustanciales; no exigir aprobación unánime de socios.
- Guiar la preparación de validación y un PR enfocado para la tarea aceptada.
- Proponer cambios de memoria duradera como actualizaciones revisables; no promover recuerdos a reglas vigentes automáticamente.

### Excluye

- Crear o modificar código del proyecto SaaS donde Orbit se use.
- Implementar ejecución, enrutamiento de modelos, interfaz TUI, métricas de tokens o transporte MCP.
- Crear perfiles nativos de subagentes o sus adaptadores de Codex, Claude Code y Antigravity.
- Afirmar compatibilidad uniforme de delegación o memoria entre hosts.
- Reemplazar la guía completa de orquestación o el contrato de memoria por instrucciones duplicadas dentro de la skill.

## Contexto y reglas

| Referencia | Estado | Alcance | Relevancia |
| --- | --- | --- | --- |
| `docs/task-orchestration.md` | Propuesto en el producto | Flujo de tareas Orbit | Fuente del flujo detallado que la skill resumirá y aplicará. |
| `docs/memory-contract.md` | Contrato del producto | Memoria Markdown y Atomic Memory | Markdown sigue siendo la referencia compartida versionada; Atomic Memory es opcional y auxiliar. |
| `docs/agent-adapter-contract.md` | Contrato del producto | Hosts y subagentes | Las capacidades varían por host; delegación solo si el host la admite. |
| `docs/specs/orbit-atomic-memory-foundation.md` | Aprobada | Configuración inicial de Atomic Memory | Hosted Cloud configurado con alcance personal no prueba memoria compartida entre colaboradores. |

La persona responsable de cada tarea acepta su especificación. Esta aceptación no implica aprobación de todos los socios. Reglas de negocio vigentes requieren fuente, alcance y aprobación trazable. Recuerdos de Atomic Memory, documentos `proposed`, recuerdos reemplazados o registros con procedencia insuficiente no son reglas vigentes.

## Supuestos y decisiones

### Confirmados

- La skill será contenido fuente independiente del host y se instalará con namespace Orbit sin reemplazar skills ajenas.
- Markdown es el comportamiento base. Atomic Memory solo se consulta cuando está configurada para el proyecto y el host elegido puede acceder a ella.
- Se seleccionan especialistas según necesidad, solicitud y soporte del host; no son etapas fijas.

### Por verificar durante diseño

- El nombre y disparador exactos deben funcionar en los tres hosts objetivo; el adaptador puede traducir el mismo contenido fuente a su convención.
- El límite de contexto y el formato de resumen para subagentes se ajustarán con pruebas de uso en Codex, Claude Code y Antigravity.

## Escenarios de usuario

### Escenario: iniciar un cambio pequeño

- Dado un cambio local, claro y de bajo riesgo
- Cuando la persona lo solicita a través de la skill de Orbit
- Entonces la skill resume el resultado esperado, acuerda una comprobación suficiente y evita exigir artefactos SDD o subagentes innecesarios.

### Escenario: iniciar un cambio sustancial

- Dado un cambio con varias áreas, reglas de negocio o datos/BI
- Cuando la persona solicita el trabajo
- Entonces la skill recupera contexto pertinente, identifica dudas materiales, prepara una especificación y espera que la persona responsable de la tarea la acepte antes de implementar.

### Escenario: recuperar Atomic Memory

- Dado que Atomic Memory está habilitada para el proyecto
- Cuando comienza una tarea
- Entonces la skill solicita solo contexto pertinente, conserva referencias, estado y alcance disponibles, y comunica si el alcance compartido no puede verificarse.

### Escenario: delegar trabajo

- Dado que existe un subtrabajo independiente cuyo contexto y resultado se pueden acotar
- Cuando el host ofrece subagentes adecuados
- Entonces la skill recomienda la delegación con su motivo, entrega solo el contexto necesario y solicita un resumen breve con referencias a evidencia.

## Criterios de aceptación

- [ ] **AC-1 Enrutamiento proporcional:** la guía ofrece ruta ligera y SDD completa, explica señales de selección y permite que la persona pida cualquiera de las dos.
- [ ] **AC-2 Aceptación de tarea:** en ruta SDD, los criterios de aceptación y la persona responsable son visibles; la implementación sustancial no inicia antes de aceptar la especificación. No se pide aprobación de todos los socios.
- [ ] **AC-3 Autoridad de memoria:** la skill separa memoria personal, memoria Markdown del proyecto y memoria Atomic Memory; no convierte propuestas, registros reemplazados o recuerdos sin procedencia en reglas actuales.
- [ ] **AC-4 Uso opcional de Atomic Memory:** sin configuración, el flujo continúa con Markdown; con configuración, indica el alcance consultado y no afirma conexión, aislamiento ni compartición que el host/proveedor no haya verificado.
- [ ] **AC-5 Delegación selectiva:** la skill solo delega trabajo independiente y acotado cuando el host lo soporta; define contexto mínimo, permisos y formato breve de respuesta, y contempla fallback al agente principal.
- [ ] **AC-6 Validación y PR acotados:** las tareas sustanciales conectan criterios aceptados con evidencia y guían un PR enfocado a una tarea coherente.
- [ ] **AC-7 Memoria revisable:** la skill propone persistir solo información duradera aceptada y mantiene aprobación/procedencia en la memoria Markdown; no guarda transcripciones completas por defecto.
- [ ] **AC-8 No interferencia:** instalación y actualización de la skill usan namespace propio de Orbit y nunca reemplazan skills preexistentes; conflictos se informan.
- [ ] **AC-9 Portabilidad:** el contenido compartido no depende de rutas o sintaxis exclusivas de un solo host; diferencias y limitaciones se remiten al adaptador.
- [ ] **AC-10 Economía de contexto:** las instrucciones cargadas al inicio son breves y apuntan a guías detalladas solo cuando apliquen; la skill no promete ahorro de tokens sin medición.

## Dependencias y riesgos

- El descubrimiento e invocación de MCP y subagentes depende de capacidades que cambian entre hosts y versiones; validar cada adaptador por separado.
- Una skill inicial demasiado larga consumiría contexto incluso en tareas simples; mantener la entrada concisa y enlazar guías especializadas bajo demanda.
- Un recuerdo con alcance personal podría confundirse con contexto del equipo; mostrar el alcance y recurrir a Markdown cuando no pueda comprobarse.

## Preguntas abiertas

- ¿La interfaz de entrada se llamará `/orbit`, `orbit` u otro nombre dentro de cada host? La propuesta es usar el nombre neutral `orbit` y dejar al adaptador la sintaxis de invocación.
- ¿Qué campos de versión y compatibilidad deberá declarar el paquete de skills? Propuesta: metadatos mínimos comunes y el adaptador registra la versión fuente instalada.

## Evidencia de validación

| Criterio | Comprobación propuesta | Resultado |
| --- | --- | --- |
| AC-1 a AC-7, AC-9 y AC-10 | Evaluación independiente de una tarea trivial, una métrica BI ambigua, memoria no disponible/con conflicto y falta de subagentes; revisión de idioma, contenido neutral al host y límites de memoria. | Aprobado con una aclaración menor en la ruta ligera, incorporada a `skills/orbit/SKILL.md`. |
| AC-8 | Inspección del paquete y manifiesto en instalación/actualización sobre configuraciones con skills ajenas. | Fuera de T-1; pendiente para T-2 (empaquetado e instalación). |

## Límites propuestos de tarea y PR

| Tarea | Resultado | Criterios cubiertos | Dependencias | Límite del PR |
| --- | --- | --- | --- | --- |
| T-1 | Skill fuente de entrada `orbit` y comprobación de escenarios/instrucciones | AC-1 a AC-7, AC-9, AC-10 | Aceptación de esta especificación | Un PR con la skill neutral al host y su evidencia de validación. |
| T-2 | Empaquetado de la skill mediante instalación administrada sin interferir con recursos ajenos | AC-8 | T-1; decisión de rutas y propiedad por host | Un PR posterior centrado en instalación, manifiesto y conflictos. |

## Aceptación

Esta especificación aún no autoriza la implementación.

- aceptada por: persona responsable de la tarea en Codex
- aceptada el: 2026-10-23
- notas o cambios solicitados: continuar con un plan por separado para no reemplazar el plan pendiente de Atomic Memory.
