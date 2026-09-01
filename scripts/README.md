# scripts/

Todo lo que hay que correr contra el proyecto está acá. Ningún comando largo de
Docker escrito de memoria en la terminal, y ninguna dependencia instalada en la
máquina más allá de Docker.

| Script | Para qué |
|---|---|
| [`up.sh`](up.sh) | **El de todos los días.** Levanta infraestructura local y aplica migraciones. |
| [`down.sh`](down.sh) | Apaga. Con `--borrar` también vacía la base. |
| [`logs.sh`](logs.sh) | Sigue los logs. Acepta el nombre de un servicio. |
| [`migrate.sh`](migrate.sh) | Aplica migraciones. `--status` y `--dry-run` para mirar sin tocar. |
| [`test.sh`](test.sh) | Lo mismo que corre el CI. |
| [`arquitectura.sh`](arquitectura.sh) | Abre el C4 en el navegador. |

[`lib.sh`](lib.sh) no se ejecuta: lo incluyen los demás.

## Las dos formas de levantar lo mismo

```
compose.yaml                       servicios base
compose.yaml + compose.dev.yaml    overrides de desarrollo  → up.sh
```

Cuando agregues apps, usá multi-stage Dockerfiles con targets `dev` y `runtime`
en el mismo archivo — el patrón de sincro.
