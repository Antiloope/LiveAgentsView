# Decisiones

Log de decisiones tomadas, con fecha y motivo. Solo entra acá lo **acordado**.

Formato:

```
## YYYY-MM-DD — Título corto

**Quién:** …
**Decisión:** …
**Motivo:** …
```

---

## 2026-09-01 — Repo open source con estructura docs-first

**Quién:** Rodrigo
**Decisión:** El proyecto es open source bajo licencia MIT. La documentación de
definición vive en `docs/`; las specs de implementación en `docs/sdd/`. El desarrollo
local usa Docker y scripts como único punto de entrada.
**Motivo:** Replicar el flujo de trabajo probado en sincro, adaptado a un proyecto
público sin acoplar api+frontend desde el inicio.
