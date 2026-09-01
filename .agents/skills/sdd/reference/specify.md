# Specify

Escribir una spec que otro agente pueda implementar sin esta conversación.

## Preguntas

Si el pedido es vago, una ronda de 2–3 preguntas y parar, salvo cadena explícita.
No preguntar lo que ya está en `docs/03`, `docs/02`, `docs/01` o el C4.
Afirmar la lectura más probable y pedir corrección.

Preguntas que suelen cambiar el resultado:

- Qué tiene que quedar verdadero al terminar (no “qué archivos”).
- Qué queda afuera.
- Restricciones: C4, módulo dueño, no ampliar producto.
- Cadena: ¿solo spec, + implementar, + validar?

Si el pedido **es** una definición de producto no decidida: no specs de implementación.
Inbox o `docs/05-ideas-to-discuss.md`, o preguntarle a un maintainer. Ver [docs.md](docs.md).

## Archivo

1. Slug kebab-case. Path: `docs/sdd/specs/<slug>.md`.
2. Copiar [docs/sdd/templates/spec.md](../../../../../docs/sdd/templates/spec.md).
3. Llenar intento, fuera de alcance, ya decidido (citas), aceptación, preguntas que queden.
4. `estado: borrador` si hay preguntas; `lista` si se puede implementar.
5. `siguiente: implement` cuando esté `lista`; si no, `specify`.
6. Actualizar la tabla de [docs/sdd/README.md](../../../../../docs/sdd/README.md).

La spec tiene que ser autocontenida: un agente frío lee ese archivo y sabe qué hacer.
No depender del chat. Aceptación en ítems verificables, no “que quede bien”.

## Cierre

- Sin cadena: mostrar el path, las preguntas que quedan, el bloque de handoff, **parar**.
- Con `specify+implement`: marcar `lista` (el usuario ya encadenó; eso vale como ok)
  y seguir a [implement.md](implement.md).
- No codear en este paso salvo que la cadena lo pida.
