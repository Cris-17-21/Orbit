# Especificación: instalar la skill de Orbit en un proyecto

- estado: aceptada; plan pendiente de revisión
- responsable de la tarea: persona que inicia y acepta la tarea
- creada: 2026-10-23
- identificador: `orbit-project-skill-install`
- depende de: [skill de entrada del orquestador](orbit-orchestrator-entry-skill.md), [propiedad de instalación](../install-ownership.md), [contrato de adaptadores](../agent-adapter-contract.md)

## Necesidad y resultado

Quien usa Orbit necesita habilitar su flujo de trabajo en el agente que ya utiliza, desde el CLI de Orbit y dentro del proyecto actual. La instalación debe ser fácil de revisar, compatible con Codex, Claude Code y Antigravity, y conservar otras skills y archivos del proyecto.

El resultado es un comando de CLI que instala la skill fuente de Orbit en el alcance del proyecto, registra los archivos que Orbit administra y explica cualquier conflicto antes de escribir.

## Alcance

### Incluye

- Instalar `skills/orbit/SKILL.md` para Codex, Claude Code y Antigravity con una opción de agente seleccionable en el CLI.
- Usar la carpeta de skill del proyecto; Codex y Antigravity comparten `.agents/skills/orbit/`, mientras que Claude Code usa `.claude/skills/orbit/`.
- Mostrar una vista previa revisable con los destinos, acciones y conflictos antes de escribir.
- Omitir recursos que colisionan con contenido no administrado por Orbit, preservarlos y reportar la instalación parcial.
- Registrar en `.orbit/manifest.json` cada recurso nuevo instalado por Orbit con su hash, para que futuras operaciones puedan comprobar su propiedad.
- Escribir los recursos no conflictivos y el manifiesto mediante la operación administrada existente, sin tocar configuraciones personales de los agentes.

### Excluye

- Instalar skills en alcance personal/global.
- Actualizar, revertir o desinstalar skills; el manifiesto de esta tarea deja una base de propiedad para esas operaciones futuras.
- Configurar Atomic Memory, MCP, subagentes, reglas de agente o instrucciones globales.
- Descargar o ejecutar código remoto durante la instalación; la fuente instalada procede del paquete local de Orbit.
- Afirmar que el host cargó o ejecutó la skill solo porque sus archivos estén presentes.
- Cambiar el contenido o comportamiento de la skill de entrada aceptada en T-1.

## Contexto y reglas de negocio

