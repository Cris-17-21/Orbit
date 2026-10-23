# Especificación: integración inicial con Atomic Memory

- estado: aprobada
- producto: Orbit
- identificador: `orbit-atomic-memory-foundation`
- actualizado: 2026-10-23
- depende de: [contrato de memoria](../memory-contract.md), [contrato de adaptadores](../agent-adapter-contract.md)

## Objetivo

Permitir que una persona configure Atomic Memory como memoria opcional desde Orbit y que el agente seleccionado recupere contexto persistente durante el flujo de trabajo. El uso personal y el compartido deben quedar diferenciados, y Orbit debe mostrar con claridad cuándo el proveedor no permite verificar el aislamiento requerido.

La integración utiliza el servidor MCP oficial de Atomic Memory. Orbit no implementa otro backend de memoria ni reemplaza las reglas de aprobación del contrato de memoria.

## Alcance

- Incorporar una configuración guiada y una alternativa no interactiva para habilitar o deshabilitar Atomic Memory en un proyecto.
- Aceptar un servicio Atomic Memory preparado por la persona: Hosted Cloud, Connected Local o servicio autogestionado compatible, según las capacidades documentadas del proveedor. En el primer corte de Hosted Cloud, configurar solo uso personal; no declarar memoria compartida verificada.
- Configurar inicialmente Codex y Claude Code, para los que Atomic Memory publica guías MCP. Antigravity queda detrás de una validación de compatibilidad con MCP genérico.
- Distinguir la memoria personal de la memoria de un espacio de trabajo. El archivo compartido del proyecto contiene solo metadatos no secretos y la modalidad elegida; cada persona mantiene sus credenciales e identidad en su entorno local.
- Comprobar y reportar estado `deshabilitada`, `configurada`, `conectada`, `no disponible` o `no verificable`; no inferir conexión por la mera existencia de configuración.
- Actualizar la skill de entrada del orquestador para buscar contexto pertinente al iniciar tareas y tratar los resultados como referencias no confiables, no como instrucciones ni aprobaciones.
- Guardar en Atomic Memory solo resúmenes o hechos duraderos que la persona haya aceptado. Las reglas vigentes, su procedencia y aprobación permanecen registradas en la memoria Markdown del proyecto.

### Configuración versionable del proyecto

El primer modelo en `.orbit/atomic-memory.json` usa `schema_version`, `mode`, `backend` y `api_url`. `mode` admite `disabled` y `personal`; `shared` se rechaza hasta que el piloto lo valide. Para `disabled`, backend y URL se omiten. Para `hosted-cloud`, la URL predeterminada es fija y no acepta otro host. `connected-local` solo acepta localhost o una IP loopback por HTTP. `self-hosted` requiere HTTPS. Ningún campo admite credenciales, identidad de persona, query ni fragmento URL.

La modalidad `personal` selecciona el scope personal esperado por el agente; no constituye una garantía de aislamiento frente a otros usuarios con una clave de proyecto Hosted Cloud.

## Fuera del alcance

- Crear cuentas, organizaciones, proyectos, espacios o credenciales de Atomic Memory.
- Guardar secretos, tokens, correos personales o credenciales en Git, el manifiesto de Orbit o archivos versionados del proyecto.
- Garantizar que Hosted Cloud ofrece memoria de equipo entre distintas cuentas humanas cuando la documentación del proveedor no lo confirme.
- Sincronizar automáticamente los registros Markdown y Atomic Memory o resolver conflictos entre ambos.
- Instalar/administrar el runtime de Atomic Memory Core.
- Configurar Antigravity como integración oficial antes de verificar su conexión, alcance y aprobación MCP.
- Convertir recuerdos recuperados o ingeridos automáticamente en reglas de negocio vigentes.

## Supuestos y reglas

1. Atomic Memory se configura de manera optativa; el proyecto funciona con Markdown si está desactivada o no disponible.
2. Orbit invoca el servidor MCP según las convenciones del host y no envía transcripciones completas por defecto.
3. Los secretos permanecen en el almacén o configuración local del host/proveedor, nunca en los archivos del proyecto. Si un host exige una entrada manual, Orbit presenta instrucciones sin capturar ni imprimir el secreto.
4. La identidad de persona no se deriva de nombres genéricos del sistema operativo. Se configura por colaborador y el proveedor debe autenticar/autorizarlas; los valores de scope por sí solos no son autenticación.
5. El modo de espacio de trabajo solo se declara compatible después de verificarlo con dos colaboradores usando identidades independientes. Workspace y agent ID deben tener semántica definida por el backend y pasar pruebas de visibilidad.
6. Cuando el proveedor no ofrece un alcance compartido con aislamiento verificable, Orbit permite el modo personal y advierte que no proporciona memoria de equipo.
7. Atomic Memory es contexto auxiliar. La especificación SDD aceptada y la memoria Markdown aprobada son la autoridad para el trabajo del proyecto.
8. En Hosted Cloud, la clave API es de proyecto y `user_id` es un valor de la petición; no prueban identidad humana ni autorización por colaborador. El primer adaptador no expondrá modo compartido Hosted Cloud hasta que el proveedor documente y el equipo pruebe esos controles.

## Historias de usuario

