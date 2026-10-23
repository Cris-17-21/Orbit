# Especificación: inicializar memoria Markdown del proyecto con orbit init

- estado: aceptada
- responsable de la tarea: Cris (responsable de Orbit)
- creada: 2026-10-23
- aceptada por: Cris (responsable de Orbit)
- aceptada el: 2026-10-23

## Necesidad y resultado

Los equipos deben poder comenzar el flujo SDD de Orbit con memoria compartida del proyecto aunque no usen Atomic Memory. Al ejecutar orbit init se crea una estructura de memoria Markdown con plantillas claras y revisables, sin inventar reglas de negocio.

## Alcance

### Incluye

- Inicializar tres archivos de categoría dentro de project-context/ en la raíz del proyecto, siguiendo el contrato de memoria.
- Hacer la operación segura si ya existen archivos o directorios.
- Informar qué archivos se crearon, ya existían o necesitan resolución manual de conflicto.

### Excluye

- Conectar o configurar Atomic Memory, MCP o integraciones específicas de agentes.
- Generar especificaciones, planes de tareas o reglas de negocio del producto de la persona.
- Decidir la distribución final del CLI, todas sus opciones o el flujo completo de instalación interactiva.

## Contexto y reglas de negocio

| Referencia | Estado | Alcance | Relevancia |
| --- | --- | --- | --- |
| docs/memory-contract.md | diseño propuesto; dirección de trabajo aceptada para esta tarea | Memoria del proyecto Orbit | Markdown es la base; la estructura inicial contiene business-rules.md, technical-decisions.md y project-context.md. Una regla vigente necesita responsable y evidencia de aprobación. |
| docs/install-ownership.md | diseño propuesto; dirección de trabajo aceptada para esta tarea | Archivos de proyecto administrados por Orbit | Orbit no sobrescribe archivos ajenos en silencio; informa conflictos y ofrece una resolución segura. |
| docs/product-spec.md | diseño de producto propuesto; responsable aceptó en conversación el enfoque Markdown primero | CLI y memoria | orbit init inicializa flujo y memoria Markdown sin exigir Atomic Memory. |

Esta función crea solo estructura. No debe crear registros current, inventar reglas de dominio ni tratar el contenido del ejemplo como política real del equipo.

## Supuestos y decisiones

### Confirmados

- Los archivos Markdown versionados con el proyecto son la memoria compartida base; Atomic Memory es opcional.
- Los registros distinguen proposed, current y superseded. Un elemento solo pasa a current con evidencia de aprobación de la persona responsable.
- Orbit preserva archivos ajenos e informa conflictos en vez de sobrescribirlos en silencio.
- La primera versión de orbit init requiere un repositorio Git. Si se ejecuta dentro de un subdirectorio, actúa en la raíz del repositorio.

### Supuestos por verificar

- orbit init crea project-context/business-rules.md, project-context/technical-decisions.md y project-context/project-context.md en la raíz, de acuerdo con el contrato actual.
- Los archivos nuevos contienen solo encabezados breves e instrucciones con enlace al formato de registro; no agregan reglas ficticias ni preaprobadas.
- En una terminal interactiva, el comando presenta una vista previa y pregunta antes de crear archivos. --yes confirma explícitamente el mismo plan en un entorno no interactivo.
- Orbit registra los archivos que crea en .orbit/manifest.json, esquema 1, con hash de cada archivo. Un manifiesto válido puede actualizarse; uno inválido o de versión no compatible detiene el proceso antes de escribir.

### Decisiones de implementación de este incremento

- Usar la API estable Bubble Tea v1.3.10 para la confirmación interactiva. --yes funciona como confirmación explícita no interactiva.
- Fijar Go 1.24 como base del módulo de desarrollo, requerido por esa versión de Bubble Tea. La versión mínima del producto publicado queda abierta hasta definir la matriz de versiones.
- Guardar el manifiesto en .orbit/manifest.json con esquema 1. Registrar solo los archivos creados por este comando, con rutas relativas al proyecto y hashes SHA-256.
- La interfaz y las plantillas generadas se presentan en español; se conservan los nombres técnicos de archivos y opciones del CLI.

