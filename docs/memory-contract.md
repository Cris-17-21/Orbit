# Contrato de memoria de Orbit

Orbit guarda el contexto duradero del proyecto en archivos Markdown versionados junto con el código. Si Atomic Memory no está configurado, estos archivos son la fuente compartida revisable. Atomic Memory es opcional y no se requiere para usar este formato.

## Límites y alcances de memoria

Orbit distingue tres tipos de memoria:

- **Memoria personal del agente:** pertenece a la cuenta o agente de una persona. Puede guardar sus preferencias de trabajo, pero no se comparte ni copia automáticamente al proyecto.
- **Memoria del proyecto:** pertenece al repositorio y es el registro compartido predeterminado. Se revisa en Git y no necesita un servicio externo.
- **Memoria del equipo o espacio de trabajo:** contexto compartido opcional almacenado en Atomic Memory cuando se ha configurado un alcance adecuado. La audiencia y los permisos dependen de ese proyecto o espacio.

El adaptador identifica el alcance consultado e informa si no puede verificar quién tiene acceso o qué modalidad de acceso está activa. Una conexión personal no equivale a memoria del equipo. Orbit no copia registros entre alcances automáticamente. Se puede proponer explícitamente promover contexto personal a memoria de proyecto o equipo; debe conservarse la fuente y el responsable, y una persona responsable debe confirmarlo.

Markdown del proyecto sigue siendo la fuente de verdad base. Atomic Memory es un mecanismo opcional de recuperación y persistencia, no una autoridad implícita. El modo de sincronización híbrida se posterga hasta diseñar resolución de conflictos, identidad y borrado. Si las fuentes discrepan, se presenta el conflicto a la persona responsable; no se elige una por fecha ni se sobrescribe la otra en silencio.

## Categorías

- **Reglas de negocio:** comportamiento del producto, restricciones y excepciones.
- **Decisiones técnicas:** dirección técnica elegida y su justificación.
- **Contexto del proyecto:** información estable para entender el proyecto.

Las especificaciones y planes de tareas propios de un cambio van en sus artefactos SDD. Este formato de memoria no reemplaza una especificación aceptada.

## Campos obligatorios de un registro

| Campo | Significado |
| --- | --- |
| type | business-rule, technical-decision o project-context |
| id | Identificador estable de la categoría, como BR-001 o ADR-001 |
| title | Nombre breve y legible |
| content | Regla, decisión o contexto que se conservará |
| scope | Proyecto, área de producto, servicio o dominio de datos al que aplica |
| status | proposed, current o superseded |
| created | Fecha de primer registro, formato ISO AAAA-MM-DD |
| updated | Fecha de última revisión sustantiva, formato ISO AAAA-MM-DD |
| source | Enlace o referencia a conversación, especificación aceptada, PR u otra evidencia |
| owner | Persona responsable de mantener el registro |
| review_after | Fecha opcional para volver a revisar antes de confiar en el registro |

Un registro current también incluye approved_by y approved_at. Un registro proposed aporta contexto, pero no es una regla activa. Un registro superseded enlaza a su reemplazo mediante superseded_by cuando exista. Para retirar un registro, se cambia su estado a superseded con una fuente explicativa y se conserva el historial.

review_after es un recordatorio, no un vencimiento automático. Cuando pasa esa fecha, Orbit marca el registro como pendiente de revisión y pide confirmación antes de usarlo como regla de negocio que controle el trabajo. Orbit no cambia el estado ni aprueba reemplazos automáticamente. La ausencia de review_after no garantiza que un registro esté vigente: se comprueba su fuente y alcance, y se pregunta cuando el comportamiento del repositorio, las definiciones de datos u otros registros vigentes entren en conflicto.

## Ciclo de aprobación y cambio

