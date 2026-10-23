# Evaluación de Atomic Memory

- estado: propuesta de investigación y piloto
- revisión: 2026-10-23

Esta nota registra qué puede asumir Orbit de manera segura sobre Atomic Memory y propone un pequeño piloto de memoria de equipo. No configura una cuenta de Atomic Memory ni un agente.

## Lo que confirma la documentación

- Atomic Memory expone un servidor MCP con las herramientas memory_search, memory_ingest, memory_package y memory_list. MCP entrega herramientas a agentes compatibles, pero no define por sí mismo el flujo de aprobación de reglas de negocio de Orbit.
- Atomic Memory distingue el alcance personal del usuario del alcance de un espacio de trabajo. La documentación de Open Source Core describe visibilidad compartida, solo para agentes y restringida. Las lecturas del espacio identifican el espacio y el agente que llama.
- El concepto de espacio de equipo del SDK usa un usuario y un namespace. El contrato HTTP de espacios de Core usa un ID de espacio y un ID de agente. Se relacionan, pero no son campos de configuración intercambiables; Orbit debe verificar el backend e interfaz elegidos antes de asignarlos.
- Core describe el espacio como un límite de colaboración dentro de la cuenta de una persona, mientras que el SDK indica que la aplicación proporciona la identidad de usuario. La documentación no establece cómo se incorporan dos colaboradores autenticados por separado al mismo espacio ni cómo se aplica esa identidad en el servidor MCP. Es una pregunta del piloto, no una capacidad que Orbit deba dar por sentada.
- La guía actual de Hosted Cloud indica que se omita ATOMICMEMORY_SCOPE_NAMESPACE hasta corregir un defecto documentado de filtrado/ingesta de alcance. No diseñar el uso compartido del equipo suponiendo que Hosted Cloud maneja namespaces correctamente.
- La lista publicada de integraciones para agentes de programación documenta Codex y Claude Code, entre otros, pero no Antigravity. Podría funcionar una conexión MCP genérica si el host la admite, pero esa combinación no está verificada.
- El resultado recuperado de Atomic Memory es contexto, no aprobación. Orbit debe aplicar sus propias reglas de estado, procedencia, alcance y responsable del [contrato de memoria](memory-contract.md).

## Piloto recomendado

Evaluar la colaboración en un espacio desechable de Open Source Core antes de elegir backend de producción o implementar una integración en el instalador. Usar datos sintéticos. Primero confirmar si dos colaboradores autenticados por separado acceden de forma segura al mismo espacio. No compartir credenciales personales ni simular acceso compartido asignando el mismo ID de usuario. Si no se puede autenticar y autorizar a cada colaborador, registrar el bloqueo y limitar el piloto local a una sola persona.

1. Conectar un host compatible para cada colaborador al mismo espacio de prueba de Core, después de entender los límites de identidad y acceso.
2. Hacer que A guarde una regla de negocio marcada proposed, con ID estable, alcance de proyecto, fuente y responsable.
3. Hacer que B la busque y comprobar que conserva suficiente identidad y fuente para tratarla como propuesta.
4. Guardar una memoria de prueba agent_only para A y confirmar que B no puede recuperarla. Guardar una memoria compartida del espacio y confirmar que ambos pueden recuperarla.
5. Reemplazar una regla de prueba. Confirmar que la anterior no aparece como vigente solo por seguir siendo localizable y verificar que Orbit conserva el enlace de reemplazo y la evidencia de aprobación.
6. Repetir la conexión con Codex y Claude Code. Evaluar Antigravity por separado; no asumir compatibilidad.

Usar registros sintéticos. Ingresar hechos resumidos o traspasos, nunca transcripciones completas por defecto. Tras documentar el resultado, eliminar el espacio de prueba y sus datos.

## Criterios de aceptación del piloto

- Ambos colaboradores recuperan un registro de prueba compartido mediante sus propias identidades de agente.
- El servicio o una capa autenticada documentada aplica la identidad y membresía de cada colaborador. Una clave API compartida junto con IDs proporcionados por el llamador no demuestra aislamiento entre colaboradores.
- El registro privado o agent_only no aparece para el otro colaborador.
- Los resultados de búsqueda se pueden asociar con una referencia estable, fuente y estado propuesto/vigente/reemplazado; si falta esa correspondencia, se registra como bloqueo.
- Actualizar una memoria no hace que Orbit la promueva en silencio ni borre la aprobación anterior.
- El host informa si el servidor MCP está configurado y accesible. Configurado pero no verificado no equivale a conectado.
- El equipo registra tiempo de configuración, credenciales requeridas, costos de servicio/modelo, ubicación de datos y pasos de limpieza.

## Decisión posterior al piloto

Elegir entre:

- mantener Markdown y Git como única memoria compartida por ahora;
- ofrecer Atomic Memory como recuperación opcional, manteniendo Markdown como autoridad revisable; o
- investigar más un espacio autogestionado si sus controles de acceso y costo operativo encajan con el equipo.

No prometer memoria compartida por proyecto en Hosted Cloud hasta corregir y verificar el defecto de namespace documentado. No prometer soporte de Antigravity hasta que su conexión MCP y alcance pasen el piloto.

## Fuentes oficiales consultadas

- [Integraciones de Atomic Memory](https://docs.atomicstrata.ai/integrations/overview)
- [Contrato de alcances de Atomic Memory](https://docs.atomicstrata.ai/platform/scope)
- [Alcances e identidad del SDK](https://docs.atomicstrata.ai/sdk/concepts/scopes-and-identity)
- [CLI de Atomic Memory y notas actuales de Hosted Cloud](https://docs.atomicstrata.ai/cli)
- [Guía de integración con Codex](https://docs.atomicstrata.ai/integrations/coding-agents/codex/local)
- [Guía de integración con Claude Code](https://docs.atomicstrata.ai/integrations/coding-agents/claude-code/local)
