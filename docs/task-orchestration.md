# Orquestación de tareas de Orbit

Este documento define el flujo propuesto para iniciar tareas con el orquestador de Orbit. Aplica al invocar Orbit desde Codex, Claude Code o Antigravity; cada adaptador expone el mismo flujo mediante las capacidades de su agente.

## Objetivos

- Convertir una solicitud en un cambio claro y acotado.
- Recuperar contexto útil sin tratar memorias propuestas o desactualizadas como reglas vigentes.
- Delegar solo trabajo independiente, acotado y útil.
- Mantener la revisión y aceptación de la persona responsable dentro de cada tarea.
- Enfocar el contexto del agente principal y medir si delegar reduce el uso total de tokens.

## Flujo de inicio

### 1. Capturar la solicitud e identificar el proyecto

El orquestador registra el resultado esperado, el área del producto afectada, las restricciones y los roles solicitados. Identifica la raíz del repositorio y busca configuración de Orbit y artefactos SDD. Si no puede identificar el proyecto, pide que la persona lo seleccione o describa.

No convierte una idea directamente en un plan de implementación. Primero resume lo que entendió y pregunta solo aquello cuya respuesta podría cambiar el alcance, el comportamiento del negocio, el significado de los datos o la aceptación.

### 2. Elegir cuánto proceso aplicar

Propone una ruta ligera para cambios pequeños y de bajo riesgo, y la ruta SDD completa para trabajo sustancial, transversal, sensible a reglas de negocio o datos, o cuando se solicita expresamente. La persona puede pedir otro nivel de estructura.

La ruta ligera identifica el alcance y cómo comprobar el resultado. La ruta completa crea o actualiza la propuesta, especificación, diseño cuando haga falta y desglose de tareas.

### 3. Recuperar contexto relevante

Lee los registros Markdown pertinentes del proyecto. Si Atomic Memory está configurado y disponible, también consulta el espacio del proyecto o equipo elegido. La búsqueda debe ser acotada: consulta conceptos del proyecto y la tarea, e incorpora solo los recuerdos necesarios para las decisiones actuales.

Conserva el ID o referencia, fuente, alcance y estado de cada recuerdo:

- current puede orientar el borrador si su alcance y fuente corresponden.
- proposed es una hipótesis que debe confirmarse, no un requisito obligatorio.
- superseded es contexto histórico y no guía cambios nuevos, salvo que la tarea trate ese historial.
- Fuentes ausentes, alcances incompatibles o conflictos entre registros vigentes se presentan a la persona responsable.
- La memoria informa el trabajo; no reemplaza la especificación aceptada ni el comportamiento observado en el código.

La memoria personal nunca se incluye en el contexto compartido del proyecto sin una petición explícita.

### 4. Explorar el repositorio

Cuando el trabajo modifica un proyecto existente, inspecciona el código pertinente, el comportamiento actual, las pruebas o mecanismos de validación y las convenciones. Ajusta la exploración al tamaño de la tarea. Si aporta valor, un subagente de exploración en modo lectura puede entregar un informe breve con rutas y referencias a líneas.

Compara la evidencia del repositorio con la memoria. Registra las discrepancias como preguntas abiertas, sin asumir que una fuente prevalece automáticamente.

### 5. Elegir especialistas y delegar

Selecciona capacidades según la necesidad: requisitos, UX/UI, frontend, backend, datos/BI, seguridad, implementación, pruebas o revisión. Son capacidades disponibles, no etapas obligatorias.

Antes de delegar, comunica brevemente qué roles recomienda y por qué. Pide instrucciones explícitas solo si la persona solicitó controlar las delegaciones o si el agente exige confirmación. La persona también puede pedir un rol directamente.

Delega cuando el subtrabajo sea independiente, acotado y tenga un resultado claro. Entrega al subagente únicamente:

- la pregunta o entregable;
- rutas o fragmentos pertinentes;
- secciones aplicables de la especificación aceptada y reglas vigentes;
- las instrucciones del rol o skill que necesita;
- permisos adecuados, de solo lectura o con escritura.

No delegues tareas breves, dependientes o sobre archivos compartidos cuando coordinar cueste más que resolverlas directamente. Evita encargos superpuestos. Pide conclusiones concisas con evidencia y preguntas pendientes. El orquestador sintetiza resultados y no copia transcripciones completas al contexto principal.

### 6. Redactar la especificación

Para un cambio sustancial, redacta una especificación que incluya:

- necesidad de usuario o negocio y resultado deseado;
- alcance y exclusiones;
- reglas de negocio con referencias a registros de memoria;
- criterios de aceptación observables y escenarios relevantes;
- definiciones de datos/BI para métricas o procesos de datos;
- dependencias, riesgos, supuestos y decisiones pendientes;
- evidencia de validación esperada.

Distingue hechos confirmados, supuestos y propuestas. Los criterios describen resultados observables y evitan imponer detalles técnicos salvo que sean un requisito.

Usa la [plantilla de especificación](spec-template.md) como punto de partida. Se pueden omitir secciones que no apliquen, pero el alcance, la aceptación y la validación deben quedar claros.

### 7. Pedir aceptación a la persona responsable

Presenta el borrador a la persona responsable de la tarea para que lo acepte o pida cambios. No inicies la implementación mientras haya decisiones materiales de negocio o condiciones de aceptación sin resolver.

