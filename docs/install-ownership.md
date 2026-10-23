# Propiedad de instalación y gestión de conflictos en Orbit

- estado: propuesta de diseño
- actualizado: 2026-10-23

Este documento establece cómo el CLI de Orbit registra los archivos y entradas de configuración que administra. El objetivo es hacer repetibles las instalaciones locales al proyecto o personales, preservando las personalizaciones y skills existentes.

## Reglas de propiedad

1. Orbit solo es responsable de archivos o entradas de configuración incluidos en su manifiesto.
2. Orbit nunca reclama un archivo de configuración completo si solo necesita agregar un servidor, referencia de skill o sección administrada.
3. Orbit ubica sus recursos en un namespace propio siempre que el agente lo permita.
4. Se preservan las skills, archivos, ajustes y configuración ajenos.
5. Los conflictos se muestran; nunca se resuelven sobrescribiendo contenido en silencio.
6. Instalar, actualizar y desinstalar operan sobre un plan de cambios. El CLI lo muestra antes de aplicarlo en modo interactivo y ofrece una vista previa sin escrituras para automatización.

## Manifiesto de instalación

Cada alcance de instalación tiene un manifiesto de Orbit. Una instalación local al proyecto lo guarda dentro del proyecto; una instalación personal lo guarda en el directorio de configuración de Orbit del usuario.

El manifiesto incluye como mínimo:

| Campo | Propósito |
| --- | --- |
| schema_version | Versión del formato para migrarlo con seguridad |
| orbit_version | Versión exacta de Orbit instalada |
| scope | project o user |
| project_root | Raíz canónica en una instalación de proyecto |
| installed_at | Fecha y hora de instalación |
| resources[] | Archivos y entradas administrados por Orbit |

Cada recurso registra:

| Campo | Propósito |
| --- | --- |
| resource_id | Identidad estable de Orbit, independiente de la ruta |
| agent | Adaptador propietario o referencia del recurso |
| kind | file, config-entry o managed-section |
| target | Ruta canónica o selector estructurado de clave/sección |
| source | Ruta del paquete de Orbit y versión que lo creó |
| installed_digest | Hash de los bytes administrados o de la entrada serializada |
| previous_value | Valor anterior o marca de ausencia al agregar una entrada |
| backup_ref | Copia opcional de recuperación para un reemplazo aceptado |
| state | managed, modified, conflict u orphaned |

El manifiesto no guarda credenciales, claves API ni secretos. Si un host requiere credenciales, se usa su almacén seguro o una referencia a variable de entorno y solo se registra que la conexión está configurada.

## Flujo de instalación

1. Resolver y validar el alcance solicitado y la raíz del proyecto.
2. Obtener la versión elegida de Orbit y verificar su hash publicado.
3. Detectar agentes seleccionados y sus rutas actuales de skills/configuración sin modificarlas.
4. Crear un plan con cada archivo nuevo, entrada, conflicto e integración opcional omitida.
5. Comprobar escape de rutas, destinos duplicados, archivos existentes sin propietario y versiones de manifiesto incompatibles.
6. Mostrar el plan en la interfaz. En modo no interactivo, imprimirlo con dry-run y exigir opciones explícitas para resolver conflictos.
7. Aplicar escrituras mediante archivos temporales y reemplazo atómico cuando la plataforma lo permita. Preservar entradas ajenas y guardar recuperación antes de un reemplazo aceptado.
8. Escribir o actualizar el manifiesto solo tras aplicar y comprobar los recursos. Ante un fallo, restaurar el estado anterior usando el registro de transacción e informar si la recuperación quedó incompleta.

Por defecto, se omite el destino en conflicto y se continúa con recursos no conflictivos, informando que la integración es parcial. La persona puede elegir otro nombre con prefijo Orbit, un alcance distinto o resolverlo manualmente. Orbit no sobrescribe el destino por defecto.

## Flujo de actualización

Para cada recurso, comparar el destino actual con installed_digest:

- **Recurso administrado sin cambios:** reemplazarlo con la versión elegida y actualizar el hash.
- **Archivo o sección modificados por la persona:** preservarlos e informar el conflicto. Ofrecer una versión candidata lado a lado, una vista previa de fusión de tres vías cuando exista una base confiable, o saltar. Nunca reemplazar en silencio.
- **Recurso ausente:** ofrecer restaurarlo desde la versión elegida.
- **Recurso nuevo en la versión:** planificarlo como destino nuevo y aplicar controles normales de colisión.
- **Recurso que Orbit ya no distribuye:** marcarlo orphaned y ofrecer quitarlo; no eliminarlo automáticamente durante la actualización.

Antes de actualizar, crear un punto de recuperación con la versión anterior, el manifiesto y el contenido exacto de los recursos que cambiarán. La reversión restaura ese punto solo si el contenido actual sigue coincidiendo con lo instalado por la actualización. Si el recurso cambió después, conservarlo e informar un conflicto manual.

## Flujo de desinstalación

La desinstalación lee solo el manifiesto del alcance seleccionado y muestra una vista previa:

- Quitar archivos administrados sin cambios.
- Quitar entradas de configuración agregadas por Orbit solo si coinciden con el hash instalado; preservar otras claves y comentarios cuando el formato lo permita.
- Restaurar un valor de configuración anterior solo si el actual aún coincide con el instalado por Orbit.
- Dejar recursos modificados por la persona e indicar sus rutas para limpieza manual.
- Nunca quitar recursivamente una carpeta que contenga recursos ajenos.

Tras una limpieza correcta, quitar el manifiesto. Si quedan conflictos, conservar un manifiesto o informe pequeño de recursos pendientes hasta que se resuelvan o reconozcan.

## Edición de configuración compartida de agentes

Algunos hosts guardan servidores MCP o ajustes en archivos JSON/TOML compartidos. Orbit debe preferir el CLI del host para agregar una entrada con nombre único si preserva la configuración ajena. Si no, el adaptador modifica solo una clave namespaced o sección administrada y registra el valor exacto anterior y nuevo.

Si no es posible editar de forma acotada y verificable sin reescribir contenido ajeno, el adaptador no modifica el archivo automáticamente. Puede generar un parche para que la persona lo aplique y marcar la integración como manual en doctor.

## Recuperación e informe de auditoría

Cada instalación, actualización, reversión y desinstalación resume:

- versiones de Orbit anterior y nueva;
- archivos y entradas agregados, modificados, restaurados, omitidos o con conflicto;
- ubicación de respaldos/recuperación;
- integraciones que necesitan pasos manuales;
- siguiente comando para diagnosticar.

El CLI debe ofrecer orbit doctor en modo de solo lectura para revisar integridad del manifiesto, versión instalada, recursos ausentes o modificados y estado de integración de agentes.

## Decisiones de implementación pendientes

- Nombre exacto del manifiesto y directorio de configuración del usuario.
- Algoritmo de hash y serialización canónica de entradas estructuradas.
- Retención de puntos de recuperación y forma de eliminarlos.
- Posibilidad de renombrar un recurso entre versiones conservando resource_id.
- Qué CLI de agentes puede agregar o quitar entradas con alcance de forma segura.
- Cómo manejar archivos compartidos que un agente modifique durante una transacción de instalación.

Resolver estos detalles en la arquitectura del CLI y las especificaciones de adaptadores antes de escribir en la configuración de cada host.