| Referencia | Estado | Alcance | Relevancia para esta tarea |
| --- | --- | --- | --- |
| `docs/install-ownership.md` | Contrato propuesto existente | Archivos administrados por Orbit | Planificar antes de escribir; no sobrescribir recursos ajenos; guardar ownership y hash en el manifiesto. |
| `docs/agent-adapter-contract.md` | Contrato de producto | Adaptadores de agentes | Cada host conserva su ruta; las capacidades no se deben dar por uniformes. |
| `docs/specs/orbit-orchestrator-entry-skill.md` | Aprobada e implementada en PR #8 | Skill fuente `orbit` | Define la única fuente a instalar; su namespace es propio de Orbit. |
| Documentación oficial de Codex | Consultada el 2026-10-23 | Skills del proyecto | Codex descubre skills de repositorio en `.agents/skills`; [documentación](https://learn.chatgpt.com/docs/build-skills). |
| Documentación oficial de Antigravity | Consultada el 2026-10-23 | Skills del proyecto | Antigravity App, CLI e IDE documentan `.agents/skills` para proyectos; [documentación](https://antigravity.google/docs/skills). |
| Documentación oficial de Claude | Consultada el 2026-10-23 | Claude Code | Claude Code descubre skills de proyecto en `.claude/skills`; [documentación](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview). |

Orbit administra solamente los destinos de la skill que registre. No reclama la carpeta completa, no fusiona skills con nombres iguales y no reemplaza recursos existentes sin propiedad verificable. La vista previa representa el plan que se va a aplicar; los cambios en destinos o manifiesto después de la vista previa deben detener la escritura y solicitar una nueva revisión.

## Supuestos y decisiones

### Confirmados por la persona responsable

- La primera instalación usa alcance por proyecto para que el equipo pueda revisar y compartir el flujo mediante Git.
- La instalación no debe interferir con las skills que ya tenga cada persona.
- La skill se distribuye desde el repositorio y CLI de Orbit; no se instala como dependencia global del usuario.
- Codex, Claude Code y Antigravity son objetivos iniciales del adaptador.

### Propuestas para confirmar en esta especificación

- La interfaz sugerida es `orbit skill install`; en modo interactivo permite elegir agentes y confirmar el plan. En modo no interactivo requiere `--yes`; `--dry-run` muestra el plan sin escribir.
- Un destino ya existente que no esté registrado como recurso administrado por Orbit se marca como conflicto. Se omite ese destino y pueden instalarse los demás destinos no conflictivos.
- Si Codex y Antigravity se seleccionan juntos, su ruta compartida se instala una sola vez y se informa que ambos la descubren desde el mismo archivo.
- Un recurso administrado por Orbit solo puede considerarse actualizable cuando su contenido actual coincide con el hash registrado. Actualizarlo queda fuera de esta versión.
- El CLI informa que los archivos quedaron instalados y sus rutas; no intenta detectar carga o invocación efectiva del host.

## Escenarios de usuario

### Instalar en Codex y Antigravity

- Dado un proyecto Git sin skill Orbit instalada
- Cuando la persona selecciona Codex y Antigravity y acepta el plan
- Entonces Orbit copia la skill una sola vez en `.agents/skills/orbit/SKILL.md` y registra ese archivo en el manifiesto.

### Instalar en Claude Code

- Dado un proyecto Git sin skill Orbit instalada
- Cuando la persona selecciona Claude Code y acepta el plan
- Entonces Orbit copia la skill en `.claude/skills/orbit/SKILL.md` y registra ese archivo en el manifiesto.

### Preservar una skill preexistente

- Dado que uno de los destinos contiene una skill Orbit no registrada por Orbit o cuyo contenido difiere del hash registrado
- Cuando la persona revisa y aplica el plan
- Entonces Orbit conserva ese destino sin cambios, reporta el conflicto y puede aplicar los destinos independientes que no colisionan.

### Vista previa sin escrituras

- Dado un proyecto Git válido
- Cuando la persona solicita `--dry-run`
- Entonces Orbit muestra agentes, destinos, operaciones y conflictos, y no crea, modifica ni elimina ningún archivo.

## Criterios de aceptación

- [ ] **AC-1 — Selección de agentes:** el CLI permite seleccionar Codex, Claude Code, Antigravity o una combinación; las opciones de proyecto son explícitas y un agente no seleccionado no recibe archivos.
- [ ] **AC-2 — Rutas correctas:** Codex y Antigravity usan `.agents/skills/orbit/SKILL.md`; Claude Code usa `.claude/skills/orbit/SKILL.md`. Codex más Antigravity no producen escrituras duplicadas en la ruta compartida.
- [ ] **AC-3 — Vista previa:** el modo interactivo presenta el plan y pide confirmación antes de escribir; `--dry-run` muestra el mismo plan sin escrituras; la ejecución no interactiva exige `--yes`.
- [ ] **AC-4 — No interferencia:** destinos existentes sin propiedad de Orbit o que difieren del hash registrado quedan byte por byte intactos y se reportan como conflicto. Orbit puede continuar con los demás destinos independientes.
- [ ] **AC-5 — Propiedad registrada:** cada archivo creado se incorpora al manifiesto `.orbit/manifest.json` con ID estable, ruta, fuente y digest SHA-256. Un manifiesto ausente, inválido o de esquema incompatible impide escribir.
- [ ] **AC-6 — Aplicación segura:** se validan raíz del proyecto, rutas relativas, ausencia de escape del repositorio y colisiones entre destinos antes de escribir. Un cambio en los recursos o manifiesto después de la vista previa cancela la aplicación.
- [ ] **AC-7 — Fuente única:** el archivo instalado es una copia byte por byte de `skills/orbit/SKILL.md` de la distribución local de Orbit; el instalador no descarga ni ejecuta archivos remotos.
- [ ] **AC-8 — Resultado claro:** el resumen en español separa instalados, ya presentes, omitidos y conflictos, lista sus rutas y no declara que el agente haya cargado la skill.
- [ ] **AC-9 — Sin alcance global:** no se escriben rutas de configuración personal/global de Codex, Claude Code ni Antigravity.
- [ ] **AC-10 — Integración segura con `orbit init`:** instalar la skill en un proyecto sin memoria previa crea o extiende el manifiesto de Orbit sin modificar ni duplicar los archivos de memoria ya existentes.

## Dependencias y riesgos

- Las rutas y capacidades de los agentes pueden cambiar; verificar versiones y documentación oficial al implementar, manteniendo las rutas como datos del adaptador y cubriéndolas con pruebas.
- Codex y Antigravity comparten una ruta de proyecto: seleccionar ambos debe producir una sola instalación administrada.
- La distribución desde binario requiere que el archivo fuente de la skill viaje dentro del artefacto local de Orbit; el mecanismo de empaquetado se define en el plan de implementación.
- Un manifiesto que Orbit cree en versiones previas podría no tener recursos para skills; migrarlo o extenderlo no debe perder la propiedad de memoria ya registrada.
- La presencia del archivo demuestra instalación en disco, no que una versión específica del host lo haya cargado o activado.

## Preguntas abiertas

- ¿La interfaz interactiva debe preseleccionar el agente detectado cuando haya exactamente uno instalado, o siempre presentar selección vacía? Resolver en el plan del CLI sin cambiar el criterio de que la persona ve el destino antes de aplicar.
- ¿El alcance de proyecto requiere detectar la raíz Git como lo hace `orbit init`, o permitir también directorios no Git con confirmación explícita? Propuesta inicial: requerir raíz Git para que el manifiesto y la skill queden versionados en un límite claro.
- ¿Qué estrategia de empaquetado del archivo fuente se ajusta al proceso real de distribución de Orbit? Resolver en el plan sin añadir descargas remotas al instalador.

## Evidencia de validación

| Criterio | Comprobación propuesta | Resultado |
| --- | --- | --- |
| AC-1, AC-2, AC-8 y AC-9 | Pruebas de planes por agente, combinaciones, rutas y resumen; revisión en Codex, Claude Code y Antigravity disponibles. | Pendiente de implementación. |
| AC-3 a AC-7 | Pruebas en repositorios temporales con destinos ausentes, propios, modificados, manifiestos inválidos y cambios entre vista previa y aplicación; verificar que los bytes ajenos no cambian. | Pendiente de implementación. |
| AC-10 | Ejecutar `orbit init` y después instalar la skill; comprobar que una segunda inicialización reconoce los archivos y hashes originales. | Pendiente de implementación. |

## Límite propuesto de tarea y PR

| Tarea | Resultado | Criterios cubiertos | Dependencias | Límite del PR |
| --- | --- | --- | --- | --- |
| T-2.1 | Plan de instalación, selección de agentes y destinos proyecto | AC-1, AC-2, AC-8, AC-9 | Especificación aceptada | Incluir solo interfaz de CLI y planificación/adaptadores necesarios. |
| T-2.2 | Aplicación administrada, manifiesto, conflictos y validación | AC-3 a AC-7, AC-10 | T-2.1 y mecanismo local de empaquetado acordado | Un PR enfocado en instalación segura y sus pruebas. |

## Aceptación

La persona responsable acepta el alcance de esta especificación. La implementación sigue pendiente de aceptar el plan técnico y las tareas.

- aceptada por: Cris (responsable de Orbit)
- aceptada el: 2026-10-23
- notas o cambios solicitados: se acepta el alcance; revisar en el plan la interfaz concreta del CLI y el requisito de raíz Git antes de implementar.
