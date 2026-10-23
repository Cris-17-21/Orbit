# Métrica de conversión de invitaciones (borrador de ejemplo)

- estado: borrador
- responsable: responsable de la tarea (por asignar)
- creada: 2026-10-22
- reglas de negocio relacionadas: ../../project-context/business-rules.md (BR-001 y BR-002 propuestos)
- decisión técnica relacionada: ../../project-context/technical-decisions.md (ADR-001 propuesto)

Este es un ejemplo trabajado, no una especificación de producto aceptada. Resolver las preguntas abiertas y obtener aceptación de la persona responsable antes de implementar.

## Necesidad

Las personas responsables de un espacio quieren saber cuántas invitaciones producen cuentas activadas para evaluar la incorporación de sus miembros.

## Objetivo

Definir y mostrar la tasa de conversión de invitaciones para un espacio y periodo seleccionados.

## Alcance

- Definir qué invitaciones forman el denominador.
- Definir qué personas invitadas cuentan como convertidas.
- Mostrar cantidad de invitaciones, conversiones y tasa para un periodo seleccionado.
- Conservar la identidad del espacio para no atribuir eventos de un cliente a otro.

## Fuera del alcance

- Elegir proveedor de BI o almacén de datos.
- Cambiar la experiencia de correo o registro.
- Definir facturación, retención u otras métricas de activación del producto.

## Reglas de negocio propuestas

Estas reglas son propuestas enlazadas a los archivos de memoria del proyecto; ninguna está aprobada todavía:

- BR-001: una invitación se convierte después de aceptarse y completarse el registro.
- BR-002: solo las personas administradoras de un espacio pueden enviar invitaciones.

## Definición de métrica (propuesta)

- **Denominador:** invitaciones válidas y distintas enviadas durante el periodo seleccionado.
- **Numerador:** invitaciones del denominador cuya persona invitada aceptó y completó el registro antes del corte del informe.
- **Tasa de conversión:** numerador dividido entre denominador, expresado como porcentaje.
- Excluir invitaciones de prueba, canceladas o inválidas si el producto puede identificarlas. La persona responsable confirma las exclusiones.
- Si el denominador está vacío, mostrar “sin datos” en vez de 0 %.

La ventana de informe, conversiones tardías, reenvíos y exclusiones exactas siguen sin resolver y se deben acordar antes de aceptar la especificación.

## Criterios de aceptación

1. Para un espacio y periodo, el informe muestra el denominador, numerador y tasa según la definición aceptada.
2. Una invitación cuenta como máximo una vez en cada cantidad, incluso si se reenvía.
3. Una invitación cuenta en el numerador solo después de aceptarse y completarse el registro.
4. Las invitaciones de otros espacios nunca aparecen en las cantidades del espacio seleccionado.
5. Si el denominador está vacío, el informe muestra el estado sin datos acordado y no presenta una tasa numérica engañosa.
6. Con los mismos datos, espacio, rango de fechas y corte aceptado, ejecuciones repetidas producen el mismo numerador, denominador y tasa.

## Preguntas abiertas para la persona responsable

- ¿Qué periodo y zona horaria debe usar la métrica?
- ¿Una aceptación posterior al periodo puede contar como conversión? ¿Qué corte corresponde?
- ¿Reenviar crea una invitación nueva o actualiza la existente?
- ¿Qué estados se excluyen y el sistema puede identificar invitaciones de prueba?
- ¿Qué significa “registro de cuenta completado” en el flujo actual?
- ¿En qué informe o pantalla del panel debe aparecer la métrica?
- ¿Qué fuente de datos, modelo de almacén y latencia de actualización se necesitan?

## Evidencia de validación para adjuntar durante la implementación

- Comprobaciones automatizadas del numerador, denominador, duplicados, aislamiento de espacios y estado sin datos.
- Revisión de consulta o modelo que demuestre que el cálculo coincide con la definición aceptada.
- Captura o vista previa del informe para un espacio y periodo representativos, si incluye UI.
