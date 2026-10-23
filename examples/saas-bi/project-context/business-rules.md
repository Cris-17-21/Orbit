# Reglas de negocio

### BR-001: Definición de conversión de invitaciones

- type: business-rule
- scope: invitaciones de espacios de trabajo e informes de activación
- status: proposed
- created: 2026-10-22
- updated: 2026-10-22
- source: ../specs/invitation-conversion/spec.md (borrador ilustrativo)
- owner: responsable de producto (rol de ejemplo)

Una invitación cuenta como convertida solo cuando la persona invitada la acepta y completa el registro de cuenta. Enviar o abrir una invitación no cuenta como conversión.

### BR-002: Elegibilidad para invitar

- type: business-rule
- scope: invitaciones de espacios de trabajo
- status: proposed
- created: 2026-10-22
- updated: 2026-10-22
- source: ../specs/invitation-conversion/spec.md (borrador ilustrativo)
- owner: responsable de producto (rol de ejemplo)

Solo una persona administradora del espacio puede enviar una invitación. La persona responsable de la tarea debe decidir si las invitaciones vencen y cómo se manejan correos duplicados o ya registrados antes de aceptar esta regla.
