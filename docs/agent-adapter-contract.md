# Contrato de adaptadores de agentes de Orbit

- estado: propuesta de diseño
- actualizado: 2026-10-23

Orbit mantiene un solo flujo y una biblioteca fuente de skills. Cada adaptador traduce ese contrato a los archivos y capacidades de ejecución que admite un agente. No debe insinuar que todos los agentes se comportan igual.

## Responsabilidades compartidas

Cada adaptador ofrece al CLI de Go estas operaciones:

1. **Detectar:** identificar si el host está instalado, su versión, raíces de configuración activas y alcances personales o de proyecto disponibles.
2. **Describir capacidades:** informar soporte y evidencia de instrucciones, skills, perfiles de subagentes, delegación en ejecución, MCP, hooks de ciclo de vida y métricas de uso.
3. **Planificar:** devolver los archivos y entradas de configuración exactos que pretende crear, actualizar, omitir o quitar.
4. **Aplicar:** escribir solo recursos aprobados por el instalador de Orbit y registrados en el manifiesto de propiedad.
5. **Verificar:** informar si el host encuentra el flujo, las skills, perfiles de subagentes y conexión opcional con Atomic Memory instalados.
6. **Quitar:** retirar o restaurar solo recursos propios del manifiesto, según las [reglas de propiedad de instalación](install-ownership.md).

Valores de capacidad:

- **native:** el host ofrece la capacidad y el adaptador puede configurarla.
- **manual:** el host la admite, pero la persona debe completar una acción en la interfaz, autenticación o confianza.
- **fallback:** Orbit continúa con comportamiento reducido en el agente principal sin fingir que existe capacidad nativa.
- **unavailable:** la versión del host/runtime no la ofrece o Orbit no puede verificarla.

El adaptador informa versión del host y evidencia para cada afirmación native. Ante una versión desconocida, usa capacidades conservadoras y muestra un aviso de compatibilidad.

## Contrato compartido de skills

- Mantener la fuente canónica en un namespace propio de Orbit.
- Usar el formato de carpeta Agent Skills: un archivo SKILL.md por skill, metadatos concisos de nombre y descripción, y recursos opcionales.
- Mantener las skills enfocadas. Cargar instrucciones detalladas solo al seleccionarlas o invocarlas.
- Anteponer orbit- a nombres de skills y comandos slash generados cuando las reglas del host lo permitan.
- Preservar nombres y directorios de skills de la persona. Resolver colisiones con el flujo seguro de conflictos de Orbit.
- No insertar instrucciones extensas de flujo en todos los prompts globales del host. Instalar una instrucción breve que remita a la skill del orquestador.
- Mantener plantillas y artefactos del proyecto en el repositorio; no copiar contexto arbitrario del proyecto a configuración global.

Cada adaptador determina la ruta y metadatos específicos según la documentación actual del host.

## Contrato compartido de subagentes

Un perfil neutral al host contiene:

- ID estable del rol y nombre corto;
- descripción de delegación y condiciones para activarlo;
- instrucciones enfocadas y formato de salida esperado;
- referencias de contexto requeridas;
- herramientas y permisos mínimos, marcados como lectura o escritura;
- política de concurrencia sugerida, sin fijar modelo o proveedor por defecto.

El adaptador traduce el perfil a archivos nativos de subagentes o metadatos de plugin cuando existan. El orquestador delega solo trabajo independiente y acotado, y entrega únicamente la especificación, memoria, archivos y skills pertinentes. Pide conclusiones breves con referencias de evidencia y preguntas abiertas.

Si el host/runtime activo no ofrece subagentes nativos, fallback significa que el agente principal hace el trabajo o pide a la persona iniciar otra sesión. Orbit no crea prompts anidados y los llama subagentes, ni afirma ahorro de tokens sin evidencia. Si el host expone uso, registra por separado el consumo principal y delegado; de lo contrario indica que no se puede medir el ahorro de esa ejecución.

## Matriz de capacidades

Esta matriz recoge el diseño actual. Las rutas exactas y compatibilidad por versión son detalles de implementación y deben revisarse en fuentes oficiales cuando se construya cada adaptador.

| Capacidad | Codex | Claude Code | Antigravity |
| --- | --- | --- | --- |
| Instrucciones de proyecto | Reconoce AGENTS.md; mantener breve y con namespace de Orbit | CLAUDE.md y skills; mantener breve la guía siempre cargada | Reglas y skills de espacio; preferir alcance del proyecto |
| Skills compartidas | Admite skills SKILL.md; empaquetar o instalar según el alcance documentado | SKILL.md según sus convenciones; el contenido se carga al invocarse | Admite Agent Skills, skills de espacio y plugins |
| Perfiles nativos de subagentes | Depende del runtime; verificar la superficie Codex instalada. La API de OpenAI Agents documenta delegación multiagente en su runtime administrado | Subagentes personalizados con contextos, herramientas y permisos separados, y skills precargadas opcionales | El CLI documenta subagentes asíncronos; los plugins pueden incluir perfiles |
| Alternativa de delegación | Usar agente principal si el runtime Codex activo no delega | Usar agente principal si delegar no aporta | Usar agente principal si la superficie Antigravity no ofrece el control necesario |
| MCP | Disponible; configurar mediante ajustes/CLI admitidos por Codex | Disponible en alcances local, proyecto y usuario | Disponible; CLI/IDE permite configuración global o del espacio |
| Atomic Memory | Documenta integración MCP con Codex | Documenta integración con Claude Code | Admite MCP, pero la lista actual de Atomic Memory no incluye adaptador específico; iniciar con MCP genérico y validar en piloto |
| Visibilidad de uso/tokens | Depende del host y versión; informar solo valores medidos | Depende del proveedor/runtime; subagentes pueden contar para el mismo límite | Depende del host y versión; informar solo valores medidos |

