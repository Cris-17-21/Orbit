# Especificación del producto Orbit

- estado: borrador para revisión
- producto: Orbit
- actualizado: 2026-10-23
- repositorio: https://github.com/Cris-17-21/Orbit

Este documento consolida la dirección del producto y las decisiones de diseño aceptadas. No es la especificación de implementación de una función. Las decisiones abiertas deben resolverse antes de construir las capacidades afectadas.

## Propósito

Orbit es un conjunto de herramientas instalable para equipos que desarrollan software con agentes de programación con IA. Ayuda a convertir solicitudes en especificaciones aceptadas, tareas pequeñas, pull requests acotados y validaciones con evidencia. También permite compartir contexto útil entre colaboradores, agentes y sesiones.

Orbit se distribuye desde GitHub y se configura mediante un CLI en Go con una interfaz guiada para terminal. Funciona con agentes que la persona ya usa; no proporciona ni instala los modelos o productos de agentes.

## Usuarios y necesidades

### Usuarios principales

- Desarrolladores y responsables de producto que trabajan con agentes de programación con IA.
- Equipos pequeños que necesitan compartir reglas, decisiones y contexto de tareas.
- Equipos que construyen SaaS sustanciales, incluso con funciones de BI y datos.

### Necesidades

- Convertir solicitudes poco claras en tareas acotadas con criterios verificables.
- Hacer explícitas y revisables las reglas de negocio.
- Permitir que la persona responsable de la tarea acepte su especificación sin requerir aprobación de todos los socios en cada paso.
- Dividir trabajo en PR pequeños que puedan revisarse e integrarse por separado cuando sus dependencias lo permitan.
- Reutilizar contexto del proyecto entre agentes y sesiones sin tratar notas antiguas como reglas vigentes.
- Usar subagentes especializados cuando aporten valor y controlar el contexto y costo de tokens.
- Instalar, actualizar, inspeccionar y quitar Orbit sin alterar la configuración existente de los agentes ni las skills instaladas por la persona.

## Decisiones de producto aceptadas

- El nombre del proyecto es Orbit.
- El CLI se escribe en Go y usa Bubble Tea para su interfaz guiada de terminal.
- Las primeras integraciones previstas son Codex, Claude Code y Antigravity.
- El flujo y las skills fuente son compartidos; cada agente tiene un adaptador para sus formatos y capacidades.
- Los roles de subagentes se seleccionan según la tarea o cuando alguien los pide; no son etapas obligatorias.
- Delegar trabajo independiente y acotado con solo el contexto necesario y resultados concisos con evidencia. Medir el ahorro de tokens en vez de asumir que delegar siempre cuesta menos.
- Markdown del proyecto es la memoria predeterminada y versionada. Atomic Memory es opcional. Un modo híbrido se posterga hasta diseñar sincronización y resolución de conflictos.
- Los registros de memoria separan reglas de negocio, decisiones técnicas y contexto del proyecto. Incluyen tipo, contenido, alcance, estado, fecha, fuente y responsable. Los registros vigentes identifican quién los aprobó; los reemplazados enlazan al nuevo cuando existe.
- La instalación es interactiva, clara y navegable con teclado; también debe permitir opciones no interactivas.
- Orbit administra solo sus propios archivos registrados. Debe preservar skills y configuración ajenas durante instalación, actualización y desinstalación.
- Orbit permite elegir versión, buscar actualizaciones, actualizar y volver a una versión previa.
- El idioma principal de la interfaz y la documentación del proyecto es el español. Se conservan en inglés los nombres propios, identificadores técnicos y términos de herramientas cuando sean la convención establecida.

## Experiencia de producto

### Instalar Orbit

La persona obtiene Orbit desde GitHub o una versión publicada y ejecuta el CLI. La configuración guiada:

1. Permite elegir instalación para el usuario o para un proyecto.
2. Detecta o permite seleccionar Codex, Claude Code y/o Antigravity.
3. Muestra los archivos y las áreas de configuración que administrará Orbit.
4. Ofrece memoria Markdown del proyecto y consulta si se desea configurar Atomic Memory.
5. Instala la versión seleccionada, informa conflictos sin sobrescribir archivos en silencio y muestra cómo iniciar el orquestador.

El flujo debe tener una alternativa no interactiva para scripts y configuración repetible del equipo.

### Iniciar una tarea

La persona invoca el orquestador desde su agente. Orbit identifica el repositorio y la solicitud, selecciona una ruta SDD ligera o completa, recupera contexto pertinente, explora el código cuando corresponda y redacta una especificación para cambios sustanciales. La persona responsable de la tarea la revisa y acepta. Después Orbit divide el alcance aceptado en tareas pequeñas, delega trabajo independiente útil, apoya la implementación y validación, y prepara PR enfocados.

