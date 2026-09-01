# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |

## Reporting a Vulnerability

**No abras un issue público** para vulnerabilidades de seguridad.

Enviá un mail a **rodrigopizarro1234@gmail.com** con:

- Descripción del problema
- Pasos para reproducirlo
- Impacto estimado
- Versión o commit afectado (si aplica)

Intentamos responder en 72 horas. Te avisaremos cuando se publique un fix.

## Buenas prácticas en este repo

- No commitees secretos (`.env`, claves, tokens). Usá `.env.example` como plantilla.
- Las migraciones aplicadas no se editan: corregí con una migración nueva.
- Los cambios de seguridad relevantes deberían tener spec en `docs/sdd/` cuando el
  cambio no es trivial.
