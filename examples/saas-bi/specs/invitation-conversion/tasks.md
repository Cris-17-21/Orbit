# Tareas de la métrica de conversión de invitaciones (borrador)

- estado: borrador
- responsable: responsable de la tarea (por asignar)
- especificación: spec.md

Este desglose es una propuesta, no una autorización de implementación. Primero la persona responsable resuelve las preguntas abiertas y acepta spec.md. Mantén cada tarea en su propio PR pequeño cuando sea práctico. Si al inspeccionar el proyecto cambian dependencias o alcance, actualiza este plan antes de programar.

## Tarea 1: Definir los datos del ciclo de vida de invitación

- **Objetivo:** identificar registros o eventos existentes de invitación y registro necesarios para calcular la conversión.
- **Alcance:** inspeccionar esquemas y flujos actuales; documentar identificadores y fechas estables de invitación, espacio, envío, aceptación y registro completado. Proponer un cambio mínimo de evento o modelo solo si los datos actuales no bastan.
- **Depende de:** especificación aceptada y resolución de preguntas sobre eventos/fuente de datos.
- **Límite del PR:** solo contrato de datos o esquema; sin UI del panel.
- **Evidencia de aceptación:** revisión de esquema o contrato de eventos que confirme atribución a un solo espacio y deduplicación; notas de migración y compatibilidad si cambia el esquema.
- **Cubre:** criterios 2, 3 y 4.

## Tarea 2: Calcular la métrica de conversión

- **Objetivo:** producir numerador, denominador, tasa y comportamiento sin datos según la definición aceptada.
- **Alcance:** implementar la consulta o servicio con ventana, zona horaria, corte, reenvíos y exclusiones acordadas.
- **Depende de:** tarea 1, si establece o cambia el contrato de datos requerido.
- **Límite del PR:** cálculo y validación enfocada; sin cambios de diseño del panel.
- **Evidencia de aceptación:** pruebas automatizadas o consultas reproducibles para denominador, momento de conversión, duplicados/reenvíos, aislamiento de espacios, exclusiones, corte y denominador vacío.
- **Cubre:** criterios 1 a 5.

## Tarea 3: Mostrar la métrica

- **Objetivo:** mostrar cantidades y tasa acordadas a una persona autorizada del espacio.
- **Alcance:** agregar la superficie del informe, rango de fechas, estados de carga/error/sin datos y controles de acceso necesarios.
- **Depende de:** tarea 2 y decisión aceptada sobre ubicación del informe y roles autorizados.
- **Límite del PR:** UI del informe y su integración; evitar cambios ajenos de navegación o sistema visual.
- **Evidencia de aceptación:** verificaciones de cantidades, tasa, fechas, estados de carga/error/sin datos y autorización del espacio; adjuntar captura o vista previa cuando el proyecto lo permita.
- **Cubre:** criterios 1, 4 y 5.

## Tarea 4: Verificar el comportamiento de extremo a extremo

- **Objetivo:** confirmar que el informe publicado coincide con la especificación en todo el ciclo de invitación.
- **Alcance:** validar datos representativos desde la ingesta/modelado hasta el informe, incluido el corte aceptado.
- **Depende de:** tareas 1 a 3.
- **Límite del PR:** agregar o ajustar validación integral y notas operativas; corregir producto en otro PR si el hallazgo excede esta tarea.
- **Evidencia de aceptación:** registrar resultados para invitación convertida y no convertida, duplicado/reenvío, otro espacio y periodo vacío; confirmar estabilidad de resultados con entradas idénticas.
- **Cubre:** criterios 1 a 6.

## Dependencias

    Especificación aceptada
              |
              v
    Tarea 1: datos del ciclo de vida
              |
              v
    Tarea 2: cálculo de métrica
              |
              v
    Tarea 3: interfaz del informe
              |
              v
    Tarea 4: validación integral

Si la inspección demuestra que los datos actuales ya soportan la métrica, la tarea 1 puede convertirse en una revisión breve con evidencia y la tarea 2 avanzar sin cambio de esquema. La persona responsable actualiza el plan y mantiene acotado el PR.
