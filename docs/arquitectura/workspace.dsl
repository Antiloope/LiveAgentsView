/*
 * C4 de LiveAgentsView — modelo único del que salen los cuatro niveles.
 *
 * Esto no es documentación de lo construido: es el lugar donde se define, de
 * arriba hacia abajo, qué va a existir. El código se estructura para parecerse
 * a este archivo, no al revés. Si algo no está acá, todavía no está pensado.
 *
 * Para verlo:  ./scripts/arquitectura.sh   →  http://localhost:8081
 *
 * Estado de cada pieza: la propiedad `estado` dice si está construida
 * (`hecho`), si está esbozada (`esbozo`) o si está definida pero sin código
 * (`pendiente`).
 */

workspace "LiveAgentsView" "Proyecto open source — definición pendiente." {

    !identifiers hierarchical

    model {

        contributor = person "Contributor" "Desarrolla o documenta el proyecto."

        liveAgentsView = softwareSystem "LiveAgentsView" "Sistema principal. Alcance por definir." {
            tags "Pendiente"
            properties {
                "estado" "pendiente"
                "falta" "Definir visión, alcance y primer deployable en apps/"
            }
        }

        contributor -> liveAgentsView "Usa y contribuye a"
    }

    views {
        systemContext liveAgentsView "SystemContext" {
            include *
            autoLayout
        }

        styles {
            element "Person" {
                shape Person
            }
            element "Software System" {
                background #1168bd
                color #ffffff
            }
            element "Pendiente" {
                opacity 50
            }
        }
    }
}