## Comportamiento por host

### Codex

- Instalar la skill compartida del orquestador y una instrucción breve dentro del alcance de usuario o proyecto seleccionado.
- Agregar Atomic Memory MCP solo si la persona optó por usarlo y las credenciales están disponibles mediante un mecanismo seguro admitido.
- Tratar subagentes como una capacidad del runtime, no como una suposición basada en el nombre del agente. La API administrada Codex/Agents documenta orquestación, pero no demuestra que todos los clientes o versiones Codex la admitan.
- Verificar skills, MCP y delegación disponible en la superficie instalada, e informar soporte parcial claramente.

### Claude Code

- Instalar skills con namespace e instrucciones de proyecto según las convenciones de alcance del host.
- Instalar perfiles de subagentes con permisos mínimos y descripciones concisas; las instrucciones completas se cargan al ejecutarse.
- Configurar Atomic Memory MCP en el alcance de proyecto cuando el uso compartido corresponda al proyecto. Respetar el flujo de confianza y aprobación de Claude Code e indicar si la persona debe aprobar o autenticar el servidor.
- No copiar credenciales a ajustes versionados ni archivos de skills.

### Antigravity

- Preferir el paquete de plugin documentado cuando ofrezca un contenedor estable con namespace para skills, perfiles, reglas, MCP y hooks.
- Para skills y MCP locales al proyecto, usar las ubicaciones de espacio documentadas por la versión activa de Antigravity.
- Atomic Memory no documenta actualmente un paquete específico para agentes de programación Antigravity. Configurar su endpoint como MCP genérico solo tras validar el esquema, identidad de alcance y credenciales. doctor debe etiquetar esa ruta como MCP genérico, no como adaptador oficial de Atomic Memory.
- El host podría requerir aprobación de confianza en la interfaz. Informarlo como manual y verificar la conexión después.

## Comportamiento común para Atomic Memory

Cuando se habilite, el orquestador debe mantener un comportamiento de memoria consistente entre hosts:

- Buscar el alcance configurado del proyecto/espacio al comenzar una tarea.
- Incluir solo recuerdos pertinentes y conservar fuente, alcance, estado y procedencia.
- Usar identidad estable para la persona/equipo y una identidad de cliente diferenciada por adaptador/agente cuando sea necesario.
- Guardar conocimiento duradero aprobado solo después de proponer una actualización revisable; mantener separada la memoria personal.
- Usar las herramientas MCP de Atomic Memory mediante el adaptador del host. La skill de flujo determina cuándo y cómo invocarlas; el servidor gestiona almacenamiento y alcance.
- Si Atomic Memory está configurado pero no disponible, continuar con Markdown del proyecto, avisar que no se pudo recuperar memoria compartida y no afirmar que se guardó.

El adaptador debe verificar los alcances y requisitos vigentes de Atomic Memory al implementarse; la identidad del proveedor y el comportamiento de conexión del host pueden cambiar.

## Verificación de instalación

orbit doctor informa, para cada host elegido:

- nombre y versión detectados;
- alcance elegido y destinos administrados;
- estado de descubrimiento de instrucciones y skills del orquestador;
- estado de perfiles de subagentes y delegación por separado;
- servidor MCP configurado, pendiente de aprobación/autenticación, conectado o no disponible;
- modalidad de Atomic Memory: integración del proveedor, MCP genérico, deshabilitada o desconocida;
- comportamientos incompatibles, manuales o alternativos.

doctor no debe marcar un servidor MCP como conectado solo porque exista una entrada de configuración. Si el host no ofrece estado de conexión, informa configurado, conexión sin verificar.

## Referencias oficiales consultadas

- [Skills de Codex y OpenAI](https://developers.openai.com/plugins/concepts/skills)
- [API de Codex/Agents: varios agentes](https://developers.openai.com/api/docs/guides/agents-api/multi-agent)
- [Skills de Claude Code](https://code.claude.com/docs/en/skills)
- [Subagentes de Claude Code](https://code.claude.com/docs/en/sub-agents)
- [MCP de Claude Code](https://code.claude.com/docs/en/mcp)
- [Plugins y skills de Antigravity](https://www.antigravity.google/docs/cli/plugins/)
- [Subagentes del CLI de Antigravity](https://www.antigravity.google/docs/cli/features/)
- [MCP de Antigravity](https://www.antigravity.google/docs/mcp)
- [Integraciones de Atomic Memory](https://docs.atomicstrata.ai/integrations/overview)
- [Alcances de Atomic Memory](https://docs.atomicstrata.ai/platform/scope)

Estas fuentes describen productos y versiones distintos. Volver a comprobar soporte, rutas de configuración y autenticación al implementar o actualizar un adaptador.

## Decisiones de implementación abiertas

- Versión mínima admitida de cada agente y cómo detectarla.
- Si usar paquetes Codex, skills locales al proyecto o ambos para cada alcance.
- Conversión de perfiles de roles y permisos a cada sistema de subagentes.
- Modalidades de Atomic Memory admitidas inicialmente en cada host.
- Métricas disponibles y cómo registrar uso de tokens sin almacenar prompts sensibles.
- Pruebas de adaptadores en CI sin requerir sesiones autenticadas de agentes.
