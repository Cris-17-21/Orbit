# Evaluación de Atomic Memory

- estado: propuesta de investigación y piloto
- revisión: 2026-10-23

Esta nota registra qué puede asumir Orbit de manera segura sobre Atomic Memory y propone un pequeño piloto de memoria de equipo. No configura una cuenta de Atomic Memory ni un agente.

## Lo que confirma la documentación

- Atomic Memory expone un servidor MCP con las herramientas memory_search, memory_ingest, memory_package y memory_list. MCP entrega herramientas a agentes compatibles, pero no define por sí mismo el flujo de aprobación de reglas de negocio de Orbit.
- `memory_package` acepta un presupuesto de tokens para limitar el contexto recuperado. Orbit establece como valor máximo de inicio 800 tokens y reserva el paquete para consultas amplias; para una pregunta concreta prefiere una sola búsqueda con límite de cinco resultados.
- `memory_ingest` admite modos de extracción (`text` y `messages`) y registro determinista (`verbatim`). La documentación actual indica que Core 1.2.0+ con política predeterminada rechaza ingestas `verbatim` sin `contentClass: summary` o `redacted`; el linaje se envía en `provenance` (`source`, `sourceUrl`, `sourceId`), no como metadatos de integración.
- Atomic Memory distingue el alcance personal del usuario del alcance de un espacio de trabajo. La documentación de Open Source Core describe visibilidad compartida, solo para agentes y restringida. Las lecturas del espacio identifican el espacio y el agente que llama.
- El concepto de espacio de equipo del SDK usa un usuario y un namespace. El contrato HTTP de espacios de Core usa un ID de espacio y un ID de agente. Se relacionan, pero no son campos de configuración intercambiables; Orbit debe verificar el backend e interfaz elegidos antes de asignarlos.
- Core describe el espacio como un límite de colaboración dentro de la cuenta de una persona, mientras que el SDK indica que la aplicación proporciona la identidad de usuario. La documentación no establece cómo se incorporan dos colaboradores autenticados por separado al mismo espacio ni cómo se aplica esa identidad en el servidor MCP. Es una pregunta del piloto, no una capacidad que Orbit deba dar por sentada.
- La guía actual de Hosted Cloud indica que se omita ATOMICMEMORY_SCOPE_NAMESPACE hasta corregir un defecto documentado de filtrado/ingesta de alcance. No diseñar el uso compartido del equipo suponiendo que Hosted Cloud maneja namespaces correctamente.
- La lista publicada de integraciones para agentes de programación documenta Codex y Claude Code, entre otros, pero no Antigravity. Podría funcionar una conexión MCP genérica si el host la admite, pero esa combinación no está verificada.
- El resultado recuperado de Atomic Memory es contexto, no aprobación. Orbit debe aplicar sus propias reglas de estado, procedencia, alcance y responsable del [contrato de memoria](memory-contract.md).

### Hallazgos de Hosted Cloud revisados el 2026-10-23

- La consola organiza proyectos dentro de una organización; las memorias, claves, trazas y uso quedan asociados al proyecto Cloud.
- Una clave `amc_…` da acceso al proyecto y se configura aparte de la sesión web de la consola. El endpoint de MCP documentado es `https://api.atomicstrata.ai`.
- Las llamadas MCP usan las mismas rutas de memoria que la API. En los ejemplos, `user_id` viaja en la petición junto con la clave de proyecto. La documentación consultada no describe autenticación humana por petición ni control de membresía por colaborador para estas llamadas.
- Por eso, una clave de proyecto más un `user_id` local permite identificar particiones, pero no demuestra autorización individual. Orbit no habilitará modo compartido basándose solo en esos valores.
- La documentación del CLI `am` advierte defectos actuales de namespace/ingesta y de derivación de scope user; Orbit no usará `am integrate` como generador hasta que esas advertencias se resuelvan y se vuelvan a verificar.
- La ruta que recomienda hoy Atomic Memory para Hosted Cloud es MCP manual, con `ATOMICMEMORY_SCOPE_USER` explícito por persona, `ATOMICMEMORY_SCOPE_LOCK=true` y sin `ATOMICMEMORY_SCOPE_NAMESPACE` por la asimetría documentada entre ingesta y búsqueda.
- Claude Code permite expansión de variables en `.mcp.json`, aunque ciertas variables con nombres de credenciales se vacían intencionalmente. Validar el nombre de variable elegido y el flujo real antes de prometer configuración automática.

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

## Registro de ejecución

Completar una copia de este registro por cada backend y combinación de agentes. No incluir claves, tokens, correos ni identificadores personales; usar etiquetas como `A` y `B` y referencias redactadas a la evidencia.

```text
Estado: pendiente | aprobado | bloqueado
Fecha:
Backend, plan y versión:
Hosts/agentes y versiones:
Identidades A y B autenticadas por separado: sí | no | no comprobado
Membresía del mismo espacio y mecanismo documentado:
Alcance y visibilidad observados por cada identidad:

Caso: A guarda recuerdo personal; B no lo recupera
Resultado: pasa | falla | no ejecutado
Evidencia redactada:

Caso: A guarda recuerdo compartido; B lo recupera con referencia y procedencia
Resultado: pasa | falla | no ejecutado
Evidencia redactada:

Caso: una propuesta no se trata como regla vigente sin aprobación
Resultado: pasa | falla | no ejecutado
Evidencia redactada:

Caso: reemplazo conserva historial y enlace a la aprobación
Resultado: pasa | falla | no ejecutado
Evidencia redactada:

Secretos ausentes de Git, salida y evidencia: sí | no
Tiempo de configuración y costos observados:
Ubicación/retención de datos conocida: sí | no; detalle:
Datos sintéticos y espacio de prueba eliminados: sí | no | pendiente
Limitaciones o bloqueos:
Conclusión: Markdown solamente | Atomic Memory personal | Atomic Memory compartida verificada
Responsable y referencia de revisión:
```

La conclusión `Atomic Memory compartida verificada` requiere que pasen los criterios de identidad, aislamiento y recuperación compartida. Si algún caso no se ejecutó, la modalidad correspondiente queda pendiente; no se infiere a partir de una configuración MCP presente.

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
- [Consola Cloud](https://docs.atomicstrata.ai/cloud/console)
- [Proyectos Cloud](https://docs.atomicstrata.ai/cloud/how-to/projects)
- [Autenticación Cloud](https://docs.atomicstrata.ai/cloud/authentication)
- [Claves API Cloud](https://docs.atomicstrata.ai/cloud/how-to/api-keys)
- [Configuración MCP de Claude Code](https://code.claude.com/docs/en/mcp)
- [CLI Atomic Memory: MCP manual y defectos de `am integrate`](https://docs.atomicstrata.ai/cli)