1. **Registrar:** anotar una posibilidad como proposed, con fuente, alcance y responsable. Ideas de conversaciones, inferencias de agentes y recuerdos de Atomic Memory empiezan como propuestas salvo que exista aprobación explícita.
2. **Revisar:** la persona responsable valida redacción, alcance, evidencia y trabajo afectado. La regla de negocio corresponde a la persona responsable del producto o dominio, o a quien haya delegado; la decisión técnica, a su responsable técnico; el contexto, a quien responde por su exactitud.
3. **Aceptar:** cambiar a current solo tras la confirmación de la persona responsable. Registrar approved_by y approved_at. Quien acepta una especificación puede aceptar memoria del alcance de su tarea. Las políticas de todo el equipo siguen el proceso que el equipo haya acordado; Orbit no requiere aprobación unánime de cada tarea.
4. **Revisar un cambio:** crear o editar una propuesta que cite el registro vigente. Al aceptarla, marcar el registro anterior como superseded y enlazar ambos. El cambio debe quedar revisable en el PR o en un mecanismo de auditoría de Atomic Memory cuando exista.
5. **Volver a comprobar:** si hay evidencia contradictoria, revisión pendiente, responsable desconocido o alcance ambiguo, marcarlo para revisión humana en el contexto de la tarea. No tratarlo como regla activa hasta resolverlo.

Atomic Memory puede representar aprobación, procedencia, alcance e historial de otra manera. El adaptador mapea estos campos cuando sea posible e informa qué garantías no puede ofrecer. No se debe afirmar que un elemento está aprobado, compartido, auditado o vigente solo porque fue recuperado de Atomic Memory.

## Comportamiento del agente

1. Lee registros pertinentes y cita sus IDs al usarlos en una especificación o plan.
2. Considera current el estado declarado, pero comprueba fuente y alcance. Pregunta a la persona responsable si falta la fuente, el alcance no corresponde o hay conflicto.
3. Nunca promueve una propuesta o recuerdo a current sin confirmación de la persona responsable.
4. Conserva el historial: marca como superseded los registros reemplazados y enlaza el reemplazo; no reescribe silenciosamente una decisión que ya guio trabajo.
5. Propone cambios de memoria como diferencias Markdown revisables. La persona responsable puede aceptar registros de su tarea; los cambios de política de equipo siguen el proceso acordado.
6. Mantiene separada la memoria personal. No copia preferencias personales ni información privada a la memoria compartida sin una instrucción explícita.

## Organización de archivos

Guarda los registros por categoría:

    project-context/
      business-rules.md
      technical-decisions.md
      project-context.md

Cada archivo puede contener varios registros. Mantén cada uno autocontenido y coloca los campos obligatorios debajo de su encabezado.

## Plantilla de registro

    ### <ID>: <Título>

    - type: <business-rule | technical-decision | project-context>
    - scope: <área a la que aplica>
    - status: <proposed | current | superseded>
    - created: <AAAA-MM-DD>
    - updated: <AAAA-MM-DD>
    - source: <enlace o referencia de evidencia>
    - owner: <persona responsable>
    - approved_by: <obligatorio cuando el estado es current>
    - approved_at: <obligatorio cuando el estado es current; AAAA-MM-DD>
    - review_after: <opcional; AAAA-MM-DD>
    - superseded_by: <ID de reemplazo, si corresponde>

    <Contenido>

## Ejemplo: regla de negocio

    ### BR-001: Definición de conversión de invitaciones

    - type: business-rule
    - scope: invitaciones de espacio de trabajo e informes de activación
    - status: proposed
    - created: 2026-10-22
    - updated: 2026-10-22
    - source: ejemplo ilustrativo de Orbit; no aprobado para un producto real
    - owner: responsable de producto

    Una invitación se considera convertida solo después de que la persona invitada la acepta y completa el registro de cuenta. Enviar o abrir una invitación no cuenta como conversión.

Este ejemplo sigue en estado proposed: ilustra el formato, pero no es una regla confirmada para un producto real. El proyecto debe sustituir las fuentes y responsables ilustrativos por su propia evidencia y personas responsables.
