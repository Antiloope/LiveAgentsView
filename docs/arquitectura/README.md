# Arquitectura (C4)

El modelo vive en [workspace.dsl](workspace.dsl). Es la fuente de verdad de la
estructura del sistema: de acá sale el código, no al revés.

## Ver el diagrama

```bash
./scripts/arquitectura.sh
```

Abre Structurizr Lite en `http://localhost:8081`.

## Reglas

- Si aparece un módulo, servicio o dependencia nueva, **primero** va al C4.
- Cada pieza puede llevar propiedades `estado` y `falta` para reflejar progreso.
- El CI valida que `workspace.dsl` parsea correctamente.

## Estado

Esqueleto mínimo al 2026-09-01. Pendiente definir el sistema real cuando exista
visión y alcance.
