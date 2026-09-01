# Triage

Correr esto cuando el usuario pide un cambio y no nombró un paso.

## 1. ¿Ya hay spec?

Si un archivo en `docs/sdd/specs/` matchea (título, slug, archivos que toca):
cargar [implement.md](implement.md) o [validate.md](validate.md) según `siguiente`.
No ofrecer “¿arrancamos de cero?”.

## 2. ¿Es interesante?

**Sí** (recomendar spec, no codear todavía):

- Módulo, tabla, servicio o dependencia nueva (C4)
- Flujo o componente nuevo
- Comportamiento de producto, permisos, onboarding
- Más de un enfoque razonable, o el pedido es vago
- Cruza varias apps o capas, o varios docs de definición

**No** (hacerlo, sin spec):

- Typo, lint, test puntual, rename local, una línea
- El usuario dijo rápido / chico / sin spec / andá de una

En el límite, una frase: qué se va a tocar y “¿spec o de una?”.
Si no contestan y el cambio es chico, de una. Si es grande, preguntar y parar.

## 3. Recomendar

No un sermón. Tres líneas:

1. Por qué este cambio conviene spec (una razón).
2. Dos o tres preguntas que cambian el resultado (no un cuestionario).
3. Cómo seguir: “respondo y especificamos”, “especificá e implementá”,
   “hasta validar”, o “de una, sin spec”.

Usar el AskQuestion tool si está; si no, preguntar y **parar**.
Una ronda por defecto; segunda solo si la respuesta abre un hueco material.
Máximo tres preguntas por ronda.

## 4. Cadena

| Dicen | Hacer |
|---|---|
| solo el cambio, y es interesante | preguntas → specify → parar (humano) |
| “especificá e implementá” / “encadená” | specify + implement en esta sesión |
| “hasta validar” / “el flujo completo” | los tres pasos |
| “de una” / “sin spec” | implementar, no crear stub |

Encadenar no saltea escribir la spec: la escribe y sigue.
El humano puede cortar entre pasos.

## 5. Después del triage

- Specify → [specify.md](specify.md)
- Sin spec → implementar con las reglas de `AGENTS.md` (C4, no inventar producto)
- Docs crudos / destilar / decidir / estado → [docs.md](docs.md)