El comportamiento detallado está descrito en [orquestación de tareas](task-orchestration.md).

### Compartir memoria del proyecto

La memoria Markdown vive en el repositorio y se revisa mediante Git. Atomic Memory, si está configurado, permite recuperar y guardar contexto con alcance entre sesiones. La memoria informa el trabajo, pero no aprueba por sí sola una regla de negocio ni reemplaza un artefacto SDD aceptado.

Las reglas iniciales se describen en el [contrato de memoria](memory-contract.md).

### Mantener Orbit

El CLI muestra la versión instalada y los archivos administrados, busca versiones disponibles, actualiza a una versión elegida y restaura una versión anterior. Incluye orbit doctor para inspeccionar la instalación e informar problemas de configuración con pasos concretos.

## Flujo compartido

1. Entender la solicitud e identificar el proyecto.
2. Elegir trabajo ligero o SDD completo según alcance, riesgo y preferencia.
3. Recuperar memoria Markdown y, si está habilitada, memoria de Atomic Memory.
4. Explorar el comportamiento y las convenciones existentes.
5. Seleccionar solo roles especialistas útiles y subtrabajos acotados.
6. Redactar especificación, reglas de negocio, supuestos, preguntas y criterios verificables.
7. Pedir a la persona responsable que acepte o revise la especificación.
8. Dividir el alcance aceptado en tareas pequeñas con dependencias y evidencia.
9. Implementar, validar y preparar PR pequeños.
10. Proponer actualizaciones duraderas de memoria.

El flujo se adapta a cambios pequeños; no todas las tareas requieren todas las fases ni un subagente.

## Áreas iniciales de capacidad

### CLI

Comandos propuestos:

| Comando | Propósito |
| --- | --- |
| orbit | Abrir la configuración guiada o el menú principal |
| orbit install | Instalar una versión elegida de Orbit y las integraciones seleccionadas |
| orbit init | Inicializar el flujo de trabajo y la estructura de memoria Markdown del proyecto |
| orbit doctor | Diagnosticar archivos administrados e integraciones configuradas |
| orbit update | Actualizar a una versión elegida o compatible más reciente |
| orbit update --check | Mostrar actualizaciones disponibles sin cambiar archivos |
| orbit rollback | Restaurar una versión instalada previamente |
| orbit uninstall | Quitar solo archivos y configuración administrados por Orbit |

Los nombres y la separación entre install e init siguen abiertos hasta diseñar el CLI.

### Orquestador y skills

Orbit incluirá una skill de entrada pequeña para iniciar el flujo y skills enfocadas en actividades recurrentes: exploración, requisitos, especificación SDD, planificación, datos/BI, implementación, pruebas, UI/UX, seguridad y revisión. El catálogo exacto sigue abierto y debe surgir del flujo acordado y de casos de uso reales.

Cuando sea posible, las skills se guardan una sola vez como contenido fuente. Los adaptadores las instalan o empaquetan en la ubicación que espera cada agente. Las skills existentes de la persona permanecen disponibles e intactas. Los conflictos de nombres se informan y resuelven sin reemplazos silenciosos.

### Catálogo de roles de subagentes

Roles posibles: exploración del repositorio, requisitos y reglas de negocio, diseño técnico, UX, UI, frontend, backend/API, datos/BI, seguridad, implementación, pruebas, documentación, rendimiento y revisión de cambios. Este es un catálogo de capacidades, no un grupo fijo que siempre se ejecuta. Qué roles admite cada host como subagentes nativos y su correspondencia entre agentes sigue abierto.

### Adaptadores de agentes

Cada adaptador documenta y administra solo los archivos propios del host que se necesitan para exponer instrucciones, skills, perfiles de subagentes y conexión opcional de Atomic Memory MCP. Las diferencias deben ser visibles en doctor y en los resúmenes de instalación. Orbit no debe afirmar que la delegación funciona igual en todos los hosts si sus capacidades difieren.

Los detalles de compatibilidad deben verificarse con documentación oficial vigente al implementar cada adaptador. Las responsabilidades compartidas están descritas en el [contrato de adaptadores](agent-adapter-contract.md).

### Memoria del proyecto

Markdown es la alternativa base que no requiere servicios externos. Atomic Memory se configura de forma opcional. La persona selecciona o proporciona la conexión de proyecto o espacio correspondiente. Orbit distingue alcances personales y compartidos, conserva procedencia y no promueve en silencio registros propuestos o antiguos. La autenticación, identidad y modalidades local y alojada siguen abiertas.

