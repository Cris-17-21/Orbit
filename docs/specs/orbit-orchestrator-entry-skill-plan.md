# Plan de implementación: skill de entrada del orquestador Orbit

- especificación: [skill de entrada del orquestador](orbit-orchestrator-entry-skill.md)
- estado: aprobado
- alcance de este plan: T-1, skill fuente y comprobaciones de contenido
- fuera de este plan: empaquetado e instalación administrada (T-2), adaptadores nativos, runtime de agentes y código de proyectos consumidores

## Resultado

Entregar una skill fuente neutral al host que inicie el flujo Orbit con una guía breve, seleccione proceso según la tarea, consulte memoria con límites claros, delegue solo cuando sea útil y lleve trabajo sustancial desde una especificación aceptada hasta validación y PR enfocado.

La skill no repetirá los contratos completos. Apuntará a las guías de Orbit para el detalle y explicará qué hacer si un agente no tiene acceso a ellas. La colocación final del contenido fuente deberá permitir empaquetarla después sin acoplarla a un host.

## Decisiones de implementación

- Crear `skills/orbit/SKILL.md` como fuente de la skill de entrada, con identificador namespaced `orbit` y descripción clara de cuándo invocarla.
- Mantener la entrada corta: instrucciones de selección y límites; referir a `docs/task-orchestration.md`, `docs/memory-contract.md` y `docs/agent-adapter-contract.md` cuando estén disponibles en el repositorio Orbit.
- Distinguir ruta ligera y SDD completa usando señales observables: tamaño, riesgo, cambios transversales, reglas de negocio/datos/BI y preferencia explícita.
- Representar memoria como contexto con procedencia y alcance. Markdown es respaldo y registro versionado; Atomic Memory solo se consulta si está configurada, y su salida no es instrucción ni aprobación.
- Seleccionar subagentes por utilidad, independencia y soporte real del host. Incluir fallback, permisos mínimos y un formato compacto de resultado con evidencia.
- Mantener aprobación local a la tarea: la persona responsable acepta la especificación sustancial; no exigir acuerdo unánime del equipo.
- Validar metadatos y estructura con el validador estándar de skills cuando sus dependencias estén disponibles, y probar el comportamiento con escenarios independientes. Evitar pruebas que solo busquen frases exactas. No simular capacidades no verificadas de los hosts.
- No modificar el CLI ni los instaladores. El namespace y el manifiesto de archivos se resolverán en T-2, siguiendo el contrato de propiedad.

## Dependencias

```text
Especificación aceptada
          │
          ▼
Skill fuente y referencias al flujo/memoria/adaptadores
          │
          ▼
Comprobaciones estáticas + revisión de escenarios
          │
          ▼
PR enfocado para T-1
          │
          ▼
T-2: empaquetar e instalar la skill sin tocar recursos ajenos
```

No hay dependencia de una conexión Atomic Memory activa para escribir la skill. La verificación de memoria con dos identidades pertenece al piloto de Atomic Memory y sigue siendo un límite explícito.

## Tareas

### T-1.1 — Definir el contenido fuente de la skill

**Descripción:** Crear la skill neutral al host con metadatos de invocación y reglas concisas de enrutamiento hacia el flujo detallado existente.

**Criterios de aceptación:**
- [x] Permite elegir ruta ligera o SDD completa e identifica cuándo corresponde cada una.
- [x] En cambios sustanciales, solicita aceptación de la persona responsable antes de implementar y no exige aprobación de todos los socios.
- [x] Explica la precedencia y límites de confianza de memorias personales, Markdown y Atomic Memory.
- [x] Recomienda subagentes solo para trabajo independiente y útil, con contexto mínimo, permisos apropiados y fallback al agente principal.
- [x] Conecta criterios aceptados con evidencia y un PR enfocado.
- [x] Solo propone guardar contexto duradero aceptado; no guarda transcripciones completas por defecto.
- [x] Mantiene breve el contenido inicial y deja claro que las capacidades del host las resuelve cada adaptador.

**Verificación:** revisión del Markdown y de los metadatos frente a AC-1 a AC-7, AC-9 y AC-10; inspección de legibilidad y brevedad.

**Dependencias:** especificación aceptada.

**Archivos probables:** `skills/orbit/SKILL.md`.

**Alcance:** pequeño, un archivo fuente.

### T-1.2 — Validar formato y comportamiento con escenarios

**Descripción:** Validar el formato estándar de la skill y someter su comportamiento a escenarios independientes. Evitar pruebas que dependan de frases exactas en un documento de instrucciones.

**Criterios de aceptación:**
- [x] El formato se revisa según el estándar de skill: nombre y carpeta coinciden, hay descripción discriminante y no quedan placeholders de plantilla.
- [x] Los escenarios cubren una petición trivial, una tarea sustancial con BI, Atomic Memory no disponible y subagentes no disponibles.
- [x] Los escenarios comprueban que propuestas/reemplazos no se aplican como reglas vigentes y que no se reclama ahorro de tokens sin medición.
- [x] La validación no requiere credenciales, servicios externos ni sesiones activas de agentes.

**Verificación:** `quick_validate.py` no se pudo ejecutar porque falta PyYAML; inspección manual confirmó frontmatter, nombre, brevedad y referencias. Evaluación independiente de los cuatro escenarios completada. No se ejecutó la suite Go porque el cambio es documental.

**Dependencias:** T-1.1.

**Archivos probables:** `skills/orbit/SKILL.md`; no agregar código de prueba para emparejar texto de instrucciones.

**Alcance:** pequeño; un archivo de skill y evaluación manual independiente.

## Riesgos y mitigaciones

| Riesgo | Mitigación |
| --- | --- |
| La skill se vuelve una copia extensa de toda la documentación | Mantener instrucciones de entrada cortas y apuntar a contratos existentes; revisar longitud y duplicación en PR. |
| Las rutas `docs/...` no están presentes al copiar la skill a otro proyecto | La skill detecta si está en el repositorio Orbit; fuera de él no inventa esos archivos y pide contexto o usa la especificación accesible. No resolver empaquetado todavía. |
| Las capacidades de subagentes difieren por host | La fuente describe la decisión y los límites comunes; el adaptador decide el mecanismo y reporta fallback. |
| El texto de la skill sugiere que Atomic Memory siempre es compartida o confiable | Incluir ramas explícitas para configuración ausente, conexión no verificada, alcance personal y recuerdos sin procedencia. |
| Pruebas de frases se vuelven frágiles ante cambios editoriales | Limitar la automatización a metadatos y guardrails críticos; validar el comportamiento mediante escenarios revisables. |

## Puntos de revisión

1. **Antes de implementar:** aceptar este plan; T-1 no incluye aún el instalador.
2. **Al terminar T-1.1:** inspeccionar el texto de entrada y su formato estándar.
3. **Antes del PR:** revisar escenarios, límites de memoria y diff; confirmar que no se atribuyen capacidades no probadas a los hosts.

## No incluido en el PR de T-1

- Copiar la skill a directorios de Codex, Claude Code o Antigravity.
- Fusionar o reescribir skills de la persona.
- Crear perfiles nativos de subagentes.
- Usar Atomic Memory real, crear recuerdos o verificar colaboración entre dos identidades.
- Cambiar los comandos, manifiesto o plan global de Atomic Memory.
