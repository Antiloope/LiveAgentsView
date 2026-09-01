---
titulo: 
slug: 
estado: borrador
creada: YYYY-MM-DD
actualizada: YYYY-MM-DD
siguiente: specify | implement | validate | ninguna
cadena: specify | specify+implement | specify+implement+validate | ninguna
---

# Spec: <titulo>

## Intento

Qué tiene que quedar verdadero cuando esto termine. Una o dos frases.

## Fuera de alcance

Qué no entra. Explícito.

## Ya decidido

Citas a `docs/03-decisions.md`, alcance, C4, visión.
Si el pedido inventa producto, no sigue: va a `docs/05-ideas-to-discuss.md` o se pregunta.

## Preguntas abiertas

- [ ] …

Vacío cuando la spec está `lista`.

## Aceptación

Lista concreta, verificable. El validador marca cada ítem.

- [ ] …

## Cómo

Notas de implementación: archivos, módulos, C4, migraciones. Las llena quien implementa.

## Validación

Las llena quien valida. Huecos, atajos, lo que divergió y si la spec se actualizó.

## Handoff

```
Spec: docs/sdd/specs/<slug>.md
Estado: <estado>
Siguiente: <specify|implement|validate|ninguna>
```