Esta aceptación corresponde a la tarea. No requiere que todos los socios aprueben cada especificación. El equipo puede definir por separado qué políticas compartidas o decisiones de alto impacto necesitan un acuerdo más amplio.

Tras la aceptación, registra quién la dio y cuándo en el artefacto SDD. La aceptación autoriza trabajo dentro del alcance acordado; un cambio material de alcance o reglas vuelve a revisión de la persona responsable.

### 8. Dividir el alcance aceptado en tareas pequeñas

Solo después de aceptar la especificación, crea un desglose ordenado. Cada tarea indica un resultado, alcance, dependencias, criterios cubiertos y evidencia de validación. Mantén las tareas lo bastante pequeñas para un PR enfocado. Propón paralelismo solo cuando no compitan por archivos, interfaces o contratos de datos mutables.

### 9. Implementar una tarea a la vez

El agente implementador recibe la especificación aceptada, una tarea, sus dependencias y los criterios que debe satisfacer. Mantiene el cambio dentro del alcance y comunica los archivos modificados, decisiones tomadas y desviaciones del plan aceptado.

Si aparece una ambigüedad material, cambia una regla de negocio o hace falta ampliar el alcance, detén el trabajo afectado y presenta la evidencia y el ajuste propuesto a la persona responsable. No cambies la especificación en silencio ni agregues mejoras ajenas. La persona actualiza y vuelve a aceptar el alcance afectado antes de continuar. Se puede avanzar en paralelo en tareas independientes cuyos archivos e interfaces no entren en conflicto.

### 10. Validar contra los criterios aceptados

Vincula cada criterio aplicable con evidencia observable. Usa las comprobaciones existentes y agrega verificaciones puntuales cuando el cambio lo requiera. Registra comandos y resultados, además de los criterios que no se pudieron verificar y el motivo. Que pasen las pruebas no demuestra por sí solo que una regla de negocio o definición BI sea correcta; agrega evidencia de dominio o datos cuando corresponda.

Si falla una comprobación, informa el fallo y si bloquea la aceptación. No debilites un criterio ni elimines una comprobación solo para lograr que pase. Si la evidencia demuestra que un criterio es incorrecto o inviable, propone el ajuste a la persona responsable.

### 11. Preparar un pull request enfocado

Cada PR entrega un resultado coherente y se puede revisar por sí mismo. Prefiere una tarea por PR; combina tareas solo si forman un cambio inseparable. La descripción enlaza la especificación y la tarea aceptadas, lista los criterios abordados, resume la implementación e incluye evidencia, limitaciones conocidas y cambios de memoria propuestos. Señala explícitamente cambios de esquema, migración, permisos o definición de datos.

Deja fuera formato, limpieza y funciones especulativas que no correspondan. Si una tarea aceptada no cabe en un PR revisable, divídela y actualiza dependencias antes de seguir.

### 12. Revisar, integrar y cerrar la tarea

La persona responsable o revisora designada verifica el comportamiento, el alcance del PR, la evidencia y las reglas pertinentes. Sigue las protecciones y políticas de revisión del repositorio; Orbit no supone que puede fusionar ni omitirlas. Resuelve observaciones dentro del mismo alcance o crea una tarea posterior.

Tras fusionar o registrar otra disposición, actualiza el estado y deja un traspaso breve con el PR o commit, criterios terminados, pendientes y evidencia. Si surgió una decisión duradera, propone actualizar la [memoria del proyecto](memory-contract.md) con fuente y estado. Una propuesta no se presenta como política vigente hasta que la acepte su responsable.

### 13. Guardar el contexto de traspaso

Antes de cerrar la sesión o pasar el trabajo a otro agente, prepara un resumen breve con:

- referencias a la especificación aceptada y la tarea;
- trabajo terminado y estado actual;
- decisiones y fuentes;
- bloqueos y preguntas abiertas;
- siguiente acción y evidencia de validación pendiente.

Guarda decisiones duraderas y reutilizables en Markdown del proyecto como cambios revisables. Si Atomic Memory está disponible, propone memorias compartidas adecuadas con procedencia y estado. No guardes transcripciones completas por defecto.

## Resultado esperado antes de implementar

Para tareas sustanciales, la persona debe poder revisar:

1. Una síntesis de la solicitud y el contexto existente pertinente.
2. Conflictos o decisiones pendientes de producto, negocio o datos.
3. Roles especialistas y delegaciones propuestos, con motivos breves.
4. El borrador de especificación y sus criterios de aceptación.
5. El desglose en tareas, límites de PR y evidencia de validación.

La persona responsable acepta la especificación antes de iniciar la implementación. Los artefactos y su estado deben quedar visibles en el proyecto, no solo en la conversación.

## Indicadores de ahorro de tokens

Compara el costo de delegar con el contexto que ahorró. Cuando el agente lo permita, registra:

- cantidad y rol de las tareas delegadas;
- tokens de entrada/salida o la mejor estimación disponible;
- hallazgos duplicados o descartados;
- si la delegación cambió la especificación, el plan o la implementación;
- evidencia de tiempo y calidad resultantes.

No afirmes que los subagentes ahorran tokens solo por usar contextos separados. Delega cuando mejore el enfoque, el paralelismo o la cobertura de revisión, y ajusta la política según los resultados observados.