- Como colaborador, quiero activar Atomic Memory para mi proyecto sin subir mis credenciales a Git.
- Como equipo, queremos compartir recuerdos aprobados entre colaboradores sin exponer recuerdos personales o privados.
- Como responsable de tarea, quiero que el orquestador use recuerdos pertinentes sin elevar propuestas antiguas a reglas vigentes.
- Como usuario, quiero que Orbit continúe con la memoria Markdown si Atomic Memory está caída o sin configurar.

## Criterios de aceptación

- **AC-1 Opt-in:** Atomic Memory está deshabilitada por defecto; la configuración informa qué archivos del proyecto y del usuario modificaría antes de aplicarlos.
- **AC-2 Credenciales:** ningún secreto de Atomic Memory queda escrito en archivos versionados, manifiestos, logs o salida normal del CLI.
- **AC-3 Identidad:** el modo compartido no usa un único usuario/API key común como sustituto de la identidad de cada colaborador. La configuración de proyecto no almacena identidad personal.
- **AC-4 Aislamiento:** en un piloto con dos identidades separadas, un recuerdo personal de A no aparece para B; un recuerdo marcado para el espacio de trabajo aparece solo si ambos tienen acceso autenticado al espacio.
- **AC-5 Fallo seguro:** si falta workspace/agent scope requerido, la identidad no se puede verificar o el servidor no está disponible, Orbit no afirma que la memoria compartida funciona; continúa con Markdown e informa el estado.
- **AC-6 Recuperación:** la skill de entrada consulta Atomic Memory solo cuando está habilitada, solicita un contexto acotado y conserva referencia/estado del recuerdo si están disponibles.
- **AC-7 Autoridad:** una regla recuperada en estado `proposed`, `replaced` o sin procedencia suficiente no se aplica como regla vigente.
- **AC-8 Escritura revisable:** Orbit no ingiere una regla de negocio como vigente hasta que la persona responsable acepte su especificación; la memoria Markdown aprobada mantiene la fuente y el responsable.
- **AC-9 Sincronización:** si Atomic Memory falla, la tarea puede continuar con Markdown; Orbit no afirma haber persistido en Atomic Memory un dato que no confirmó el servidor.
- **AC-10 Propiedad:** deshabilitar la integración quita únicamente entradas MCP y recursos registrados por Orbit; conserva configuración ajena y skills preexistentes.
- **AC-11 Compatibilidad:** Codex y Claude Code tienen una ruta de configuración y verificación documentada. Antigravity se informa como no compatible/no verificado hasta pasar la misma prueba.
- **AC-12 Diagnóstico:** `orbit doctor` distingue entre configuración presente, conectividad comprobada y alcance compartido verificado.

## Verificación

### Automatizada

- Pruebas unitarias de generación y validación de configuración por host y alcance.
- Pruebas de regresión que aseguren que los secretos no se serializan, muestran ni se incluyen en archivos versionados.
- Pruebas de preservación de configuraciones y skills preexistentes.
- `go test ./...`, `go vet ./...` y `git diff --check`.

### Piloto manual obligatorio para declarar memoria compartida

- Configurar dos entornos de colaborador con credenciales e identidades distintas en el mismo espacio de prueba.
- Comprobar ingesta/recuperación compartida y aislamiento de un recuerdo personal o `agent_only`.
- Confirmar workspace ID, caller agent ID, visibilidad y autoridad de cada registro según el backend elegido.
- Registrar proveedor, versión, pasos, estados observados y limitaciones; borrar datos sintéticos al concluir.

La prueba con dos identidades no se sustituye por una prueba unitaria ni por compartir una API key.

## Decisiones que permanecen abiertas

- Qué backend y plan de Atomic Memory usará el equipo como primer destino compartido.
- Si el modo compartido podrá usar Hosted Cloud o requerirá Open Source/Core autogestionado para satisfacer la identidad y los alcances requeridos.
- Cómo se suministran secretos a cada host de manera segura y compatible en Windows, macOS y Linux.
- Comandos finales de Orbit y archivos de configuración específicos de cada host.
- Qué referencias de origen/estado Atomic Memory devuelve por MCP y cómo mapearlas sin perder trazabilidad.

## Referencias oficiales consultadas

- [Integraciones y herramientas MCP](https://docs.atomicstrata.ai/integrations/overview)
- [Contrato de scopes e identidad](https://docs.atomicstrata.ai/sdk/concepts/scopes-and-identity)
- [Contrato de alcance de Open Source Core](https://docs.atomicstrata.ai/platform/scope)
- [CLI `am` y configuración MCP](https://docs.atomicstrata.ai/cli)
- [Integración MCP de Codex](https://docs.atomicstrata.ai/integrations/coding-agents/codex/local)
- [Integración MCP de Claude Code](https://docs.atomicstrata.ai/integrations/coding-agents/claude-code/local)
- [Consola Cloud](https://docs.atomicstrata.ai/cloud/console)
- [Proyectos Cloud](https://docs.atomicstrata.ai/cloud/how-to/projects)
- [Autenticación Cloud](https://docs.atomicstrata.ai/cloud/authentication)
- [Claves API Cloud](https://docs.atomicstrata.ai/cloud/how-to/api-keys)
- [Configuración MCP de Claude Code](https://code.claude.com/docs/en/mcp)

La documentación de Atomic Memory cambia. Volver a verificar comandos, campos de scope, secretos y compatibilidad inmediatamente antes de implementar cada adaptador.