## Escenarios de usuario

### Iniciar memoria Markdown sin Atomic Memory

- Dado un proyecto Git sin archivos de memoria de Orbit
- Cuando la persona responsable ejecuta orbit init y acepta el plan de archivos
- Entonces Orbit crea los tres archivos de categoría e informa sus rutas

### Preservar contenido existente

- Dado que ya existe uno o más destinos no registrados como administrados por Orbit
- Cuando la persona ejecuta orbit init
- Entonces Orbit conserva el contenido y señala cada conflicto para revisión manual

### Repetir la inicialización de forma segura

- Dado que la estructura de memoria ya se inicializó
- Cuando la persona ejecuta orbit init de nuevo
- Entonces Orbit no duplica contenido ni reemplaza ediciones y reporta los archivos existentes

## Criterios de aceptación

- [ ] **AC-1:** En un proyecto Git válido sin directorio de destino, orbit init muestra las tres rutas y solo las crea tras la confirmación.
- [ ] **AC-2:** Cada archivo generado identifica su categoría y enlaza al formato de registro; no contiene reglas ficticias ni registros current.
- [ ] **AC-3:** Los archivos existentes que Orbit no administra quedan idénticos byte por byte y el comando informa ruta y conflicto claramente.
- [ ] **AC-4:** Repetir la inicialización no duplica plantillas ni sobrescribe ediciones posteriores.
- [ ] **AC-5:** La estructura Markdown funciona sin Atomic Memory ni servicios externos.
- [ ] **AC-6:** El resumen separa rutas creadas, existentes y en conflicto, y recomienda qué hacer ante conflictos.
- [ ] **AC-7:** El manifiesto registra cada archivo creado con ID de recurso estable y hash SHA-256; las ejecuciones posteriores reconocen archivos intactos y avisan de archivos editados sin sobrescribirlos.
- [ ] **AC-8:** Los mensajes, confirmación interactiva y plantillas generadas están en español. La confirmación acepta s y mantiene y por compatibilidad.

## Dependencias y riesgos

- La implementación depende del comando CLI y las reglas de propiedad segura descritas en docs/install-ownership.md.
- Orbit debe distinguir archivos creados por sí mismo de los que ya existían para que una inicialización o desinstalación posterior no reclame contenido ajeno.
- La ruta project-context/ puede chocar con una carpeta existente; el comando debe inspeccionarla e informar antes de escribir.

## Preguntas abiertas

- La versión mínima de Go para el producto sigue abierta. En este incremento se usa Go 1.24, mínimo declarado por Bubble Tea v1.3.10; revisar la política de soporte antes de publicar binarios.

## Evidencia de validación

| Criterio | Comprobación o evidencia | Resultado |
| --- | --- | --- |
| AC-1, AC-2, AC-5, AC-6, AC-8 | Ejecutar el CLI en un proyecto Git temporal; revisar rutas, contenidos, resumen e idioma | Aprobado en este incremento |
| AC-3, AC-4, AC-7 | Pruebas con archivos preexistentes, ediciones posteriores y archivo administrado ausente; comparar contenido y manifiesto | Aprobado en este incremento |

## Límite propuesto de tarea y PR

| Tarea | Resultado | Criterios cubiertos | Dependencias | Límite del PR |
| --- | --- | --- | --- | --- |
| T-1 | Inicializar memoria Markdown de forma segura mediante orbit init | AC-1 a AC-8 | Base del CLI y seguimiento de propiedad | Un PR para vista previa, escrituras seguras, repetición, español y verificaciones enfocadas |

## Aceptación

La persona responsable acepta esta especificación y autoriza la implementación dentro de su alcance:

- aceptada por: Cris (responsable de Orbit)
- aceptada el: 2026-10-23
- notas: la persona responsable aceptó que el repositorio Git sea un requisito; la política de soporte de Go no bloquea este incremento