Los hallazgos actuales y el piloto propuesto están en la [evaluación de Atomic Memory](atomic-memory-evaluation.md). No constituyen un compromiso de incluir esa integración.

### Ejemplo SaaS y BI

El ejemplo muestra categorías de memoria, una regla de negocio propuesta, una decisión técnica propuesta, un borrador de métrica de conversión de invitaciones, un desglose de tareas y una plantilla para PR enfocados. Es ficticio y no está aceptado hasta que una persona responsable lo revise.

## Aislamiento y propiedad de la instalación

- Preferir instalación dentro del proyecto cuando el equipo quiera compartir el flujo mediante Git.
- Ofrecer instalación para el usuario cuando se quiera usar personalmente en varios proyectos.
- Registrar cada archivo y fragmento de configuración administrado, incluida la versión de Orbit.
- Ubicar los recursos en un espacio propio de Orbit cuando el host lo permita.
- Combinar solo entradas identificadas de Orbit y preservar las demás entradas y, cuando sea práctico, el formato.
- Si un archivo ajeno se sobrescribiría o una skill entra en conflicto, detener esa escritura y explicar una solución segura.
- Al desinstalar, quitar solo recursos propios. Preservar cambios de la persona y archivos ajenos; informar limpieza manual pendiente.
- Las actualizaciones muestran la versión objetivo y los cambios relevantes, crean un punto de recuperación y permiten volver atrás.

El formato del manifiesto y el comportamiento ante cambios hechos por la persona siguen abiertos. Las reglas propuestas están en [propiedad y conflictos de instalación](install-ownership.md).

## Atributos de calidad

- **Portable:** funciona en Windows, macOS y Linux compatibles; la matriz exacta sigue abierta.
- **Seguro:** no sobrescribe configuración en silencio, no guarda secretos en el repositorio y gestiona explícitamente credenciales de Atomic Memory.
- **Revisable:** especificaciones, planes y cambios de memoria se presentan como diferencias Markdown cuando corresponda.
- **Incremental:** se pueden instalar y actualizar capacidades por versión sin obligar a adoptar todas las skills nuevas.
- **Eficiente:** carga instrucciones detalladas y memoria solo cuando son pertinentes; limita entradas y resultados de subagentes.
- **Diagnosticable:** orbit doctor informa agentes detectados, recursos administrados, versión y estado de integraciones.
- **Accesible:** la configuración interactiva se maneja con teclado y ofrece salida textual y opciones no interactivas.

## Señales de éxito

Evaluar la utilidad de Orbit con tareas reales del equipo. Señales candidatas:

- Un colaborador nuevo configura Orbit e inicia un flujo sin editar archivos manualmente.
- Las especificaciones identifican reglas de negocio y evidencia de aceptación antes de implementar.
- El trabajo llega en PR enfocados con criterios trazables y evidencia de validación.
- Otra sesión o colaborador recupera decisiones pertinentes sin tratar propuestas o registros reemplazados como vigentes.
- Delegar reduce investigación duplicada o contexto del agente principal, contabilizando el costo y calidad de los subagentes.
- Instalar, actualizar, revertir y desinstalar preserva skills y configuración de la persona.

Los objetivos numéricos y la forma de evaluar siguen abiertos.

## Fuera del alcance inicial

- Reemplazar Codex, Claude Code o Antigravity.
- Exigir Atomic Memory para usar Orbit.
- Crear un servicio alojado de memoria de equipo o un nuevo runtime de agentes.
- Obligar a que todo cambio siga el ciclo SDD completo.
- Exigir aprobación de todos los socios para cada tarea o PR.
- Instalar un conjunto fijo y obligatorio de subagentes en cada proyecto.

## Decisiones pendientes

1. Versión mínima de Go y matriz de sistemas operativos y arquitecturas.
2. Distribución desde GitHub: binarios publicados, go install, gestores de paquetes o combinación.
3. Comandos definitivos y flujo interactivo para separar instalación e inicialización de proyecto.
4. Convenciones de directorio por host para instalación personal y local, y resolución de conflictos de skills.
5. Formato de artefactos SDD y si los documentos de diseño son opcionales por defecto.
6. Registro de archivos administrados, detección de ediciones y resolución de conflictos de actualización.
7. Autenticación, identidad del proyecto o espacio y modalidades de conexión admitidas para Atomic Memory.
8. Capacidades de subagentes por host, mapeo de permisos y disponibilidad de métricas de tokens/costo.
9. Si Orbit ofrece una plantilla propia de PR opcional o solo un ejemplo.
10. Política de versiones, garantías de compatibilidad y migración de formatos de configuración anteriores.

Resolver cada decisión antes de implementar la capacidad que afecta; mientras tanto, se puede avanzar con trabajo independiente.
